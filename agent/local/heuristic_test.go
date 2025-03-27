package local_test

import (
	"testing"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/config"

	"github.com/stretchr/testify/assert"
)

func TestHeuristicsGetScore(t *testing.T) {
	channel := local.Channel{
		Active:         true,
		Age:            200,
		Capacity:       500_000,
		NumForwards:    14,
		ForwardsAmount: 2_000,
		Fees:           10,
		PingTime:       200,
		FlapCount:      3,
	}

	tests := []struct {
		desc          string
		expectedScore float64
		heuristics    *local.Heuristics
		channel       local.Channel
	}{
		{
			desc:       "Default values",
			heuristics: local.NewHeuristics(config.DefaultCloseWeights),
			channel: local.Channel{
				Active:         true,
				Capacity:       1_000_000,
				Age:            50,
				NumForwards:    25,
				ForwardsAmount: 25_000,
				Fees:           1_500,
				PingTime:       700,
				FlapCount:      1,
			},
			expectedScore: 5.1,
		},
		{
			desc:       "Default values 2",
			heuristics: local.NewHeuristics(config.DefaultCloseWeights),
			channel: local.Channel{
				Active:         false,
				Capacity:       1_000_000,
				Age:            250,
				NumForwards:    5,
				ForwardsAmount: 200,
				Fees:           15,
				PingTime:       50,
				FlapCount:      1,
			},
			expectedScore: 2.1,
		},
		{
			desc: "Full weights",
			heuristics: local.NewHeuristics(config.CloseWeights{
				Capacity:       1,
				Active:         1,
				Age:            1,
				NumForwards:    1,
				ForwardsAmount: 1,
				Fees:           1,
				PingTime:       1,
				FlapCount:      1,
			}),
			channel: local.Channel{
				Active:         true,
				Capacity:       2_000_000,
				Age:            250,
				NumForwards:    32,
				ForwardsAmount: 5_000,
				Fees:           500,
				PingTime:       550,
				FlapCount:      1,
			},
			expectedScore: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.heuristics.Update(channel)
			tt.heuristics.Update(tt.channel)

			score := tt.heuristics.GetScore(tt.channel)

			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestHeuristicsUpdate(t *testing.T) {
	config := config.CloseWeights{
		Active:         0.5,
		Capacity:       0.3,
		NumForwards:    0.2,
		ForwardsAmount: 0.4,
		Fees:           0.1,
		PingTime:       0.8,
		Age:            0.9,
	}
	channel := local.Channel{
		Capacity:       100,
		NumForwards:    50,
		ForwardsAmount: 200,
		Fees:           30,
		PingTime:       700,
		Age:            800,
		Active:         true,
	}

	h := local.NewHeuristics(config)
	h.Update(channel)

	result := 1.0
	assert.Equal(t, result*config.Capacity, h.Capacity.GetScore(channel.Capacity))
	assert.Equal(t, result*config.NumForwards, h.NumForwards.GetScore(channel.NumForwards))
	assert.Equal(t, result*config.ForwardsAmount, h.ForwardsAmount.GetScore(channel.ForwardsAmount))
	assert.Equal(t, result*config.Fees, h.Fees.GetScore(channel.Fees))
	assert.Equal(t, result*config.PingTime, h.PingTime.GetScore(uint64(channel.PingTime)))
	assert.Equal(t, result*config.Age, h.Age.GetScore(uint64(channel.Age)))
	assert.Equal(t, result*config.Active, h.Active.GetScore(1))
}

func TestHeuristicsUpdateField(t *testing.T) {
	config := config.CloseWeights{Capacity: 0.6}
	channel1 := local.Channel{Capacity: 100}
	channel2 := local.Channel{Capacity: 200}
	channel3 := local.Channel{Capacity: 300}
	value := uint64(200)
	expectedScore := 0.3

	h := local.NewHeuristics(config)
	h.Update(channel1)
	h.Update(channel2)
	h.Update(channel3)

	actualScore := h.Capacity.GetScore(value)
	assert.Equal(t, expectedScore, actualScore)
}
