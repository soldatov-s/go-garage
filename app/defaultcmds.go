package app

import (
	"github.com/spf13/cobra"
)

// CreateServeCmd create serve command
func CreateServeCmd(handler func(cmd *cobra.Command, args []string)) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "command for starting HTTP server and connections to third services",
		Run:   handler,
	}
}
