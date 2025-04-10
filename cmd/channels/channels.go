package channels

import (
	"github.com/spf13/cobra"
)

// NewCmd returns a new scores command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Channels operations",
		Long:  "Perform channels operations such as opening, closing and updating routing policies",
	}

	cmd.AddCommand(
		NewCloseCmd(),
		NewOpenCmd(),
		NewUpdateCmd(),
	)

	return cmd
}
