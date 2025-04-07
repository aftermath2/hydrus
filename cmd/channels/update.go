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

// NewUpdateCmd returns a new run command.
func NewUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Evaluate local channels and update their routing policies",
		RunE: cmd.Run(func(ctx context.Context, config *config.Config, lnd lightning.Client, logger logger.Logger) error {
			localNode, err := local.GetNode(ctx, config.Agent, lnd)
			if err != nil {
				return err
			}
			logger.Debugf("Local node: %s", localNode)

			logger.Info("Evaluating channels to update")
			agent := agent.New(config.Agent, lnd)
			return agent.UpdatePolicies(ctx, localNode)
		}),
	}
}
