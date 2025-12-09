package agent

import (
	"testing"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func TestGetCandidateNodes(t *testing.T) {
	tests := []struct {
		desc               string
		localNode          local.Node
		graph              graph.Graph
		expectedCandidates []nodeCandidate
		blocklist          []string
	}{
		{
			desc:               "No candidates",
			expectedCandidates: []nodeCandidate{},
			localNode:          local.Node{},
			graph: graph.Graph{
				Heuristics: *graph.NewHeuristics(config.DefaultOpenWeights),
				Nodes:      []graph.Node{},
			},
			blocklist: []string{},
		},
		{
			desc: "Two candidates",
			expectedCandidates: []nodeCandidate{
				{
					PublicKey: "dave",
					Addresses: []string{},
					Score:     6.6,
				},
				{
					PublicKey: "bob",
					Addresses: []string{},
					Score:     6.367,
				},
			},
			localNode: local.Node{
				PublicKey:       "alice",
				ChannelPeers:    map[string]struct{}{},
				SyncPeers:       map[string]struct{}{},
				MaxOpenChannels: 5,
				Channels:        local.Channels{},
			},
			graph: graph.Graph{
				Heuristics: *graph.NewHeuristics(config.DefaultOpenWeights),
				Nodes: []graph.Node{
					{
						PublicKey: "alice",
						Capacity:  500_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{
								BaseFee:        0,
								FeeRate:        200,
								InboundBaseFee: 2,
								InboundFeeRate: 200,
								MinHTLC:        1,
								MaxHTLC:        500_000,
							},
						},
						Centrality: graph.Centrality{
							Degree:      0.2,
							Betweenness: 0.3,
							Eigenvector: 5,
							Closeness:   0.7,
						},
						NumFeatures: 5,
					},
					{
						PublicKey: "bob",
						Capacity:  1_000_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{
								BaseFee:        0,
								FeeRate:        100,
								InboundBaseFee: 1,
								InboundFeeRate: 100,
								MinHTLC:        1,
								MaxHTLC:        100_000,
							},
						},
						Centrality: graph.Centrality{
							Degree:      0.6,
							Betweenness: 0.2,
							Eigenvector: 3,
							Closeness:   0.7,
						},
						NumFeatures: 12,
					},
					{
						PublicKey: "carol",
						Capacity:  500_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{
								BaseFee:        0,
								FeeRate:        200,
								InboundBaseFee: 2,
								InboundFeeRate: 200,
								MinHTLC:        1,
								MaxHTLC:        500_000,
							},
						},
						Centrality: graph.Centrality{
							Degree:      0.2,
							Betweenness: 0.3,
							Eigenvector: 5,
							Closeness:   0.7,
						},
						NumFeatures: 5,
					},
					{
						PublicKey: "dave",
						Capacity:  3_000_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{
								BaseFee:        0,
								FeeRate:        300,
								InboundBaseFee: 0,
								InboundFeeRate: 300,
								MinHTLC:        1,
								MaxHTLC:        300_000,
							},
						},
						Centrality: graph.Centrality{
							Degree:      0.5,
							Betweenness: 0.5,
							Eigenvector: 2,
							Closeness:   0.5,
						},
						NumFeatures: 12,
					},
				},
			},
			blocklist: []string{"carol"},
		},
		{
			desc: "Fee rate",
			expectedCandidates: []nodeCandidate{
				{
					PublicKey: "bob",
					Addresses: []string{},
					Score:     1,
				},
				{
					PublicKey: "carol",
					Addresses: []string{},
					Score:     0.863,
				},
				{
					PublicKey: "dave",
					Addresses: []string{},
					Score:     0.725,
				},
			},
			localNode: local.Node{
				PublicKey:       "alice",
				ChannelPeers:    map[string]struct{}{},
				SyncPeers:       map[string]struct{}{},
				MaxOpenChannels: 5,
				Channels:        local.Channels{},
			},
			graph: graph.Graph{
				Heuristics: *graph.NewHeuristics(config.OpenWeights{
					Capacity: 0,
					Features: 0,
					Hybrid:   0,
					Centrality: config.CentralityWeights{
						Degree:      0,
						Betweenness: 0,
						Eigenvector: 0,
						Closeness:   0,
					},
					Channels: config.ChannelsWeights{
						BaseFee:        0,
						FeeRate:        1,
						InboundBaseFee: 0,
						InboundFeeRate: 0,
						MinHTLC:        0,
						MaxHTLC:        0,
					},
				}),
				Nodes: []graph.Node{
					{
						PublicKey: "alice",
						Capacity:  500_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{FeeRate: 200},
							{FeeRate: 2_000},
						},
					},
					{
						PublicKey: "bob",
						Capacity:  1_000_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{FeeRate: 1},
							{FeeRate: 1},
						},
					},
					{
						PublicKey: "carol",
						Capacity:  500_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{FeeRate: 50},
							{FeeRate: 500},
						},
					},
					{
						PublicKey: "dave",
						Capacity:  3_000_000,
						Addresses: []string{},
						Channels: []graph.Channel{
							{FeeRate: 100},
							{FeeRate: 1_000},
						},
					},
				},
			},
			blocklist: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, node := range tt.graph.Nodes {
				tt.graph.Heuristics.Update(node)
			}

			candidates := getCandidateNodes(logger.New(""), tt.localNode, tt.graph, tt.blocklist)

			assert.Equal(t, tt.expectedCandidates, candidates)
		})
	}
}

