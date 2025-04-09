package channels

import (
	"context"

	"github.com/aftermath2/hydrus/agent"
	"github.com/aftermath2/hydrus/agent/local"
	"github.com/aftermath2/hydrus/cmd"
	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/spf13/cobra"
)

// NewOpenCmd returns a new run command.
func NewOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Evaluate nodes to connect to and create the funding transaction",
		RunE: cmd.Run(func(ctx context.Context, config *config.Config, lnd lightning.Client, logger logger.Logger) error {
			localNode, err := local.GetNode(ctx, config.Agent, lnd)
			if err != nil {
				return err
			}
			logger.Debugf("Local node: %s", localNode)

			if localNode.SatvB > config.Agent.ChannelManager.MaxSatvB {
				logger.Infof(
					"Skipping... The estimated transaction fee per virtual byte (%d) is higher than the maximum (%d)",
					localNode.SatvB,
					config.Agent.ChannelManager.MaxSatvB,
				)
				return nil
			}

			logger.Info("Evaluating channels to close")
			agent := agent.New(config.Agent, lnd)
			return agent.OpenChannels(ctx, localNode)
		}),
	}
}
