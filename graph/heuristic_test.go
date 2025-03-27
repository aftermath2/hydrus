package graph

import (
	"testing"

	"github.com/aftermath2/hydrus/config"

	"github.com/stretchr/testify/assert"
)

func TestHeuristicsGetScore(t *testing.T) {
	node := Node{
		Capacity:  500_000,
		Addresses: []string{},
		Channels: []Channel{
			{
				BaseFee:        0,
				FeeRate:        200,
				InboundBaseFee: 2,
				InboundFeeRate: 200,
				MinHTLC:        1,
				MaxHTLC:        500_000,
			},
		},
		Centrality: Centrality{
			Degree:      0.2,
			Betweenness: 0.3,
			Eigenvector: 5,
			Closeness:   0.7,
		},
		NumFeatures: 5,
	}

	tests := []struct {
		desc          string
		expectedScore float64
		heuristics    *Heuristics
		node          Node
	}{
		{
			desc:       "Default values",
			heuristics: NewHeuristics(config.DefaultOpenWeights),
			node: Node{
				Capacity:  1_000_000,
				Addresses: []string{},
				Channels: []Channel{
					{
						BaseFee:        0,
						FeeRate:        100,
						InboundBaseFee: 1,
						InboundFeeRate: 100,
						MinHTLC:        1,
						MaxHTLC:        100_000,
					},
				},
				Centrality: Centrality{
					Degree:      0.6,
					Betweenness: 0.2,
					Eigenvector: 3,
					Closeness:   0.7,
				},
				NumFeatures: 12,
			},
			expectedScore: 7.4,
		},
		{
			desc:       "Default values 2",
			heuristics: NewHeuristics(config.DefaultOpenWeights),
			node: Node{
				Capacity:  3_000_000,
				Addresses: []string{},
				Channels: []Channel{
					{
						BaseFee:        0,
						FeeRate:        300,
						InboundBaseFee: 0,
						InboundFeeRate: 300,
						MinHTLC:        1,
						MaxHTLC:        300_000,
					},
				},
				Centrality: Centrality{
					Degree:      0.5,
					Betweenness: 0.5,
					Eigenvector: 2,
					Closeness:   0.5,
				},
				NumFeatures: 12,
			},
			expectedScore: 6,
		},
		{
			desc: "Full weights",
			heuristics: NewHeuristics(config.OpenWeights{
				Capacity:              1,
				Features:              1,
				Hybrid:                1,
				BaseFee:               1,
				FeeRate:               1,
				InboundBaseFee:        1,
				InboundFeeRate:        1,
				MinHTLC:               1,
				MaxHTLC:               1,
				DegreeCentrality:      1,
				BetweennessCentrality: 1,
				EigenvectorCentrality: 1,
				ClosenessCentrality:   1,
			}),
			node: Node{
				Capacity:  1_000_000,
				Addresses: []string{},
				Channels: []Channel{
					{
						BaseFee:        0,
						FeeRate:        100,
						InboundBaseFee: 1,
						InboundFeeRate: 100,
						MinHTLC:        1,
						MaxHTLC:        100_000,
					},
				},
				Centrality: Centrality{
					Degree:      0.6,
					Betweenness: 0.2,
					Eigenvector: 3,
					Closeness:   0.7,
				},
				NumFeatures: 12,
			},
			expectedScore: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.heuristics.Update(node)
			tt.heuristics.Update(tt.node)

			score := tt.heuristics.GetScore(tt.node)

			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestHeuristicsUpdate(t *testing.T) {
	config := config.OpenWeights{
		Capacity:              0.6,
		Features:              0.7,
		Hybrid:                1,
		DegreeCentrality:      0.5,
		BetweennessCentrality: 0.1,
		ClosenessCentrality:   0.9,
		EigenvectorCentrality: 1,
		BaseFee:               0.6,
		FeeRate:               0.7,
		InboundBaseFee:        0.8,
		InboundFeeRate:        0.9,
		MinHTLC:               1,
		MaxHTLC:               1,
	}
	node := Node{
		Capacity: 100,
		Channels: []Channel{
			{
				ID:             1,
				BaseFee:        0,
				FeeRate:        200,
				InboundBaseFee: 1,
				InboundFeeRate: 0,
				MinHTLC:        1,
				MaxHTLC:        1_000_000,
			},
		},
		NumFeatures: 3,
		Centrality: Centrality{
			Degree:      2.5,
			Betweenness: 3.0,
			Eigenvector: 1,
			Closeness:   1.5,
		},
	}

	h := NewHeuristics(config)
	h.Update(node)

	result := 1.0
	assert.Equal(t, result*config.Capacity, h.Capacity.GetScore(node.Capacity))
	assert.Equal(t, result*config.Features, h.Features.GetScore(node.NumFeatures))
	assert.Equal(t, result*config.DegreeCentrality, h.DegreeCentrality.GetScore(node.Centrality.Degree))
	assert.Equal(t, result*config.BetweennessCentrality, h.BetweennessCentrality.GetScore(node.Centrality.Betweenness))
	assert.Equal(t, result*config.ClosenessCentrality, h.ClosenessCentrality.GetScore(node.Centrality.Closeness))
	assert.Equal(t, result*config.EigenvectorCentrality, h.EigenvectorCentrality.GetScore(node.Centrality.Eigenvector))
	assert.Equal(t, result*config.BaseFee, h.BaseFee.GetScore(node.Channels[0].BaseFee))
	assert.Equal(t, result*config.FeeRate, h.FeeRate.GetScore(node.Channels[0].FeeRate))
	assert.Equal(t, result*config.InboundBaseFee, h.InboundBaseFee.GetScore(node.Channels[0].InboundBaseFee))
	assert.Equal(t, result*config.InboundFeeRate, h.InboundFeeRate.GetScore(node.Channels[0].InboundFeeRate))
	assert.Equal(t, result*config.MinHTLC, h.MinHTLC.GetScore(node.Channels[0].MinHTLC))
	assert.Equal(t, result*config.MaxHTLC, h.MaxHTLC.GetScore(node.Channels[0].MaxHTLC))
}

func TestHeuristicsUpdateField(t *testing.T) {
	config := config.OpenWeights{Capacity: 0.6}
	node1 := Node{Capacity: 100}
	node2 := Node{Capacity: 200}
	node3 := Node{Capacity: 300}
	value := uint64(200)
	expected := 0.3

	h := NewHeuristics(config)
	h.Update(node1)
	h.Update(node2)
	h.Update(node3)

	actual := h.Capacity.GetScore(value)
	assert.Equal(t, expected, actual)
}

func TestIsHybrid(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected bool
	}{
		{
			name:     "Empty list",
			input:    make([]string, 0),
			expected: false,
		},
		{
			name:     "Single clearnet address",
			input:    []string{"mail.example.com"},
			expected: false,
		},
		{
			name:     "Single Tor address",
			input:    []string{"on.example.onion"},
			expected: false,
		},
		{
			name:     "Both present",
			input:    []string{"mail.example.com", "on.example.onion"},
			expected: true,
		},
		{
			name:     "Multiple cases",
			input:    []string{"a.b.c", "x.y.z", "on.example.onion"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHybrid(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
