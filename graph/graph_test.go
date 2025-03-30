package graph_test

import (
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/heuristic"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	expectedGraph := graph.Graph{
		Nodes: []graph.Node{
			{
				PublicKey: "alice",
				Centrality: graph.Centrality{
					Degree:      1,
					Betweenness: 0,
					Eigenvector: 1,
					Closeness:   1,
				},
				Addresses: []string{"localhost"},
				Channels: []graph.Channel{
					{
						ID:             1,
						Point:          "1",
						PeerPublicKey:  "carol",
						Capacity:       1_000_000,
						BaseFee:        1000,
						FeeRate:        100,
						InboundBaseFee: 0,
						InboundFeeRate: 0,
						MinHTLC:        1,
						MaxHTLC:        1_000_000,
					},
				},
				NumFeatures: 0,
				Capacity:    1_000_000,
			},
			{
				PublicKey: "carol",
				Centrality: graph.Centrality{
					Degree:      1,
					Betweenness: 0,
					Eigenvector: 1,
					Closeness:   1,
				},
				Addresses: []string{"127.0.0.1"},
				Channels: []graph.Channel{
					{
						ID:             1,
						Point:          "1",
						PeerPublicKey:  "alice",
						Capacity:       1_000_000,
						BaseFee:        0,
						FeeRate:        300,
						InboundBaseFee: 0,
						InboundFeeRate: 10_000,
						MinHTLC:        1,
						MaxHTLC:        500_000,
					},
				},
				NumFeatures: 1,
				Capacity:    1_000_000,
			},
		},
		Heuristics: graph.Heuristics{
			Capacity: heuristic.NewFull[uint64](1_000_000, 1_000_000, config.DefaultOpenWeights.Capacity, false),
			Features: heuristic.NewFull(0, 1, config.DefaultOpenWeights.Features, false),
			Hybrid:   heuristic.NewFull(0, 1, config.DefaultOpenWeights.Hybrid, false),
			Centrality: &graph.CentralityHeuristics{
				Degree:      heuristic.NewFull[float64](1, 1, config.DefaultOpenWeights.Centrality.Degree, false),
				Betweenness: heuristic.NewFull[float64](0, 0, config.DefaultOpenWeights.Centrality.Betweenness, false),
				Eigenvector: heuristic.NewFull[uint64](1, 1, config.DefaultOpenWeights.Centrality.Eigenvector, false),
				Closeness:   heuristic.NewFull[float64](1, 1, config.DefaultOpenWeights.Centrality.Closeness, false),
			},
			Channels: &graph.Channels{
				BaseFee:        heuristic.NewFull[uint64](0, 1000, config.DefaultOpenWeights.Channels.BaseFee, true),
				FeeRate:        heuristic.NewFull[uint64](100, 300, config.DefaultOpenWeights.Channels.FeeRate, true),
				InboundBaseFee: heuristic.NewFull[int64](0, 0, config.DefaultOpenWeights.Channels.InboundBaseFee, true),
				InboundFeeRate: heuristic.NewFull[int64](0, 10000, config.DefaultOpenWeights.Channels.InboundFeeRate, true),
				MinHTLC:        heuristic.NewFull[uint64](1, 1, config.DefaultOpenWeights.Channels.MinHTLC, true),
				MaxHTLC:        heuristic.NewFull[uint64](500_000, 1_000_000, config.DefaultOpenWeights.Channels.MaxHTLC, false),
				BlockHeight:    heuristic.NewFull[uint64](0, 0, config.DefaultOpenWeights.Channels.BlockHeight, true),
			},
		},
	}

	channelGraph := &lnrpc.ChannelGraph{
		Nodes: []*lnrpc.LightningNode{
			{
				PubKey:    "alice",
				Addresses: []*lnrpc.NodeAddress{{Addr: "localhost"}},
				Features:  map[uint32]*lnrpc.Feature{},
			},
			{
				PubKey:    "carol",
				Addresses: []*lnrpc.NodeAddress{{Addr: "127.0.0.1"}},
				Features: map[uint32]*lnrpc.Feature{
					1: {
						IsKnown: true,
					},
				},
			},
		},
		Edges: []*lnrpc.ChannelEdge{
			{
				ChannelId: 1,
				ChanPoint: "1",
				Node1Pub:  "alice",
				Node2Pub:  "carol",
				Capacity:  1_000_000,
				Node1Policy: &lnrpc.RoutingPolicy{
					MinHtlc:                 1,
					FeeBaseMsat:             1_000,
					FeeRateMilliMsat:        100,
					Disabled:                false,
					MaxHtlcMsat:             1_000_000,
					InboundFeeBaseMsat:      0,
					InboundFeeRateMilliMsat: 0,
				},
				Node2Policy: &lnrpc.RoutingPolicy{
					MinHtlc:                 1,
					FeeBaseMsat:             0,
					FeeRateMilliMsat:        300,
					Disabled:                false,
					MaxHtlcMsat:             500_000,
					InboundFeeBaseMsat:      0,
					InboundFeeRateMilliMsat: 10_000,
				},
			},
		},
	}

	lndMock := lightning.NewClientMock()
	lndMock.On("DescribeGraph", t.Context()).Return(channelGraph, nil)

	g, err := graph.New(t.Context(), config.DefaultOpenWeights, lndMock)
	assert.NoError(t, err)

	assert.Equal(t, expectedGraph, g)
}

