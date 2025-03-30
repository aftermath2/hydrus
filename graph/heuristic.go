package graph

import (
	"math"
	"strings"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/heuristic"
)

// Heuristics contains useful information from the network graph that can be used to decide which nodes to
// connect to.
type Heuristics struct {
	Capacity              *heuristic.Heuristic[uint64]  `json:"capacity,omitempty"`
	Features              *heuristic.Heuristic[int]     `json:"features,omitempty"`
	Hybrid                *heuristic.Heuristic[int]     `json:"hybrid,omitempty"`
	BaseFee               *heuristic.Heuristic[uint64]  `json:"base_fee,omitempty"`
	FeeRate               *heuristic.Heuristic[uint64]  `json:"fee_rate,omitempty"`
	InboundBaseFee        *heuristic.Heuristic[int64]   `json:"inbound_base_fee,omitempty"`
	InboundFeeRate        *heuristic.Heuristic[int64]   `json:"inbound_fee_rate,omitempty"`
	MinHTLC               *heuristic.Heuristic[uint64]  `json:"min_htlc,omitempty"`
	MaxHTLC               *heuristic.Heuristic[uint64]  `json:"max_htlc,omitempty"`
	BlockHeight           *heuristic.Heuristic[uint64]  `json:"block_height,omitempty"`
	DegreeCentrality      *heuristic.Heuristic[float64] `json:"degree_centrality,omitempty"`
	BetweennessCentrality *heuristic.Heuristic[float64] `json:"betweenness_centrality,omitempty"`
	EigenvectorCentrality *heuristic.Heuristic[uint64]  `json:"eigenvector_centrality,omitempty"`
	ClosenessCentrality   *heuristic.Heuristic[float64] `json:"closeness_centrality,omitempty"`
}

// NewHeuristics returns a new Heuristics object with its values initialized and ready to be updated.
func NewHeuristics(weight config.OpenWeights) *Heuristics {
	return &Heuristics{
		Capacity:              heuristic.New[uint64](weight.Capacity, false),
		Features:              heuristic.New[int](weight.Features, false),
		Hybrid:                heuristic.NewFull(0, 1, weight.Hybrid, false),
		DegreeCentrality:      heuristic.New[float64](weight.DegreeCentrality, false),
		BetweennessCentrality: heuristic.New[float64](weight.BetweennessCentrality, false),
		ClosenessCentrality:   heuristic.New[float64](weight.ClosenessCentrality, false),
		EigenvectorCentrality: heuristic.New[uint64](weight.EigenvectorCentrality, false),
		BaseFee:               heuristic.New[uint64](weight.BaseFee, true),
		FeeRate:               heuristic.New[uint64](weight.FeeRate, true),
		InboundBaseFee:        heuristic.New[int64](weight.InboundBaseFee, true),
		InboundFeeRate:        heuristic.New[int64](weight.InboundFeeRate, true),
		MinHTLC:               heuristic.New[uint64](weight.MinHTLC, true),
		MaxHTLC:               heuristic.New[uint64](weight.MaxHTLC, false),
		BlockHeight:           heuristic.New[uint64](weight.BlockHeight, true),
	}
}

// GetScore returns a node's score based on the heuristics collected.
func (h *Heuristics) GetScore(node Node) float64 {
	score := 0.0
	score += h.Capacity.GetScore(node.Capacity)
	score += h.Features.GetScore(node.NumFeatures)
	score += h.DegreeCentrality.GetScore(node.Centrality.Degree)
	score += h.BetweennessCentrality.GetScore(node.Centrality.Betweenness)
	score += h.EigenvectorCentrality.GetScore(node.Centrality.Eigenvector)
	score += h.ClosenessCentrality.GetScore(node.Centrality.Closeness)

	hybrid := 0
	if isHybrid(node.Addresses) {
		hybrid = 1
	}
	score += h.Hybrid.GetScore(hybrid)

	chanScore := 0.0
	for _, channel := range node.Channels {
		chanScore += h.BaseFee.GetScore(channel.BaseFee)
		chanScore += h.FeeRate.GetScore(channel.FeeRate)
		chanScore += h.InboundBaseFee.GetScore(channel.InboundBaseFee)
		chanScore += h.InboundFeeRate.GetScore(channel.InboundFeeRate)
		chanScore += h.MinHTLC.GetScore(channel.MinHTLC)
		chanScore += h.MaxHTLC.GetScore(channel.MaxHTLC)
		chanScore += h.BlockHeight.GetScore(channel.BlockHeight)
	}

	// Get the node's channels score mean value
	chanScore /= float64(len(node.Channels))
	score += chanScore

	return math.Round(score*1000) / 1000
}

// Update heuristics based on the node values.
func (h *Heuristics) Update(node Node) {
	h.Capacity.Update(node.Capacity)
	h.Features.Update(node.NumFeatures)
	h.DegreeCentrality.Update(node.Centrality.Degree)
	h.BetweennessCentrality.Update(node.Centrality.Betweenness)
	h.EigenvectorCentrality.Update(node.Centrality.Eigenvector)
	h.ClosenessCentrality.Update(node.Centrality.Closeness)

	for _, channel := range node.Channels {
		h.BaseFee.Update(channel.BaseFee)
		h.FeeRate.Update(channel.FeeRate)
		h.InboundBaseFee.Update(channel.InboundBaseFee)
		h.InboundFeeRate.Update(channel.InboundFeeRate)
		h.MinHTLC.Update(channel.MinHTLC)
		h.MaxHTLC.Update(channel.MaxHTLC)
		h.BlockHeight.Update(channel.BlockHeight)
	}
}

// isHybrid returns whether the node is available on both clearnet and Tor or not.
func isHybrid(addresses []string) bool {
	hasClearnet := false
	hasTor := false

	for _, address := range addresses {
		host, _, _ := strings.Cut(address, ":")
		if strings.HasSuffix(host, ".onion") {
			hasTor = true
			continue
		}
		hasClearnet = true
	}

	return hasClearnet && hasTor
}
