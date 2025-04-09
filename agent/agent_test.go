package agent

import (
	"testing"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/channel"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSelectNodes(t *testing.T) {
	ctx := t.Context()
	lndMock := lightning.NewClientMock()
	agent := agent{
		lnd:    lndMock,
		logger: logger.New(""),
		config: config.Agent{
			MaxChannelSize: 10_000_000,
		},
	}
	localNode := local.Node{
		SyncPeers: map[string]struct{}{
			"alice": {},
		},
		AllocatedBalance: 12_000_000,
		MaxOpenChannels:  2,
	}
	candidates := []nodeCandidate{
		{
			PublicKey: "alice",
			Score:     3,
		},
		{
			PublicKey: "bob",
			Addresses: []string{"localhost"},
			Score:     2,
		},
		{
			PublicKey: "carol",
			Score:     1,
		},
	}
	expectedNodes := map[string]uint64{
		"alice": localNode.AllocatedBalance / localNode.MaxOpenChannels,
		"bob":   localNode.AllocatedBalance / localNode.MaxOpenChannels,
	}

	lndMock.On("ConnectPeer", ctx, candidates[1].PublicKey, candidates[1].Addresses).Return(nil)

	nodes := agent.selectNodes(ctx, localNode, candidates)

	assert.Equal(t, expectedNodes, nodes)
}

func TestSelectChannels(t *testing.T) {
	tests := []struct {
		desc             string
		agent            agent
		localNode        local.Node
		candidates       []channelCandidate
		expectedChannels map[string]bool
	}{
		{
			desc: "Allow force closes",
			agent: agent{
				logger: logger.New(""),
				config: config.Agent{
					AllowForceCloses: true,
					HeuristicWeights: config.HeuristicsWeights{
						Close: config.DefaultCloseWeights,
					},
				},
			},
			localNode: local.Node{
				MaxCloseChannels: 2,
			},
			candidates: []channelCandidate{
				{
					ChannelPoint: "1",
					Active:       false,
					Score:        1,
				},
				{
					ChannelPoint: "2",
					Active:       true,
					Score:        1.4,
				},
				{
					ChannelPoint: "3",
					Active:       true,
					Score:        1.5,
				},
			},
			expectedChannels: map[string]bool{
				"1": true,
				"2": false,
			},
		},
		{
			desc: "Do not allow force closes",
			agent: agent{
				logger: logger.New(""),
				config: config.Agent{
					AllowForceCloses: false,
					HeuristicWeights: config.HeuristicsWeights{
						Close: config.DefaultCloseWeights,
					},
				},
			},
			localNode: local.Node{
				MaxCloseChannels: 2,
			},
			candidates: []channelCandidate{
				{
					ChannelPoint: "1",
					Active:       false,
					Score:        1,
				},
				{
					ChannelPoint: "2",
					Active:       true,
					Score:        1.4,
				},
				{
					ChannelPoint: "3",
					Active:       true,
					Score:        1.5,
				},
			},
			expectedChannels: map[string]bool{
				"2": false,
				"3": false,
			},
		},
		{
			desc: "High scored channels",
			agent: agent{
				logger: logger.New(""),
				config: config.Agent{
					AllowForceCloses: false,
					HeuristicWeights: config.HeuristicsWeights{
						Close: config.DefaultCloseWeights,
					},
				},
			},
			localNode: local.Node{
				MaxCloseChannels: 2,
			},
			candidates: []channelCandidate{
				{Score: 3},
				{Score: 3.4},
				{Score: 3.5},
			},
			expectedChannels: map[string]bool{},
		},
		{
			desc: "Do not close channels",
			agent: agent{
				logger: logger.New(""),
				config: config.Agent{
					AllowForceCloses: false,
					HeuristicWeights: config.HeuristicsWeights{
						Close: config.DefaultCloseWeights,
					},
				},
			},
			localNode: local.Node{
				MaxCloseChannels: 0,
			},
			candidates: []channelCandidate{
				{Score: 3},
				{Score: 3.4},
				{Score: 3.5},
			},
			expectedChannels: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			channels := tt.agent.selectChannels(tt.localNode, tt.candidates)

			assert.Equal(t, tt.expectedChannels, channels)
		})
	}
}

