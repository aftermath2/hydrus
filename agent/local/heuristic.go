package local

import (
	"math"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/heuristic"
)

// Heuristics contains useful information from the local channels that can be used to decide which channels to
// close.
type Heuristics struct {
	Active         *heuristic.Heuristic[int]    `json:"active,omitempty"`
	Capacity       *heuristic.Heuristic[uint64] `json:"capacity,omitempty"`
	NumForwards    *heuristic.Heuristic[uint64] `json:"num_forwards,omitempty"`
	ForwardsAmount *heuristic.Heuristic[uint64] `json:"forwards_amount,omitempty"`
	Fees           *heuristic.Heuristic[uint64] `json:"fees,omitempty"`
	PingTime       *heuristic.Heuristic[uint64] `json:"ping_time,omitempty"`
	BlockHeight    *heuristic.Heuristic[uint64] `json:"block_height,omitempty"`
	FlapCount      *heuristic.Heuristic[uint64] `json:"flap_count,omitempty"`
}

// NewHeuristics returns a new Heuristics object with its values initialized and ready to be updated.
func NewHeuristics(weight config.CloseWeights) *Heuristics {
	return &Heuristics{
		Active:         heuristic.NewFull(0, 1, weight.Active, false),
		Capacity:       heuristic.New[uint64](weight.Capacity, false),
		NumForwards:    heuristic.New[uint64](weight.NumForwards, false),
		ForwardsAmount: heuristic.New[uint64](weight.ForwardsAmount, false),
		Fees:           heuristic.New[uint64](weight.Fees, false),
		BlockHeight:    heuristic.New[uint64](weight.BlockHeight, true),
		PingTime:       heuristic.New[uint64](weight.PingTime, true),
		FlapCount:      heuristic.New[uint64](weight.FlapCount, true),
	}
}

// GetScore returns a channel's score based on the heuristics collected.
func (h *Heuristics) GetScore(channel Channel) float64 {
	score := 0.0
	active := 0
	if channel.Active {
		active = 1
	}

	score += h.Active.GetScore(active)
	score += h.Capacity.GetScore(channel.Capacity)
	score += h.BlockHeight.GetScore(uint64(channel.BlockHeight))
	score += h.NumForwards.GetScore(channel.NumForwards)
	score += h.ForwardsAmount.GetScore(channel.ForwardsAmount)
	score += h.Fees.GetScore(channel.Fees)
	score += h.PingTime.GetScore(uint64(channel.PingTime))
	score += h.FlapCount.GetScore(uint64(channel.FlapCount))

	return math.Round(score*1000) / 1000
}

// Update heuristics based on the node values.
func (h *Heuristics) Update(channel Channel) {
	h.Capacity.Update(channel.Capacity)
	h.NumForwards.Update(channel.NumForwards)
	h.ForwardsAmount.Update(channel.ForwardsAmount)
	h.Fees.Update(channel.Fees)
	h.BlockHeight.Update(uint64(channel.BlockHeight))
	h.PingTime.Update(uint64(channel.PingTime))
	h.FlapCount.Update(uint64(channel.FlapCount))
}