func BenchmarkNew(b *testing.B) {
	ctx := b.Context()
	channelGraph := newGraphMock()

	lndMock := lightning.NewClientMock()
	lndMock.On("DescribeGraph", ctx).Return(channelGraph, nil)

	for b.Loop() {
		graph.New(ctx, config.DefaultOpenWeights, lndMock)
	}
}

func TestNewSkipChannels(t *testing.T) {
	channelGraph := &lnrpc.ChannelGraph{
		Edges: []*lnrpc.ChannelEdge{
			{
				ChannelId: 1,
				ChanPoint: "1",
				Node1Pub:  "alice",
				Node2Pub:  "carol",
				Capacity:  1_000_000,
			},
		},
	}

	lndMock := lightning.NewClientMock()
	lndMock.On("DescribeGraph", t.Context()).Return(channelGraph, nil)

	_, err := graph.New(t.Context(), config.DefaultOpenWeights, lndMock)
	assert.Error(t, err)
}

func TestNewDiscardNode(t *testing.T) {
	channelGraph := &lnrpc.ChannelGraph{
		Nodes: []*lnrpc.LightningNode{
			{
				PubKey:    "alice",
				Addresses: []*lnrpc.NodeAddress{},
				Features:  map[uint32]*lnrpc.Feature{},
			},
		},
	}

	lndMock := lightning.NewClientMock()
	lndMock.On("DescribeGraph", t.Context()).Return(channelGraph, nil)

	g, err := graph.New(t.Context(), config.DefaultOpenWeights, lndMock)
	assert.NoError(t, err)

	assert.Empty(t, g.Nodes)
}

