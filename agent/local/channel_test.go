package local

import (
	"testing"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/heuristic"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetChannels(t *testing.T) {
	ctx := t.Context()
	lndMock := lightning.NewClientMock()

	ch1 := &lnrpc.Channel{
		ChanId:       191315023298560,
		RemotePubkey: "1",
		ChannelPoint: "e5b8ccc43b4eea6e2664a843e27d82c6d71d2885e7aef73777dd35c737c1d7bc:1",
		Active:       true,
		Capacity:     1_000_000,
	}
	ch2 := &lnrpc.Channel{
		ChanId:       152250023293560,
		RemotePubkey: "2",
		ChannelPoint: "62dbcff9fb4c46dfbb432dd2b70f87d4a1aa94cbc5267f03dca1c7da73e6516c:0",
		Active:       true,
		Capacity:     5_000_000,
	}
	ch3 := &lnrpc.Channel{
		ChanId:       210650287939605,
		RemotePubkey: "3",
		ChannelPoint: "507b5e874024f33d0f24605ae5a9096b31011152bd53e8eb3418b88c3babec5e:0",
		Active:       true,
		Capacity:     100_000,
	}

	forwardsResp := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: []*lnrpc.ForwardingEvent{
			{
				ChanIdIn:   ch1.ChanId,
				ChanIdOut:  ch2.ChanId,
				AmtInMsat:  1_000,
				AmtOutMsat: 900,
				FeeMsat:    100,
			},
			{
				ChanIdIn:   ch3.ChanId,
				ChanIdOut:  ch2.ChanId,
				AmtInMsat:  15_000,
				AmtOutMsat: 14_000,
				FeeMsat:    1_000,
			},
			{
				ChanIdIn:   ch1.ChanId,
				ChanIdOut:  ch3.ChanId,
				AmtInMsat:  5_000,
				AmtOutMsat: 4_500,
				FeeMsat:    500,
			},
		},
		LastOffsetIndex: 3,
	}
	lndMock.On("ListForwards", ctx, mock.Anything, uint32(0)).Return(forwardsResp, nil)

	weights := config.CloseWeights{
		Capacity:       0.2,
		Active:         1,
		NumForwards:    1,
		ForwardsAmount: 0.3,
		Fees:           0.9,
		PingTime:       0.5,
		FlapCount:      1,
		Age:            1,
	}
	channels := []*lnrpc.Channel{ch1, ch2, ch3, {Private: true}}
	peers := []*lnrpc.Peer{
		{
			PubKey:    ch1.RemotePubkey,
			PingTime:  1_000,
			FlapCount: 2,
		},
		{
			PubKey:    ch2.RemotePubkey,
			PingTime:  100,
			FlapCount: 1,
		},
		{
			PubKey:    ch3.RemotePubkey,
			PingTime:  4_500,
			FlapCount: 10,
		},
	}

	expectedChannels := Channels{
		List: []Channel{
			{
				ID:              ch1.ChanId,
				RemotePublicKey: ch1.RemotePubkey,
				Point:           ch1.ChannelPoint,
				Age:             174,
				Capacity:        uint64(ch1.Capacity),
				Active:          ch1.Active,
				NumForwards:     2,
				ForwardsAmount:  6_000,
				Fees:            600,
				PingTime:        peers[0].PingTime,
				FlapCount:       peers[0].FlapCount,
			},
			{
				ID:              ch2.ChanId,
				RemotePublicKey: ch2.RemotePubkey,
				Point:           ch2.ChannelPoint,
				Age:             138,
				Capacity:        uint64(ch2.Capacity),
				Active:          ch2.Active,
				NumForwards:     2,
				ForwardsAmount:  14_900,
				Fees:            1_100,
				PingTime:        peers[1].PingTime,
				FlapCount:       peers[1].FlapCount,
			},
			{
				ID:              ch3.ChanId,
				RemotePublicKey: ch3.RemotePubkey,
				Point:           ch3.ChannelPoint,
				Age:             191,
				Capacity:        uint64(ch3.Capacity),
				Active:          ch3.Active,
				NumForwards:     2,
				ForwardsAmount:  19_500,
				Fees:            1_500,
				PingTime:        peers[2].PingTime,
				FlapCount:       peers[2].FlapCount,
			},
		},
		Heuristics: Heuristics{
			Active:         heuristic.NewFull(0, 1, weights.Active, false),
			Capacity:       heuristic.NewFull(uint64(ch3.Capacity), uint64(ch2.Capacity), weights.Capacity, false),
			NumForwards:    heuristic.NewFull[uint64](2, 2, weights.NumForwards, false),
			ForwardsAmount: heuristic.NewFull[uint64](6_000, 19_500, weights.ForwardsAmount, false),
			Fees:           heuristic.NewFull[uint64](600, 1_500, weights.Fees, false),
			Age:            heuristic.NewFull[uint64](138, 191, weights.Age, true),
			PingTime:       heuristic.NewFull(uint64(peers[1].PingTime), uint64(peers[2].PingTime), weights.PingTime, true),
			FlapCount:      heuristic.NewFull(uint64(peers[1].FlapCount), uint64(peers[2].FlapCount), weights.FlapCount, true),
		},
	}

	chans, err := getChannels(ctx, lndMock, weights, channels, peers)
	assert.NoError(t, err)

	assert.Equal(t, expectedChannels, chans)
}

