package scores

import (
	"github.com/spf13/cobra"
)

// NewCmd returns a new scores command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scores",
		Short: "Show channels and nodes scores",
		Long:  "Show scoring information for nodes and channels, providing insights based on different metrics and heuristics.",
	}

	cmd.AddCommand(
		NewChannelsCmd(),
		NewNodesCmd(),
	)

	return cmd
}
