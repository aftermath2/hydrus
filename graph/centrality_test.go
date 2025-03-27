package graph

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	alice  = "alice"
	bob    = "bob"
	carol  = "carol"
	dave   = "dave"
	erin   = "erin"
	frank  = "frank"
	george = "george"
	harold = "harold"
)

var nodeIndices = map[string]int{
	alice:  0,
	bob:    1,
	carol:  2,
	dave:   3,
	erin:   4,
	frank:  5,
	george: 6,
	harold: 7,
}

var adjList = [][]int{
	{nodeIndices[carol], nodeIndices[frank]},
	{nodeIndices[dave]},
	{nodeIndices[alice], nodeIndices[erin]},
	{nodeIndices[bob], nodeIndices[erin], nodeIndices[frank]},
	{nodeIndices[carol], nodeIndices[dave], nodeIndices[george]},
	{nodeIndices[alice], nodeIndices[dave], nodeIndices[harold]},
	{nodeIndices[erin]},
	{nodeIndices[frank]},
}

func TestNewAdjacencyList(t *testing.T) {
	nodes := []Node{
		{
			PublicKey: alice,
			Channels: []Channel{
				{PeerPublicKey: carol},
				{PeerPublicKey: frank},
			},
		},
		{
			PublicKey: bob,
			Channels: []Channel{
				{PeerPublicKey: dave},
			},
		},
		{
			PublicKey: carol,
			Channels: []Channel{
				{PeerPublicKey: alice},
				{PeerPublicKey: erin},
			},
		},
		{
			PublicKey: dave,
			Channels: []Channel{
				{PeerPublicKey: bob},
				{PeerPublicKey: frank},
				{PeerPublicKey: erin},
			},
		},
		{
			PublicKey: erin,
			Channels: []Channel{
				{PeerPublicKey: carol},
				{PeerPublicKey: dave},
				{PeerPublicKey: george},
			},
		},
		{
			PublicKey: frank,
			Channels: []Channel{
				{PeerPublicKey: alice},
				{PeerPublicKey: dave},
				{PeerPublicKey: harold},
			},
		},
		{
			PublicKey: george,
			Channels: []Channel{
				{PeerPublicKey: erin},
			},
		},
		{
			PublicKey: harold,
			Channels: []Channel{
				{PeerPublicKey: frank},
			},
		},
	}
	actualAdjList := newAdjacencyList(nodes, nodeIndices)

	for _, list := range actualAdjList {
		slices.Sort(list)
	}

	assert.Equal(t, adjList, actualAdjList)
}

func TestGetCentrality(t *testing.T) {
	expectedDistances := []int{14, 17, 14, 11, 12, 12, 18, 18}
	expectedCentrality := []float64{2, 0, 2, 10, 8, 8, 0, 0}
	distances, centrality := getCentrality(t.Context(), nodeIndices, adjList)

	assert.Equal(t, expectedDistances, distances)
	assert.Equal(t, expectedCentrality, centrality)
}

func TestGetEigenvectorCentrality(t *testing.T) {
	expectedResult := []uint64{26, 15, 26, 37, 32, 32, 15, 15}
	result := getEigenvectorCentrality(nodeIndices, adjList)

	assert.Equal(t, expectedResult, result)
}

func TestGetNodeCentrality(t *testing.T) {
	expectedDistances := []int{0, 3, 1, 2, 2, 1, 3, 2}
	expectedCentrality := []float64{0, 0, 1, 0.5, 0.5, 1.5, 0, 0}

	distances, centrality := getNodeCentrality(adjList, 0, len(nodeIndices))

	assert.Equal(t, expectedDistances, distances)
	assert.Equal(t, expectedCentrality, centrality)
}
