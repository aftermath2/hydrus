package agent

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

// NewRunCmd returns a new run command.
func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the agent, executing actions on intervals",
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

			agent := agent.New(config.Agent, lnd)

			logger.Info("Evaluating channels to close")
			if err := agent.CloseChannels(ctx, localNode); err != nil {
				return err
			}

			logger.Info("Evaluating channels to open")
			return agent.OpenChannels(ctx, localNode)
		}),
	}
}
