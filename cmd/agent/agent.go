package agent

import (
	"github.com/spf13/cobra"
)

// NewCmd returns a new agent command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent operations",
	}

	cmd.AddCommand(
		NewRunCmd(),
	)
	return cmd
}