func TestUpdatePolicies(t *testing.T) {
	ctx := t.Context()
	lndMock := lightning.NewClientMock()
	agent := agent{
		lnd:    lndMock,
		logger: logger.New(""),
		config: config.Agent{
			MaxChannelSize: 10_000_000,
		},
		channelManager: channel.NewManager(config.ChannelManager{}, lndMock),
	}
	publicKey := "test"
	channelID := uint64(191315023298560)
	channelPoint := "1"
	localNode := local.Node{
		PublicKey: publicKey,
		Channels: local.Channels{
			List: []local.Channel{
				{
					ID:           channelID,
					Point:        channelPoint,
					LocalBalance: 2_463_000,
					Capacity:     5_000_000,
				},
			},
		},
	}
	forwardsResp := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: []*lnrpc.ForwardingEvent{
			{
				ChanIdIn: channelID,
				AmtIn:    30_000,
			},
			{
				ChanIdIn: channelID,
				AmtIn:    1_200_000,
			},
			{
				ChanIdOut: channelID,
				AmtOut:    520_000,
			},
			{
				ChanIdOut: channelID,
				AmtOut:    30_000,
			},
			{
				ChanIdOut: channelID,
				AmtOut:    1_200_000,
			},
		},
	}
	chanInfoResp := &lnrpc.ChannelEdge{
		ChannelId: channelID,
		Node1Pub:  publicKey,
		Node1Policy: &lnrpc.RoutingPolicy{
			FeeBaseMsat:      0,
			FeeRateMilliMsat: 100,
			MaxHtlcMsat:      4_600_000_000,
			TimeLockDelta:    80,
		},
	}
	expectedFeeRatePPM := uint64(108)
	expectedMaxHTLCMsat := uint64(1_970_400_000)

	lndMock.On("ListForwards", ctx, mock.Anything, mock.Anything, uint32(0)).Return(forwardsResp, nil)
	lndMock.On("GetChanInfo", ctx, channelID).Return(chanInfoResp, nil)
	lndMock.On("UpdateChannelPolicy",
		ctx,
		channelPoint,
		uint64(chanInfoResp.Node1Policy.FeeBaseMsat),
		expectedFeeRatePPM,
		expectedMaxHTLCMsat,
		uint64(chanInfoResp.Node1Policy.TimeLockDelta),
	).Return(nil)

	err := agent.UpdatePolicies(ctx, localNode)
	assert.NoError(t, err)
}

func TestGetChannelPolicy(t *testing.T) {
	ctx := t.Context()
	channel := local.Channel{ID: 1}
	publicKey := "test"

	tests := []struct {
		desc     string
		chanInfo *lnrpc.ChannelEdge
	}{
		{
			desc: "Node 1",
			chanInfo: &lnrpc.ChannelEdge{
				Node1Pub: publicKey,
				Node1Policy: &lnrpc.RoutingPolicy{
					TimeLockDelta:    80,
					MinHtlc:          1,
					FeeBaseMsat:      0,
					FeeRateMilliMsat: 100,
					Disabled:         false,
					MaxHtlcMsat:      1_000_000,
				},
			},
		},
		{
			desc: "Node 2",
			chanInfo: &lnrpc.ChannelEdge{
				Node2Pub: publicKey,
				Node2Policy: &lnrpc.RoutingPolicy{
					TimeLockDelta:    80,
					MinHtlc:          1,
					FeeBaseMsat:      0,
					FeeRateMilliMsat: 100,
					Disabled:         false,
					MaxHtlcMsat:      1_000_000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			lndMock := lightning.NewClientMock()

			lndMock.On("GetChanInfo", ctx, channel.ID).Return(tt.chanInfo, nil)

			policy, err := getChannelPolicy(ctx, lndMock, publicKey, channel)
			assert.NoError(t, err)

			if tt.chanInfo.Node1Pub == publicKey {
				assert.Equal(t, tt.chanInfo.Node1Policy, policy)
			} else {
				assert.Equal(t, tt.chanInfo.Node2Policy, policy)
			}
		})
	}
}

func TestCalcNewFeeRate(t *testing.T) {
	tests := []struct {
		desc               string
		channel            local.Channel
		feeRatePPM         uint64
		forwardsAmountIn   uint64
		forwardsAmountOut  uint64
		expectedFeeRatePPM uint64
	}{
		{
			desc: "Low local balance",
			channel: local.Channel{
				LocalBalance: 9,
				Capacity:     1_000,
			},
			expectedFeeRatePPM: 2_100,
		},
		{
			desc: "High local balance",
			channel: local.Channel{
				LocalBalance: 995,
				Capacity:     1_000,
			},
			expectedFeeRatePPM: 0,
		},
		{
			desc:               "No forwards",
			feeRatePPM:         50,
			forwardsAmountOut:  0,
			expectedFeeRatePPM: 45,
		},
		{
			desc:               "Very low ratio",
			feeRatePPM:         50,
			forwardsAmountOut:  1,
			forwardsAmountIn:   1000,
			expectedFeeRatePPM: 26,
		},
		{
			desc:               "Low ratio",
			feeRatePPM:         50,
			forwardsAmountIn:   1000,
			forwardsAmountOut:  200,
			expectedFeeRatePPM: 34,
		},
		{
			desc:               "Medium ratio",
			feeRatePPM:         50,
			forwardsAmountIn:   1000,
			forwardsAmountOut:  1000,
			expectedFeeRatePPM: 50,
		},
		{
			desc:               "High ratio",
			feeRatePPM:         50,
			forwardsAmountIn:   1000,
			forwardsAmountOut:  1700,
			expectedFeeRatePPM: 56,
		},
		{
			desc:               "Very high ratio",
			feeRatePPM:         50,
			forwardsAmountIn:   1000,
			forwardsAmountOut:  7000,
			expectedFeeRatePPM: 68,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := calcNewFeeRate(tt.channel, tt.feeRatePPM, tt.forwardsAmountIn, tt.forwardsAmountOut)
			assert.Equal(t, tt.expectedFeeRatePPM, result)
		})
	}
}

