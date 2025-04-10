package graph

import (
	"context"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
)

func newAdjacencyList(graphNodes []Node, nodeIndices map[string]int) [][]int {
	adjList := make([][]int, len(nodeIndices))
	for _, n := range graphNodes {
		i := nodeIndices[n.PublicKey]

		for _, channel := range n.Channels {
			j := nodeIndices[channel.PeerPublicKey]

			adjList[i] = append(adjList[i], j)
		}
	}

	return adjList
}

// getCentrality returns the sum of the shortests paths from all nodes and their betweenness centrality score.
func getCentrality(ctx context.Context, nodeIndices map[string]int, adjList [][]int) ([]int, []float64) {
	nodesLen := len(nodeIndices)
	sumDistances := make([]int, nodesLen)
	bCentrality := make([]float64, nodesLen)

	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, index := range nodeIndices {
		g.Go(func() error {
			d, bc := getNodeCentrality(adjList, index, nodesLen)

			for _, value := range d {
				if value == -1 {
					// The node is no longer in the graph because it was filtered out
					break
				}
				sumDistances[index] += value
			}

			mu.Lock()
			for i, value := range bc {
				bCentrality[i] += value
			}
			mu.Unlock()

			return nil
		})
	}

	_ = g.Wait()

	return sumDistances, bCentrality
}

func getEigenvectorCentrality(nodeIndices map[string]int, adjList [][]int) []uint64 {
	// 4 iterations should be enough to get the values
	iterations := 4
	matrix := make([][]uint64, iterations)

	for i := range iterations {
		matrix[i] = make([]uint64, len(nodeIndices))

		for _, nodeIndex := range nodeIndices {
			if i == 0 {
				// The first value is the number of peers the node has
				matrix[i][nodeIndex] = uint64(len(adjList[nodeIndex]))
				continue
			}

			// Sum the centrality of the peer nodes
			sum := uint64(0)
			for _, peerIndex := range adjList[nodeIndex] {
				sum += matrix[i-1][peerIndex]
			}

			matrix[i][nodeIndex] = sum
		}
	}

	return matrix[len(matrix)-1]
}

// Copyright (C) 2015-2022 Lightning Labs and The Lightning Network Developers

// stack is a simple int stack to help with readability of Brandes'
// betweenness centrality implementation below.
type stack struct {
	stack []int
}

func newStack(length int) stack {
	return stack{stack: make([]int, 0, length)}
}

func (s *stack) push(v int) {
	s.stack = append(s.stack, v)
}

func (s *stack) pop() int {
	v := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return v
}

func (s *stack) empty() bool {
	return len(s.stack) == 0
}

// queue is a simple int queue to help with readability of Brandes'
// betweenness centrality implementation below.
type queue struct {
	queue []int
}

func newQueue(length int) queue {
	return queue{queue: make([]int, 0, length)}
}

func (q *queue) push(v int) {
	q.queue = append(q.queue, v)
}

func (q *queue) pop() int {
	v := q.queue[0]
	q.queue = q.queue[1:]
	return v
}

func (q *queue) empty() bool {
	return len(q.queue) == 0
}

// getNodeCentrality returns a slice of the shortest paths to the the node s and its betweenness centrality.
//
// We first calculate the shortest paths from the start node s to all other
// nodes with BFS, then update the betweenness centrality values by using
// Brandes' backpropagation of dependencies trick.
//
// For detailed explanation please read:
// https://www.cl.cam.ac.uk/teaching/1617/MLRD/handbook/brandes.html
func getNodeCentrality(adjList [][]int, s, nodesLen int) ([]int, []float64) {
	// s = src node
	// t = dst node
	// v = intermediate node
	// w = intermediate node

	// pred[w] is the list of nodes that immediately precede w on a
	// shortest path from s to t for each node t.
	pred := make([][]int, nodesLen)

	// sigma[t] is the number of shortest paths between nodes s and t
	// for each node t.
	sigma := make([]float64, nodesLen)
	sigma[s] = 1.0

	// distances[t] holds the distances between s and t for each node t.
	// We initialize this to -1 (meaning infinity) for each t != s.
	distances := make([]int, nodesLen)
	for i := range distances {
		distances[i] = -1
	}

	// Distance to self is 0
	distances[s] = 0

	stack := newStack(nodesLen)
	queue := newQueue(nodesLen)
	queue.push(s)

	// BFS to calculate the shortest paths (sigma and pred)
	// from s to t for each node t.
	for !queue.empty() {
		v := queue.pop()
		stack.push(v)

		for _, w := range adjList[v] {
			// If distance from s to w is infinity (-1) then set it and enqueue w.
			if distances[w] < 0 {
				distances[w] = distances[v] + 1
				queue.push(w)
			}

			// If w is on a shortest path then update sigma and add v to w's predecessor list.
			if distances[w] == distances[v]+1 {
				sigma[w] += sigma[v]
				pred[w] = append(pred[w], v)
			}
		}
	}

	// delta[v] is the ratio of the shortest paths between s and t that go
	// through v and the total number of shortest paths between s and t.
	// If we have delta then the betweenness centrality is simply the sum
	// of delta[w] for each w != s.
	delta := make([]float64, nodesLen)
	bc := make([]float64, nodesLen)

	for !stack.empty() {
		w := stack.pop()

		// pred[w] is the list of nodes that immediately precede w on a shortest path from s.
		for _, v := range pred[w] {
			// Update delta using Brandes' equation.
			delta[v] += (sigma[v] / sigma[w]) * (1.0 + delta[w])
		}

		if w != s {
			// As noted above centrality is simply the sum of delta[w] for each w != s.
			//
			// Divide by two as this is an undirected graph.
			bc[w] += delta[w] / 2.0
		}
	}

	return distances, bc
}
