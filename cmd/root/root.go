package root

import (
	"github.com/aftermath2/hydrus/cmd/agent"
	"github.com/aftermath2/hydrus/cmd/channels"
	"github.com/aftermath2/hydrus/cmd/scores"

	"github.com/spf13/cobra"
)

// NewCmd returns a new root command.
func NewCmd() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:           "hydrus",
		Short:         "Lightning liquidity management agent",
		Version:       "0.2.0",
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringP("config", "c", "", "Path to the configuration file")

	cmd.AddCommand(
		agent.NewCmd(),
		channels.NewCmd(),
		scores.NewCmd(),
	)

	return cmd, nil
}