func TestCalcNewMaxHTLC(t *testing.T) {
	tests := []struct {
		desc           string
		channel        local.Channel
		expectedResult uint64
	}{
		{
			desc: "Low balance",
			channel: local.Channel{
				LocalBalance: 1,
			},
			expectedResult: 1_000,
		},
		{
			desc: "Medium balance",
			channel: local.Channel{
				LocalBalance: 764_000,
			},
			expectedResult: 611_200_000,
		},
		{
			desc: "High balance",
			channel: local.Channel{
				LocalBalance: 5_500_000,
			},
			expectedResult: 4_400_000_000,
		},
		{
			desc: "Very high balance",
			channel: local.Channel{
				LocalBalance: 23_000_000,
			},
			expectedResult: 18_400_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := calcNewMaxHTLC(tt.channel)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetPercentage(t *testing.T) {
	tests := []struct {
		desc           string
		value          uint64
		percent        uint64
		expectedResult uint64
	}{
		{
			desc:           "Round",
			value:          250,
			percent:        10,
			expectedResult: 25,
		},
		{
			desc:           "Round 2",
			value:          1200,
			percent:        25,
			expectedResult: 300,
		},
		{
			desc:           "Imprecise",
			value:          256,
			percent:        10,
			expectedResult: 25,
		},
		{
			desc:           "Imprecise 2",
			value:          2048,
			percent:        80,
			expectedResult: 1638,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := getPercentage(tt.value, tt.percent)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestSkipOpen(t *testing.T) {
	tests := []struct {
		desc      string
		config    config.Agent
		localNode local.Node
		skip      bool
	}{
		{
			desc: "No channels to open",
			localNode: local.Node{
				MaxOpenChannels: 0,
			},
			skip: true,
		},
		{
			desc: "Zero balance",
			config: config.Agent{
				MinChannelSize: 0,
			},
			localNode: local.Node{
				AllocatedBalance: 0,
				MaxOpenChannels:  1,
			},
			skip: true,
		},
		{
			desc: "Low balance",
			config: config.Agent{
				MinChannelSize: 200,
			},
			localNode: local.Node{
				AllocatedBalance: 100,
				MaxOpenChannels:  1,
			},
			skip: true,
		},
		{
			desc: "Too many channels",
			config: config.Agent{
				MinChannelSize: 200,
				MaxChannels:    2,
			},
			localNode: local.Node{
				AllocatedBalance: 300,
				NumChannels:      5,
				MaxOpenChannels:  1,
			},
			skip: true,
		},
		{
			desc: "Small batch size",
			config: config.Agent{
				MinChannelSize: 200,
				MaxChannels:    10,
				MinBatchSize:   6,
			},
			localNode: local.Node{
				AllocatedBalance: 300,
				NumChannels:      5,
				MaxOpenChannels:  3,
			},
			skip: true,
		},
		{
			desc: "No skip",
			config: config.Agent{
				MinChannelSize: 200,
				MaxChannels:    10,
				MinBatchSize:   2,
			},
			localNode: local.Node{
				AllocatedBalance: 300,
				NumChannels:      5,
				MaxOpenChannels:  3,
			},
			skip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := skipOpen(tt.config, tt.localNode)
			if tt.skip {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func getNode(t *testing.T, lndMock *lightning.ClientMock, config config.Agent, satvB uint64) {
	t.Helper()

	ctx := t.Context()

	infoResp := &lnrpc.GetInfoResponse{
		IdentityPubkey:      "test",
		SyncedToGraph:       true,
		NumActiveChannels:   5,
		NumPendingChannels:  2,
		NumInactiveChannels: 1,
		BlockHeight:         200,
	}
	walletResp := &lnrpc.WalletBalanceResponse{
		ConfirmedBalance: 1_000_000,
	}

	channelsResp := []*lnrpc.Channel{}
	peersResp := []*lnrpc.Peer{}
	closedChannelsResp := []*lnrpc.ChannelCloseSummary{}
	feeResp := satvB
	forwardsResp := &lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: []*lnrpc.ForwardingEvent{
			{
				ChanIdIn:  191315023298560,
				AmtInMsat: 500,
				FeeMsat:   10,
			},
		},
	}

	lndMock.On("GetInfo", ctx).Return(infoResp, nil)
	lndMock.On("WalletBalance", ctx, config.ChannelManager.MinConf).Return(walletResp, nil)
	lndMock.On("ListChannels", ctx).Return(channelsResp, nil)
	lndMock.On("ListPeers", ctx).Return(peersResp, nil)
	lndMock.On("ClosedChannels", ctx).Return(closedChannelsResp, nil)
	lndMock.On("EstimateTxFee", ctx, config.TargetConf).Return(feeResp, nil)
	lndMock.On("ListForwards", ctx, mock.Anything, mock.Anything, uint32(0)).Return(forwardsResp, nil)
}
