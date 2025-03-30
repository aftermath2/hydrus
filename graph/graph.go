package graph

import (
	"context"
	"encoding/binary"
	"math/big"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/pkg/errors"
)

// Graph represents the network graph from the point of view of the node.
type Graph struct {
	Heuristics Heuristics `json:"heuristics,omitzero"`
	Nodes      []Node     `json:"nodes,omitempty"`
}

// Node represents a lightning network node.
type Node struct {
	Alias       string
	PublicKey   string
	NumFeatures int
	Capacity    uint64
	Centrality  Centrality
	Addresses   []string
	Channels    []Channel
}

// Channel represents a lightning network channel.
//
// All routing policy values are in mSats.
type Channel struct {
	Point          string
	PeerPublicKey  string
	ID             uint64
	BlockHeight    uint64
	Capacity       uint64
	BaseFee        uint64
	FeeRate        uint64
	InboundBaseFee int64
	InboundFeeRate int64
	MinHTLC        uint64
	MaxHTLC        uint64
}

// Centrality contains a node's centrality values.
type Centrality struct {
	Degree      float64
	Betweenness float64
	Eigenvector uint64
	Closeness   float64
}

// New returns a new network graph from the point of view of the node.
func New(ctx context.Context, openWeights config.OpenWeights, lnd lightning.Client) (Graph, error) {
	graph, err := lnd.DescribeGraph(ctx)
	if err != nil {
		return Graph{}, errors.Wrap(err, "getting channel graph")
	}

	totalCapacity := uint64(0)
	nodesLen := len(graph.Nodes)
	channels := make(map[string][]Channel, nodesLen*2)
	skippedChannels := 0

	for _, edge := range graph.Edges {
		totalCapacity += uint64(edge.Capacity)

		// New channels may be processed by our node before they are propagated entirely.
		// Skip channels whose complete information isn't yet available to us.
		if edge.Node1Policy == nil && edge.Node2Policy == nil {
			skippedChannels++
			continue
		}

		blockHeight := GetChannelBlockHeight(edge.ChannelId)

		if !discardChannel(edge.Node1Policy) {
			channels[edge.Node1Pub] = append(channels[edge.Node1Pub], getNode1Channel(edge, blockHeight))
		}

		if !discardChannel(edge.Node2Policy) {
			channels[edge.Node2Pub] = append(channels[edge.Node2Pub], getNode2Channel(edge, blockHeight))
		}
	}

	// Fail if we skipped more than half of the network graph channels
	if skippedChannels > len(graph.Edges)/2 {
		return Graph{},
			errors.Errorf("channel graph is too incomplete to proceed, skipped %d channels", skippedChannels)
	}

	avgNodeSize := totalCapacity / uint64(nodesLen)
	totalNumChannels := len(graph.Edges)
	avgNumChannels := totalNumChannels / nodesLen

	nodes := make([]Node, 0, nodesLen)
	nodeIndices := make(map[string]int, nodesLen)

	for i, node := range graph.Nodes {
		nodeIndices[node.PubKey] = i

		capacity := uint64(0)
		for _, channel := range channels[node.PubKey] {
			capacity += channel.Capacity
		}

		// Discard nodes we know won't be ranked at the top in advance to reduce the size of the adjacency list
		if len(node.Addresses) == 0 || capacity < avgNodeSize || len(channels[node.PubKey]) < avgNumChannels {
			continue
		}

		n := Node{
			Alias:       node.Alias,
			PublicKey:   node.PubKey,
			NumFeatures: GetNumFeatures(node.Features),
			Capacity:    capacity,
			Addresses:   GetAddresses(node.Addresses),
			Channels:    channels[node.PubKey],
		}
		nodes = append(nodes, n)
	}

	// After having filtered the nodes, calculate the centrality of the remaining ones
	// We do this to avoid doing big amounts of allocations and speeding up the calculations
	adjList := newAdjacencyList(nodes, nodeIndices)
	sumDistances, betweennessCentrality := getCentrality(ctx, nodeIndices, adjList)
	eigenvectorCentrality := getEigenvectorCentrality(nodeIndices, adjList)

	// Populate nodes' centralities values
	heuristics := NewHeuristics(openWeights)
	for i := range nodes {
		index := nodeIndices[nodes[i].PublicKey]
		nodeDistances := sumDistances[index]
		closeness := 0.0

		if nodeDistances == 0 {
			// 0 means all its peers were bad and got filtered out
			closeness = 0
		} else {
			closeness = float64(len(nodes)-1) / float64(nodeDistances)
		}

		nodes[i].Centrality = Centrality{
			Degree:      float64(len(channels[nodes[i].PublicKey])) / float64(totalNumChannels),
			Betweenness: betweennessCentrality[index],
			Closeness:   closeness,
			Eigenvector: eigenvectorCentrality[index],
		}

		heuristics.Update(nodes[i])
	}

	return Graph{Nodes: nodes, Heuristics: *heuristics}, nil
}