func TestListForwards(t *testing.T) {
	ctx := t.Context()
	lndMock := lightning.NewClientMock()
	startTime := uint64(0)
	offset := uint32(0)
	events := make([]*lnrpc.ForwardingEvent, 0)
	expectedEvents := []*lnrpc.ForwardingEvent{
		{Timestamp: 1},
		{Timestamp: 2},
	}
	resp := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: expectedEvents,
		LastOffsetIndex:  uint32(len(expectedEvents)),
	}

	lndMock.On("ListForwards", ctx, startTime, offset).Return(resp, nil)

	err := listForwards(ctx, lndMock, &events, startTime, offset)
	assert.NoError(t, err)

	assert.Equal(t, expectedEvents, events)
}

func TestListForwardsRecursion(t *testing.T) {
	ctx := t.Context()
	lndMock := lightning.NewClientMock()
	startTime := uint64(0)
	offset := uint32(0)

	events := make([]*lnrpc.ForwardingEvent, 0, lightning.MaxForwardingEvents)
	expectedEvents := make([]*lnrpc.ForwardingEvent, lightning.MaxForwardingEvents)

	for i := range lightning.MaxForwardingEvents {
		expectedEvents[i] = &lnrpc.ForwardingEvent{Timestamp: uint64(i)}
	}
	resp1 := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: expectedEvents,
		LastOffsetIndex:  uint32(len(expectedEvents)),
	}
	resp2 := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: []*lnrpc.ForwardingEvent{
			{Timestamp: 1},
			{Timestamp: 2},
		},
		LastOffsetIndex: uint32(len(expectedEvents) + 2),
	}

	lndMock.On("ListForwards", ctx, startTime, uint32(0)).Return(resp1, nil).Once()
	lndMock.On("ListForwards", ctx, startTime, uint32(lightning.MaxForwardingEvents)).Return(resp2, nil).Once()

	err := listForwards(ctx, lndMock, &events, startTime, offset)
	assert.NoError(t, err)

	assert.Equal(t, append(resp1.ForwardingEvents, resp2.ForwardingEvents...), events)
}

