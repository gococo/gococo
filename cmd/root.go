package cmd

import (
	"os"

	"github.com/gococo/gococo/pkg/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gococo",
	Short: "gococo is a Go Coverage Collection tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		debug := false
		if os.Getenv("GOCOCO_DEBUG") == "true" {
			debug = true
		}
		log.NewLogger(debug)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		log.Sync()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
	}
}
