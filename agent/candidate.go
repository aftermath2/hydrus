package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
)

const (
	oneDayInBlocks      = 144
	oneMonthInBlocks    = oneDayInBlocks * 30
	threeMonthsInBlocks = oneMonthInBlocks * 3
)

// nodeCandidate represents a node we might open a channel with.
type nodeCandidate struct {
	PublicKey string   `json:"public_key"`
	Addresses []string `json:"-"`
	Score     float64  `json:"score"`
}

// channelCandidate represents a channel we might close.
type channelCandidate struct {
	ChannelPoint string  `json:"channel_point,omitempty"`
	Active       bool    `json:"active,omitempty"`
	Score        float64 `json:"score,omitempty"`
}

// getCandidateNodes returns a ranking with candidates to open a channel to.
func getCandidateNodes(
	logger logger.Logger,
	localNode local.Node,
	graph graph.Graph,
	blocklist []string,
) []nodeCandidate {
	logger.Info("Getting candidate nodes to open a channel with")
	candidates := make([]nodeCandidate, 0, len(graph.Nodes))

	for _, node := range graph.Nodes {
		if err := discardNode(localNode, node, blocklist); err != nil {
			logger.Debugf("Discarding candidate node %q: %v", node.PublicKey, err)
			continue
		}

		candidates = append(candidates, nodeCandidate{
			PublicKey: node.PublicKey,
			Addresses: node.Addresses,
			Score:     graph.Heuristics.GetScore(node),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	candidatesB, _ := json.Marshal(candidates)
	logger.Debugf("Candidate nodes: %s", candidatesB)

	return candidates
}

// discardNode returns an error if the node should be skipped or nil if not.
func discardNode(localNode local.Node, peerNode graph.Node, blocklist []string) error {
	if slices.Contains(blocklist, peerNode.PublicKey) {
		return errors.New("blocklisted")
	}

	if _, ok := localNode.ChannelPeers[peerNode.PublicKey]; ok {
		return errors.New("already sharing a channel")
	}

	// Count the number of shared channel peers between local and candidate nodes
	numSharedPeers := uint64(0)
	for _, channel := range peerNode.Channels {
		if _, ok := localNode.ChannelPeers[channel.PeerPublicKey]; ok {
			numSharedPeers++
		}
	}

	// Discard nodes sharing 30% or more peers with us
	sharedPeersThreshold := getPercentage(uint64(len(localNode.ChannelPeers)), 30)
	if len(localNode.ChannelPeers) >= 10 && numSharedPeers > sharedPeersThreshold {
		return fmt.Errorf("sharing too many channel peers (%d)", numSharedPeers)
	}

	// Use int32 to avoid overflows setting the number too high
	threeMonthsAgo := int32(localNode.CurrentBlockHeight - threeMonthsInBlocks)

	for _, closedChannel := range localNode.ClosedChannels {
		if closedChannel.RemotePubkey != peerNode.PublicKey {
			continue
		}

		if closedChannel.CloseHeight != 0 && int32(closedChannel.CloseHeight) > threeMonthsAgo {
			return fmt.Errorf("a channel was closed with this peer within the last %d blocks", threeMonthsInBlocks)
		}

		if closedChannel.CloseType == lnrpc.ChannelCloseSummary_FUNDING_CANCELED &&
			closedChannel.OpenInitiator == lnrpc.Initiator_INITIATOR_LOCAL &&
			int32(graph.GetChannelBlockHeight(closedChannel.ChanId)) > threeMonthsAgo {
			return fmt.Errorf("we failed opening a channel with this peer within the last %d blocks",
				threeMonthsInBlocks)
		}
	}

	// Our own node may be in the graph
	if localNode.PublicKey == peerNode.PublicKey {
		return errors.New("own node")
	}

	return nil
}

// getCandidateChannels returns a ranking with the candidates channels to close.
func getCandidateChannels(logger logger.Logger, localNode local.Node, keeplist []string) []channelCandidate {
	logger.Info("Getting candidate channels to close")

	candidates := make([]channelCandidate, 0, len(localNode.Channels.List))

	for _, channel := range localNode.Channels.List {
		if slices.Contains(keeplist, channel.Point) {
			logger.Debugf("Discarding candidate channel %q: channel point is in the keeplist", channel.Point)
			continue
		}

		candidates = append(candidates, channelCandidate{
			ChannelPoint: channel.Point,
			Active:       channel.Active,
			Score:        localNode.Channels.Heuristics.GetScore(channel),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score < candidates[j].Score
	})

	candidatesB, _ := json.Marshal(candidates)
	logger.Debugf("Candidate channels: %s", candidatesB)

	return candidates
}
