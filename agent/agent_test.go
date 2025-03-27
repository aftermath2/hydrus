package agent

import (
	"testing"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStart(t *testing.T) {
	tests := []struct {
		desc   string
		config config.Agent
	}{
		{
			desc: "Fee too high",
			config: config.Agent{
				ChannelManager: config.ChannelManager{
					MaxSatvB: 1,
				},
			},
		},
		{
			desc: "No channel changes",
			config: config.Agent{
				AllocationPercent: 100,
				MinChannels:       0,
				MaxChannels:       0,
				MinChannelSize:    1_000_000,
				TargetConf:        2,
				ChannelManager: config.ChannelManager{
					MinConf: 2,
				},
				HeuristicWeights: config.HeuristicsWeights{
					Close: config.DefaultCloseWeights,
					Open:  config.DefaultOpenWeights,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			lndMock := lightning.NewClientMock()
			getNode(t, lndMock, tt.config, 2)

			agent := New(tt.config, lndMock)

			err := agent.Start(t.Context())
			assert.NoError(t, err)
		})
	}
}

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

func TestSumWeightsClose(t *testing.T) {
	closeWeights := config.CloseWeights{
		Capacity:       1,
		Active:         0.5,
		NumForwards:    0.5,
		ForwardsAmount: 1,
		Fees:           0.5,
		Age:            1,
		PingTime:       0.5,
		FlapCount:      0,
	}
	expected := 5.0

	actual := SumWeights(closeWeights)

	assert.Equal(t, expected, actual)
}

func TestSumWeightsOpen(t *testing.T) {
	openWeights := config.OpenWeights{
		Capacity:              0.7,
		Features:              0.5,
		Hybrid:                1,
		BaseFee:               0.8,
		FeeRate:               1,
		InboundBaseFee:        0.4,
		InboundFeeRate:        0.4,
		MinHTLC:               1,
		MaxHTLC:               0.5,
		DegreeCentrality:      1,
		BetweennessCentrality: 1,
		EigenvectorCentrality: 1,
		ClosenessCentrality:   1,
	}
	expected := 10.3

	actual := SumWeights(openWeights)

	assert.Equal(t, expected, actual)
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
	lndMock.On("ListForwards", ctx, mock.Anything, uint32(0)).Return(forwardsResp, nil)
}