// GetAddresses parses node addresses into a string slice.
func GetAddresses(addresses []*lnrpc.NodeAddress) []string {
	addrs := make([]string, 0, len(addresses))
	for _, address := range addresses {
		addrs = append(addrs, address.Addr)
	}

	return addrs
}

func getNode1Channel(edge *lnrpc.ChannelEdge, blockHeight uint32) Channel {
	return Channel{
		ID:             edge.ChannelId,
		BlockHeight:    uint64(blockHeight),
		Point:          edge.ChanPoint,
		PeerPublicKey:  edge.Node2Pub,
		Capacity:       uint64(edge.Capacity),
		BaseFee:        uint64(edge.Node1Policy.FeeBaseMsat),
		FeeRate:        uint64(edge.Node1Policy.FeeRateMilliMsat),
		InboundBaseFee: int64(edge.Node1Policy.InboundFeeBaseMsat),
		InboundFeeRate: int64(edge.Node1Policy.InboundFeeRateMilliMsat),
		MinHTLC:        uint64(edge.Node1Policy.MinHtlc),
		MaxHTLC:        edge.Node1Policy.MaxHtlcMsat,
	}
}

func getNode2Channel(edge *lnrpc.ChannelEdge, blockHeight uint32) Channel {
	return Channel{
		ID:             edge.ChannelId,
		BlockHeight:    uint64(blockHeight),
		Point:          edge.ChanPoint,
		PeerPublicKey:  edge.Node1Pub,
		Capacity:       uint64(edge.Capacity),
		BaseFee:        uint64(edge.Node2Policy.FeeBaseMsat),
		FeeRate:        uint64(edge.Node2Policy.FeeRateMilliMsat),
		InboundBaseFee: int64(edge.Node2Policy.InboundFeeBaseMsat),
		InboundFeeRate: int64(edge.Node2Policy.InboundFeeRateMilliMsat),
		MinHTLC:        uint64(edge.Node2Policy.MinHtlc),
		MaxHTLC:        edge.Node2Policy.MaxHtlcMsat,
	}
}

// GetNumFeatures returns the number of features supported by the node.
//
// TODO: support weighted features, valuing some more than others.
func GetNumFeatures(features map[uint32]*lnrpc.Feature) int {
	count := 0
	for _, feature := range features {
		if feature.IsKnown {
			count++
		}
	}

	return count
}

func discardChannel(routingPolicy *lnrpc.RoutingPolicy) bool {
	// Attempt to remove outliers. TODO: improve calculating z-score
	return routingPolicy == nil ||
		routingPolicy.Disabled ||
		routingPolicy.FeeRateMilliMsat > 20_000 ||
		routingPolicy.FeeBaseMsat > 100_000
}

// GetChannelBlockHeight returns the block height at which a channel has been established based on its ID.
func GetChannelBlockHeight(channelID uint64) uint32 {
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, channelID)

	blockHeight := new(big.Int).SetBytes(idBytes[0:3]).Uint64()
	return uint32(blockHeight)
}
