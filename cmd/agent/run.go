package agent

import (
	"context"

	"github.com/aftermath2/hydrus/agent"
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
		RunE: cmd.Run(func(ctx context.Context, config *config.Config, lnd lightning.Client, _ logger.Logger) error {
			agent := agent.New(config.Agent, lnd)
			return agent.Run(ctx)
		}),
	}
}
