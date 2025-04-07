package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/channel"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/pkg/errors"
)

// Agent is in charge of looking for new nodes to open channels to, closing channels that are not performing
// well, and updating the routing policies of the channels that are maintained.
type Agent interface {
	CloseChannels(ctx context.Context, localNode local.Node) error
	OpenChannels(ctx context.Context, localNode local.Node) error
	UpdatePolicies(ctx context.Context, localNode local.Node) error
}

type agent struct {
	lnd            lightning.Client
	channelManager channel.Manager
	logger         logger.Logger
	config         config.Agent
}

// New returns a new agent interface.
func New(config config.Agent, lnd lightning.Client) Agent {
	return &agent{
		lnd:            lnd,
		channelManager: channel.NewManager(config.ChannelManager, lnd),
		logger:         logger.New("AGT"),
		config:         config,
	}
}

// CloseChannels evaluates the performance of local channels and closes those that do not meet minimum
// requirements.
func (a *agent) CloseChannels(ctx context.Context, localNode local.Node) error {
	if localNode.MaxCloseChannels == 0 {
		a.logger.Info("Too few channels to consider closing one, skipping channels closure")
		return nil
	}

	heuristics, err := json.Marshal(localNode.Channels.Heuristics)
	if err != nil {
		return errors.Wrap(err, "encoding channels heuristics")
	}
	a.logger.Debugf("Channels heuristics: %s", heuristics)

	candidates := getCandidateChannels(a.logger, localNode, a.config.Keeplist)

	channels := a.selectChannels(localNode, candidates)
	if len(channels) == 0 {
		a.logger.Info("No channels will be closed")
		return nil
	}

	a.logger.Infof("Closing channels: %v", channels)

	if a.config.DryRun {
		return nil
	}

	req := channel.CloseRequest{
		Channels: channels,
		SatvB:    localNode.SatvB,
	}
	return a.channelManager.Close(ctx, req)
}

// OpenChannels evaluates all nodes in the network graph, selects a list of candidates and creates a batching
// transaction opening channels to them.
func (a *agent) OpenChannels(ctx context.Context, localNode local.Node) error {
	if err := skipOpen(a.config, localNode); err != nil {
		a.logger.Infof("Skipping... %v", err)
		return nil
	}

	a.logger.Info("Generating network graph")

	networkGraph, err := graph.New(ctx, a.config.HeuristicWeights.Open, a.lnd)
	if err != nil {
		return errors.Wrap(err, "creating graph")
	}

	graphSize := len(networkGraph.Nodes)
	if graphSize == 0 {
		return errors.Wrap(err, "no nodes found in the network graph")
	}

	a.logger.Debugf("Filtered graph size: %d nodes", graphSize)

	heuristics, err := json.Marshal(networkGraph.Heuristics)
	if err != nil {
		return errors.Wrap(err, "encoding graph heuristics")
	}
	a.logger.Debugf("Graph heuristics: %s", heuristics)

	candidates := getCandidateNodes(a.logger, localNode, networkGraph, a.config.Blocklist)
	nodes := a.selectNodes(ctx, localNode, candidates)
	if len(nodes) == 0 {
		a.logger.Info("No channels will be opened")
		return nil
	}

	a.logger.Infof("Opening channels: %#v", nodes)

	if a.config.DryRun {
		return nil
	}

	req := channel.OpenRequest{
		Nodes: nodes,
		SatvB: localNode.SatvB,
	}
	return a.channelManager.Open(ctx, req)
}

func (a *agent) selectNodes(ctx context.Context, localNode local.Node, candidates []nodeCandidate) map[string]uint64 {
	nodes := make(map[string]uint64, localNode.MaxOpenChannels)
	fundingAmount := min(localNode.AllocatedBalance/localNode.MaxOpenChannels, a.config.MaxChannelSize)

	for _, candidate := range candidates {
		if len(nodes) == int(localNode.MaxOpenChannels) {
			break
		}

		if _, ok := localNode.SyncPeers[candidate.PublicKey]; !ok {
			a.logger.Debugf("Connecting with peer %q", candidate.PublicKey)

			// Try to connect to the peer and skip if we can't do it before the timeout
			if err := a.lnd.ConnectPeer(ctx, candidate.PublicKey, candidate.Addresses); err != nil {
				a.logger.Debugf("Couldn't connect with peer %q: %v. Discarding", candidate.PublicKey, err)
				continue
			}
		} else {
			a.logger.Debugf("Already connected with peer %q", candidate.PublicKey)
		}

		nodes[candidate.PublicKey] = fundingAmount
	}

	return nodes
}

func (a *agent) selectChannels(localNode local.Node, candidates []channelCandidate) map[string]bool {
	weightsSum := config.SumWeights(a.config.HeuristicWeights.Close)

	channels := make(map[string]bool, localNode.MaxCloseChannels)
	for _, candidate := range candidates {
		normalizedScore := candidate.Score * (1 / weightsSum)
		// If we have reached the maximum number of channel to close or the score is above 0.3,
		// skip the rest of the candidates
		if len(channels) >= int(localNode.MaxCloseChannels) || normalizedScore > 0.5 {
			break
		}

		forceClose := false
		if !candidate.Active {
			if a.config.AllowForceCloses {
				forceClose = true
			} else {
				// Do not force-close inactive channel, continue iterating the list
				a.logger.Infof(
					"The channel %q is inactive and force closes aren't allowed. Skipping channel closure",
					candidate.ChannelPoint,
				)
				continue
			}
		}

		channels[candidate.ChannelPoint] = forceClose
	}

	return channels
}

