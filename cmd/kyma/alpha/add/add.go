package add

import (
	"github.com/spf13/cobra"
)

//NewCmd creates a new add command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds new resources to the config.",
	}
	return cmd
}
