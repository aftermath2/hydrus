package agent

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/channel"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/graph"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/pkg/errors"
)

// Agent is in charge of looking new nodes to open channels to and closing channel that are not performing
// well.
type Agent interface {
	Start(ctx context.Context) error
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

func (a *agent) Start(ctx context.Context) error {
	localNode, err := local.GetNode(ctx, a.config, a.lnd)
	if err != nil {
		return err
	}
	a.logger.Debugf("Local node: %s", localNode)

	if localNode.SatvB > a.config.ChannelManager.MaxSatvB {
		a.logger.Infof(
			"Skipping... The estimated transaction fee per virtual byte (%d) is higher than the maximum (%d)",
			localNode.SatvB,
			a.config.ChannelManager.MaxSatvB,
		)
		return nil
	}

	a.logger.Info("Evaluating channels to close")
	if err := a.closeChannels(ctx, localNode); err != nil {
		return err
	}

	a.logger.Info("Evaluating channels to open")
	if err := a.openChannels(ctx, localNode); err != nil {
		return err
	}

	return nil
}

func (a *agent) closeChannels(ctx context.Context, localNode local.Node) error {
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

func (a *agent) openChannels(ctx context.Context, localNode local.Node) error {
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
	weightsSum := SumWeights(a.config.HeuristicWeights.Close)

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

// SumWeights returns the sum of the values stored in the weights objects.
func SumWeights[T config.CloseWeights | config.OpenWeights](weights T) float64 {
	w := reflect.ValueOf(weights)
	sum := 0.0
	for i := range w.NumField() {
		sum += w.Field(i).Interface().(float64)
	}

	return sum
}
