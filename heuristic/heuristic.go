package heuristic

import (
	"fmt"
	"math"
)

// value is a heuristic value type constraint.
type value interface {
	int | int64 | uint64 | float64
}

// Heuristic represents a network heuristic and holds its highest and lowest values.
type Heuristic[T value] struct {
	lowest        T
	highest       T
	lowerIsBetter bool
	weight        float64
}

// MarshalJSON is a workaround to avoid making fields public to use JSON marshalling.
func (h *Heuristic[T]) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf(`{"lowest": %v, "highest": %v, "weight": %.2f}`, h.lowest, h.highest, h.weight)
	return []byte(str), nil
}

// New returns a new heuristic of type T.
func New[T value](weight float64, lowerIsBetter bool) *Heuristic[T] {
	return &Heuristic[T]{
		highest:       0,
		lowest:        math.MaxInt,
		weight:        weight,
		lowerIsBetter: lowerIsBetter,
	}
}

// NewFull is like New but includes a parameter for every field in Heuristic, for those that have known values.
func NewFull[T value](lowest, highest T, weight float64, lowerIsBetter bool) *Heuristic[T] {
	return &Heuristic[T]{
		lowest:        lowest,
		highest:       highest,
		weight:        weight,
		lowerIsBetter: lowerIsBetter,
	}
}

// Update the heuristic field based on the value.
func (h *Heuristic[T]) Update(value T) {
	if h.weight == 0 {
		return
	}
	if value > h.highest {
		h.highest = value
	}
	if value < h.lowest {
		h.lowest = value
	}
}

// GetScore normalizes the value and multiplies it by the heuristic weight.
func (h *Heuristic[T]) GetScore(value T) float64 {
	if h.weight == 0 {
		return 0
	}

	if value == 0 {
		if h.lowerIsBetter {
			return 1 * h.weight
		}
		return 0
	}

	if h.highest == h.lowest {
		return 1 * h.weight
	}

	score := float64(value-h.lowest) * (1 / float64(h.highest-h.lowest))

	if h.lowerIsBetter {
		return (1 - score) * h.weight
	}

	return score * h.weight
}