// UpdatePolicies evaluates the state of all local channels and updates their routing policies to maximize
// profits and routing reliability.
func (a *agent) UpdatePolicies(ctx context.Context, localNode local.Node) error {
	startTime := uint64(time.Now().Add(-a.config.RoutingPolicies.ActivityPeriod).Unix())
	forwards, err := local.ListForwards(ctx, a.lnd, startTime, 0)
	if err != nil {
		return err
	}

	for _, channel := range localNode.Channels.List {
		policy, err := getChannelPolicy(ctx, a.lnd, localNode.PublicKey, channel)
		if err != nil {
			a.logger.Error(err)
			continue
		}

		forwardsAmountIn := uint64(0)
		forwardsAmountOut := uint64(0)
		for _, forward := range forwards {
			if channel.ID == forward.ChanIdIn {
				forwardsAmountIn += forward.AmtIn
			}
			if channel.ID == forward.ChanIdOut {
				forwardsAmountOut += forward.AmtOut
			}
		}

		feeRatePPM := uint64(policy.FeeRateMilliMsat)
		newFeeRatePPM := calcNewFeeRate(channel, feeRatePPM, forwardsAmountIn, forwardsAmountOut)
		newMaxHTLC := calcNewMaxHTLC(channel)

		// No changes required, skip
		if newFeeRatePPM == feeRatePPM && newMaxHTLC == policy.MaxHtlcMsat {
			a.logger.Infof("Channel %q requires no changes, skipping", channel.Point)
			continue
		}

		a.logger.Infof("Updating %q channel policies. Fee rate: %d ppm. Max HTLC: %d",
			channel.Point,
			newFeeRatePPM,
			newMaxHTLC,
		)

		if a.config.DryRun {
			continue
		}

		if err := a.channelManager.UpdatePolicy(ctx, channel.Point, newFeeRatePPM, newMaxHTLC); err != nil {
			return err
		}
	}

	return nil
}

func getChannelPolicy(
	ctx context.Context,
	lnd lightning.Client,
	publicKey string,
	channel local.Channel,
) (*lnrpc.RoutingPolicy, error) {
	chanInfo, err := lnd.GetChanInfo(ctx, channel.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting %q channel info", channel.Point)
	}

	if chanInfo.Node1Pub == publicKey {
		return chanInfo.Node1Policy, nil
	}
	return chanInfo.Node2Policy, nil
}

func calcNewFeeRate(channel local.Channel, feeRatePPM, forwardsAmountIn, forwardsAmountOut uint64) uint64 {
	// If the local balance is lower than 1% of the channel's capacity, set a fee of 2100 ppm
	if channel.LocalBalance < getPercentage(channel.Capacity, 1) {
		return 2_100
	}

	// If local balance is higher than 99% of the channel capacity, set a fee rate of 0
	if channel.LocalBalance > getPercentage(channel.Capacity, 99) {
		return 0
	}

	// If there were no outgoing forwards, decrease the fee rate by 10%
	if forwardsAmountOut == 0 {
		return feeRatePPM - getPercentage(feeRatePPM, 10)
	}

	ratio := float64(forwardsAmountOut) / float64(forwardsAmountIn+forwardsAmountOut)

	// If more than half of the payments are forwarded in, decrease the outgoing fee rate by delta
	if ratio < 0.5 {
		delta := float64(feeRatePPM) * (0.5 - ratio)
		return feeRatePPM - uint64(delta)
	}

	// If more than half of the payments are forwarded out, increase the outgoing fee rate by delta
	if ratio > 0.5 {
		delta := float64(feeRatePPM) * (ratio - 0.5)
		return feeRatePPM + uint64(delta)
	}

	return feeRatePPM
}

func calcNewMaxHTLC(channel local.Channel) uint64 {
	if channel.LocalBalance < 2 {
		return 1_000
	}

	// Leave a buffer of 20% of the local balance to avoid running out of liquidity and starting to fail
	// payments before the next update
	newMaxHTLC := getPercentage(channel.LocalBalance, 80)
	return newMaxHTLC * 1000
}

// getPercentage returns the specified percent of value.
func getPercentage(value, percent uint64) uint64 {
	result := (float64(value) / 100.0) * float64(percent)
	return uint64(result)
}

// skipOpen returns true and a message if there are no new channels required.
func skipOpen(config config.Agent, localNode local.Node) error {
	if localNode.MaxOpenChannels < 1 {
		return errors.New("No new channels required")
	}

	if localNode.AllocatedBalance == 0 || localNode.AllocatedBalance < config.MinChannelSize {
		return errors.Errorf("Allocated funds (%d) is less than minimum channel size (%d)",
			localNode.AllocatedBalance, config.MinChannelSize,
		)
	}

	if localNode.NumChannels > config.MaxChannels {
		return errors.Errorf("Number of channels (%d) is higher than the maximum (%d)",
			localNode.NumChannels, config.MaxChannels,
		)
	}

	if localNode.MaxOpenChannels < config.MinBatchSize {
		return errors.Errorf("Number of channels to open (%d) is lower than the minimum batch size (%d)",
			localNode.MaxOpenChannels, config.MinBatchSize,
		)
	}

	return nil
}
