package cmd

import (
	"context"

	"github.com/aftermath2/hydrus/config"
	"github.com/aftermath2/hydrus/lightning"
	"github.com/aftermath2/hydrus/logger"

	"github.com/spf13/cobra"
)

// RunE represents a command Run function that could return an error.
type RunE func(cmd *cobra.Command, args []string) error

// Run loads dependencies and wraps a function that can be used to avoid repeating the same initialization
// logic on each command.
func Run(f func(ctx context.Context, config *config.Config, lnd lightning.Client, logger logger.Logger) error) RunE {
	return func(cmd *cobra.Command, _ []string) error {
		configPath := cmd.InheritedFlags().Lookup("config")

		config, err := config.Load(configPath.Value.String())
		if err != nil {
			return err
		}

		lnd, err := lightning.NewClient(config.Lightning)
		if err != nil {
			return err
		}

		logger := logger.New("HYD")
		return f(cmd.Context(), config, lnd, logger)
	}
}