func TestDiscardNode(t *testing.T) {
	tests := []struct {
		desc      string
		localNode local.Node
		peerNode  graph.Node
		blocklist []string
		discard   bool
	}{
		{
			desc:      "Block list",
			blocklist: []string{"blocked"},
			peerNode:  graph.Node{PublicKey: "blocked"},
			discard:   true,
		},
		{
			desc: "Sharing channel",
			localNode: local.Node{
				ChannelPeers: map[string]struct{}{"alice": {}},
			},
			peerNode: graph.Node{
				PublicKey: "alice",
			},
			discard: true,
		},
		{
			desc: "Shared peers",
			localNode: local.Node{
				ChannelPeers: map[string]struct{}{
					"alice":  {},
					"carol":  {},
					"dave":   {},
					"erin":   {},
					"frank":  {},
					"george": {},
					"harold": {},
					"ian":    {},
					"jane":   {},
					"kate":   {},
				},
			},
			peerNode: graph.Node{
				PublicKey: "bob",
				Channels: []graph.Channel{
					{PeerPublicKey: "alice"},
					{PeerPublicKey: "carol"},
					{PeerPublicKey: "dave"},
					{PeerPublicKey: "erin"},
				},
			},
			discard: true,
		},
		{
			desc: "Recently closed channel",
			localNode: local.Node{
				CurrentBlockHeight: 10,
				ClosedChannels: []*lnrpc.ChannelCloseSummary{
					{
						RemotePubkey: "alice",
						CloseHeight:  8,
					},
				},
			},
			peerNode: graph.Node{
				PublicKey: "alice",
			},
			discard: true,
		},
		{
			desc: "Funding canceled",
			localNode: local.Node{
				CurrentBlockHeight: 10,
				ClosedChannels: []*lnrpc.ChannelCloseSummary{
					{
						RemotePubkey:  "alice",
						CloseHeight:   0,
						ChanId:        0x0000010000000000,
						CloseType:     lnrpc.ChannelCloseSummary_FUNDING_CANCELED,
						OpenInitiator: lnrpc.Initiator_INITIATOR_LOCAL,
					},
				},
			},
			peerNode: graph.Node{
				PublicKey: "alice",
			},
			discard: true,
		},
		{
			desc: "Own public key",
			localNode: local.Node{
				PublicKey: "carol",
			},
			peerNode: graph.Node{
				PublicKey: "carol",
			},
			discard: true,
		},
		{
			desc: "Do not discard",
			localNode: local.Node{
				PublicKey: "alice",
				ClosedChannels: []*lnrpc.ChannelCloseSummary{
					{
						RemotePubkey: "carol",
					},
				},
			},
			peerNode: graph.Node{
				PublicKey: "bob",
			},
			discard: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := discardNode(tt.localNode, tt.peerNode, tt.blocklist)
			if tt.discard {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetCandidateChannels(t *testing.T) {
	node := local.Node{
		MaxCloseChannels: 5,
		Channels: local.Channels{
			List: []local.Channel{
				{
					Point:          "1",
					Active:         true,
					BlockHeight:    10,
					Capacity:       5_000_000,
					NumForwards:    900,
					ForwardsAmount: 3_000_000_000,
					Fees:           10_000_000,
					PingTime:       300,
					FlapCount:      4,
				},
				{
					Point:          "2",
					Active:         true,
					BlockHeight:    21,
					Capacity:       12_000_000,
					NumForwards:    2_000,
					ForwardsAmount: 150_000_000_000,
					Fees:           50_000_000,
					PingTime:       100,
					FlapCount:      2,
				},
				{
					Point:          "3",
					Active:         true,
					BlockHeight:    100,
					Capacity:       100_000,
					NumForwards:    10,
					ForwardsAmount: 50_000_000,
					Fees:           250_000,
					PingTime:       150,
					FlapCount:      20,
				},
				{
					Point:          "4",
					Active:         false,
					BlockHeight:    350,
					Capacity:       20_000,
					NumForwards:    5,
					ForwardsAmount: 150,
					Fees:           1,
					PingTime:       20,
					FlapCount:      1,
				},
			},
			Heuristics: *local.NewHeuristics(config.DefaultCloseWeights),
		},
	}
	expectedCandidates := []channelCandidate{
		{
			ChannelPoint: node.Channels.List[3].Point,
			Active:       false,
			Score:        0.6,
		},
		{
			ChannelPoint: node.Channels.List[2].Point,
			Active:       true,
			Score:        1.666,
		},
		{
			ChannelPoint: node.Channels.List[0].Point,
			Active:       true,
			Score:        2.555,
		},
	}

	// Populate heuristics
	for _, channel := range node.Channels.List {
		node.Channels.Heuristics.Update(channel)
	}

	candidates := getCandidateChannels(logger.New(""), node, []string{node.Channels.List[1].Point})

	assert.Equal(t, expectedCandidates, candidates)
}
