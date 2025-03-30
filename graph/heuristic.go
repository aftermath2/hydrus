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
	Capacity   *heuristic.Heuristic[uint64] `json:"capacity,omitempty"`
	Features   *heuristic.Heuristic[int]    `json:"features,omitempty"`
	Hybrid     *heuristic.Heuristic[int]    `json:"hybrid,omitempty"`
	Centrality *CentralityHeuristics        `json:"centrality,omitempty"`
	Channels   *Channels                    `json:"channels,omitempty"`
}

// CentralityHeuristics holds ratings used to decide which nodes to connect to.
type CentralityHeuristics struct {
	Degree      *heuristic.Heuristic[float64] `json:"degree,omitempty"`
	Betweenness *heuristic.Heuristic[float64] `json:"betweenness,omitempty"`
	Eigenvector *heuristic.Heuristic[uint64]  `json:"eigenvector,omitempty"`
	Closeness   *heuristic.Heuristic[float64] `json:"closeness,omitempty"`
}

// Channels heuristics used to decide which nodes to connect to.
type Channels struct {
	BaseFee        *heuristic.Heuristic[uint64] `json:"base_fee,omitempty"`
	FeeRate        *heuristic.Heuristic[uint64] `json:"fee_rate,omitempty"`
	InboundBaseFee *heuristic.Heuristic[int64]  `json:"inbound_base_fee,omitempty"`
	InboundFeeRate *heuristic.Heuristic[int64]  `json:"inbound_fee_rate,omitempty"`
	MinHTLC        *heuristic.Heuristic[uint64] `json:"min_htlc,omitempty"`
	MaxHTLC        *heuristic.Heuristic[uint64] `json:"max_htlc,omitempty"`
	BlockHeight    *heuristic.Heuristic[uint64] `json:"block_height,omitempty"`
}

// NewHeuristics returns a new Heuristics object with its values initialized and ready to be updated.
func NewHeuristics(weight config.OpenWeights) *Heuristics {
	return &Heuristics{
		Capacity: heuristic.New[uint64](weight.Capacity, false),
		Features: heuristic.New[int](weight.Features, false),
		Hybrid:   heuristic.NewFull(0, 1, weight.Hybrid, false),
		Centrality: &CentralityHeuristics{
			Degree:      heuristic.New[float64](weight.Centrality.Degree, false),
			Betweenness: heuristic.New[float64](weight.Centrality.Betweenness, false),
			Closeness:   heuristic.New[float64](weight.Centrality.Closeness, false),
			Eigenvector: heuristic.New[uint64](weight.Centrality.Eigenvector, false),
		},
		Channels: &Channels{
			BaseFee:        heuristic.New[uint64](weight.Channels.BaseFee, true),
			FeeRate:        heuristic.New[uint64](weight.Channels.FeeRate, true),
			InboundBaseFee: heuristic.New[int64](weight.Channels.InboundBaseFee, true),
			InboundFeeRate: heuristic.New[int64](weight.Channels.InboundFeeRate, true),
			MinHTLC:        heuristic.New[uint64](weight.Channels.MinHTLC, true),
			MaxHTLC:        heuristic.New[uint64](weight.Channels.MaxHTLC, false),
			BlockHeight:    heuristic.New[uint64](weight.Channels.BlockHeight, true),
		},
	}
}

// GetScore returns a node's score based on the heuristics collected.
func (h *Heuristics) GetScore(node Node) float64 {
	score := 0.0
	score += h.Capacity.GetScore(node.Capacity)
	score += h.Features.GetScore(node.NumFeatures)
	score += h.Centrality.Degree.GetScore(node.Centrality.Degree)
	score += h.Centrality.Betweenness.GetScore(node.Centrality.Betweenness)
	score += h.Centrality.Eigenvector.GetScore(node.Centrality.Eigenvector)
	score += h.Centrality.Closeness.GetScore(node.Centrality.Closeness)

	hybrid := 0
	if isHybrid(node.Addresses) {
		hybrid = 1
	}
	score += h.Hybrid.GetScore(hybrid)

	chanScore := 0.0
	for _, channel := range node.Channels {
		chanScore += h.Channels.BaseFee.GetScore(channel.BaseFee)
		chanScore += h.Channels.FeeRate.GetScore(channel.FeeRate)
		chanScore += h.Channels.InboundBaseFee.GetScore(channel.InboundBaseFee)
		chanScore += h.Channels.InboundFeeRate.GetScore(channel.InboundFeeRate)
		chanScore += h.Channels.MinHTLC.GetScore(channel.MinHTLC)
		chanScore += h.Channels.MaxHTLC.GetScore(channel.MaxHTLC)
		chanScore += h.Channels.BlockHeight.GetScore(channel.BlockHeight)
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
	h.Centrality.Degree.Update(node.Centrality.Degree)
	h.Centrality.Betweenness.Update(node.Centrality.Betweenness)
	h.Centrality.Eigenvector.Update(node.Centrality.Eigenvector)
	h.Centrality.Closeness.Update(node.Centrality.Closeness)

	for _, channel := range node.Channels {
		h.Channels.BaseFee.Update(channel.BaseFee)
		h.Channels.FeeRate.Update(channel.FeeRate)
		h.Channels.InboundBaseFee.Update(channel.InboundBaseFee)
		h.Channels.InboundFeeRate.Update(channel.InboundFeeRate)
		h.Channels.MinHTLC.Update(channel.MinHTLC)
		h.Channels.MaxHTLC.Update(channel.MaxHTLC)
		h.Channels.BlockHeight.Update(channel.BlockHeight)
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
