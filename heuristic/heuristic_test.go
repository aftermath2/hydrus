package heuristic_test

import (
	"testing"

	"github.com/aftermath2/hydrus/heuristic"

	"github.com/stretchr/testify/assert"
)

func TestHeuristic(t *testing.T) {
	tests := []struct {
		desc          string
		weight        float64
		lowerIsBetter bool
		values        []int
		value         int
		expectedScore float64
	}{
		{
			desc:          "Highest score",
			weight:        1,
			lowerIsBetter: false,
			values:        []int{0, 1},
			value:         1,
			expectedScore: 1,
		},
		{
			desc:          "Lowest score",
			weight:        1,
			lowerIsBetter: false,
			values:        []int{0, 1},
			value:         0,
			expectedScore: 0,
		},
		{
			desc:          "Inverted highest score",
			weight:        1,
			lowerIsBetter: true,
			values:        []int{0, 1},
			value:         0,
			expectedScore: 1,
		},
		{
			desc:          "Weight",
			weight:        0.5,
			lowerIsBetter: false,
			values:        []int{0, 1},
			value:         1,
			expectedScore: 0.5,
		},
		{
			desc:          "Weight 2",
			weight:        0.8,
			lowerIsBetter: false,
			values:        []int{50_000, 125_000, 12_000, 33_500, 79_000},
			value:         28_745,
			expectedScore: 0.11854867256637168,
		},
		{
			desc:          "Inverted weight",
			weight:        0.8,
			lowerIsBetter: true,
			values:        []int{50_000, 125_000, 12_000, 33_500, 79_000},
			value:         28_745,
			expectedScore: 0.6814513274336284,
		},
		{
			desc:          "Zero weight",
			weight:        0,
			lowerIsBetter: false,
			values:        []int{2, 6, 9},
			value:         8,
			expectedScore: 0,
		},
		{
			desc:          "Zero value",
			weight:        1,
			lowerIsBetter: false,
			values:        []int{0, 2, 6, 9},
			value:         0,
			expectedScore: 0,
		},
		{
			desc:          "Zero value 2",
			weight:        1,
			lowerIsBetter: true,
			values:        []int{0, 2, 6, 9},
			value:         0,
			expectedScore: 1,
		},
		{
			desc:          "Zero value 3",
			weight:        0.6,
			lowerIsBetter: true,
			values:        []int{0, 2, 6, 9},
			value:         0,
			expectedScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			h := heuristic.New[int](tt.weight, tt.lowerIsBetter)

			for _, value := range tt.values {
				h.Update(value)
			}

			actualScore := h.GetScore(tt.value)
			assert.Equal(t, tt.expectedScore, actualScore)
		})
	}
}