func TestGetNumFeatures(t *testing.T) {
	tests := []struct {
		name     string
		features map[uint32]*lnrpc.Feature
		expected int
	}{
		{
			name:     "Empty features",
			features: make(map[uint32]*lnrpc.Feature),
			expected: 0,
		},
		{
			name: "All features known",
			features: map[uint32]*lnrpc.Feature{
				1: {IsKnown: true},
				2: {IsKnown: true},
				3: {IsKnown: true},
			},
			expected: 3,
		},
		{
			name: "No features known",
			features: map[uint32]*lnrpc.Feature{
				1: {IsKnown: false},
				2: {IsKnown: false},
				3: {IsKnown: false},
			},
			expected: 0,
		},
		{
			name: "Mixed features known and unknown",
			features: map[uint32]*lnrpc.Feature{
				1: {IsKnown: true},
				2: {IsKnown: false},
				3: {IsKnown: true},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.GetNumFeatures(tt.features)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    []*lnrpc.NodeAddress
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []*lnrpc.NodeAddress{},
			expected: []string{},
		},
		{
			name:     "Single address",
			input:    []*lnrpc.NodeAddress{{Network: "tcp", Addr: "127.0.0.1:9735"}},
			expected: []string{"127.0.0.1:9735"},
		},
		{
			name: "Multiple addresses",
			input: []*lnrpc.NodeAddress{
				{Network: "tcp", Addr: "127.0.0.1:9735"},
				{Network: "tcp", Addr: "192.168.1.1:9735"},
			},
			expected: []string{"127.0.0.1:9735", "192.168.1.1:9735"},
		},
		{
			name: "Empty address string",
			input: []*lnrpc.NodeAddress{
				{Network: "tcp", Addr: ""},
				{Network: "tcp", Addr: "test"},
			},
			expected: []string{"", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.GetAddresses(tt.input)
			assert.Equal(t, tt.expected, got)
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
			actual := graph.GetChannelBlockHeight(tt.channelID)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// Utils

var nodes = []string{
	"alice",
	"bob",
	"carol",
	"dave",
	"erin",
	"frank",
	"george",
	"harold",
}

var channels = []struct {
	node1PubKey string
	node2PubKey string
}{
	{
		node1PubKey: "alice",
		node2PubKey: "carol",
	},
	{
		node1PubKey: "alice",
		node2PubKey: "frank",
	},
	{
		node1PubKey: "carol",
		node2PubKey: "erin",
	},
	{
		node1PubKey: "bob",
		node2PubKey: "dave",
	},
	{
		node1PubKey: "dave",
		node2PubKey: "erin",
	},
	{
		node1PubKey: "dave",
		node2PubKey: "frank",
	},
	{
		node1PubKey: "erin",
		node2PubKey: "george",
	},
	{
		node1PubKey: "frank",
		node2PubKey: "harold",
	},
}

func newGraphMock() *lnrpc.ChannelGraph {
	totalNodes := 5_000
	nChannels := 10
	lNodes := make([]*lnrpc.LightningNode, 0, totalNodes)
	lChannels := make([]*lnrpc.ChannelEdge, 0, totalNodes*nChannels)

	for i := range totalNodes {
		lNodes = append(lNodes, newNode(strconv.Itoa(i)))
		for range nChannels {
			channel := newChannel(
				rand.Int(),
				strconv.Itoa(rand.IntN(totalNodes)),
				strconv.Itoa(rand.IntN(totalNodes)),
			)
			lChannels = append(lChannels, channel)
		}
	}

	return &lnrpc.ChannelGraph{Nodes: lNodes, Edges: lChannels}
}

func newNode(pubKey string) *lnrpc.LightningNode {
	return &lnrpc.LightningNode{
		LastUpdate:    0,
		PubKey:        pubKey,
		Alias:         "",
		Addresses:     []*lnrpc.NodeAddress{{Addr: "localhost:9735"}},
		Color:         "",
		Features:      map[uint32]*lnrpc.Feature{},
		CustomRecords: map[uint64][]byte{},
	}
}

func newChannel(id int, node1PubKey, node2PubKey string) *lnrpc.ChannelEdge {
	return &lnrpc.ChannelEdge{
		ChannelId:     uint64(id),
		ChanPoint:     "",
		LastUpdate:    0,
		Node1Pub:      node1PubKey,
		Node2Pub:      node2PubKey,
		Capacity:      5_000_000,
		Node1Policy:   &lnrpc.RoutingPolicy{},
		Node2Policy:   &lnrpc.RoutingPolicy{},
		CustomRecords: map[uint64][]byte{},
	}
}