func TestGetForwardsInfo(t *testing.T) {
	tests := []struct {
		name                   string
		channel                *lnrpc.Channel
		events                 []*lnrpc.ForwardingEvent
		expectedNumForwards    uint64
		expectedForwardsAmount uint64
		expectedFees           uint64
	}{
		{
			name:    "Mixed forwards",
			channel: &lnrpc.Channel{ChanId: 1},
			events: []*lnrpc.ForwardingEvent{
				{
					ChanIdIn:  1,
					AmtInMsat: 50_000,
					FeeMsat:   1_300,
				},
				{
					ChanIdOut:  1,
					AmtOutMsat: 10_000,
					FeeMsat:    200,
				},
			},
			expectedNumForwards:    2,
			expectedForwardsAmount: 60_000,
			expectedFees:           1_500,
		},
		{
			name:    "Outgoing forwards",
			channel: &lnrpc.Channel{ChanId: 1},
			events: []*lnrpc.ForwardingEvent{
				{
					ChanIdOut:  1,
					AmtOutMsat: 100_000,
					FeeMsat:    2_000,
				},
				{
					ChanIdOut:  1,
					AmtOutMsat: 30_000,
					FeeMsat:    5_000,
				},
			},
			expectedNumForwards:    2,
			expectedForwardsAmount: 130_000,
			expectedFees:           7_000,
		},
		{
			name:                   "Zero forwards",
			channel:                &lnrpc.Channel{ChanId: 1},
			events:                 []*lnrpc.ForwardingEvent{},
			expectedNumForwards:    0,
			expectedForwardsAmount: 0,
			expectedFees:           0,
		},
		{
			name:    "No forwards",
			channel: &lnrpc.Channel{ChanId: 1},
			events: []*lnrpc.ForwardingEvent{
				{
					ChanIdOut:  3,
					AmtOutMsat: 100_000,
					FeeMsat:    2_000,
				},
				{
					ChanIdOut:  2,
					AmtOutMsat: 30_000,
					FeeMsat:    5_000,
				},
				{
					ChanIdIn:  4,
					AmtInMsat: 100,
					FeeMsat:   2_000,
				},
				{
					ChanIdIn:  9,
					AmtInMsat: 15_000,
					FeeMsat:   400,
				},
			},
			expectedNumForwards:    0,
			expectedForwardsAmount: 0,
			expectedFees:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numForwards, forwardsAmount, fees := getForwardsInfo(tt.channel, tt.events)

			assert.Equal(t, tt.expectedNumForwards, numForwards)
			assert.Equal(t, tt.expectedForwardsAmount, forwardsAmount)
			assert.Equal(t, tt.expectedFees, fees)
		})
	}
}

func TestGetPeerInfo(t *testing.T) {
	remotePublicKey := "test"

	tests := []struct {
		name              string
		channel           *lnrpc.Channel
		peers             []*lnrpc.Peer
		expectedPingTime  int64
		expectedFlapCount int32
	}{
		{
			name:    "No ping",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:   remotePublicKey,
				PingTime: 0,
			}},
			expectedPingTime: 0,
		},
		{
			name:    "High ping",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:   remotePublicKey,
				PingTime: 15_000,
			}},
			expectedPingTime: 15_000,
		},
		{
			name:    "Low ping",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:   remotePublicKey,
				PingTime: 1_000,
			}},
			expectedPingTime: 1_000,
		},
		{
			name:    "Negative ping",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:   remotePublicKey,
				PingTime: -1,
			}},
			expectedPingTime: 1_500,
		},
		{
			name:    "No flap count",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:    remotePublicKey,
				FlapCount: 0,
			}},
			expectedFlapCount: 0,
		},
		{
			name:    "High flap count",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:    remotePublicKey,
				FlapCount: 25,
			}},
			expectedFlapCount: 25,
		},
		{
			name:    "Low flap count",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:    remotePublicKey,
				FlapCount: 2,
			}},
			expectedFlapCount: 2,
		},
		{
			name:    "Standard",
			channel: &lnrpc.Channel{RemotePubkey: remotePublicKey},
			peers: []*lnrpc.Peer{{
				PubKey:    remotePublicKey,
				PingTime:  500,
				FlapCount: 3,
			}},
			expectedPingTime:  500,
			expectedFlapCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pingTime, flapCount := getPeerInfo(tt.channel, tt.peers)

			assert.Equal(t, tt.expectedPingTime, pingTime)
			assert.Equal(t, tt.expectedFlapCount, flapCount)
		})
	}
}

func TestGetChannelBlockHeight(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint64
		expected  uint32
		note      string
	}{
		{
			name:      "Minimum value",
			channelID: 0,
			expected:  0,
		},
		{
			name:      "Maximum uint64",
			channelID: (1 << 64) - 1,
			expected:  0xFFFFFF,
		},
		{
			name:      "Regular channel ID",
			channelID: 191315023298560,
			expected:  174,
		},
		{
			name:      "Hex channel ID",
			channelID: 0x0001110000000000,
			expected:  0x111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GetChannelBlockHeight(tt.channelID)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
