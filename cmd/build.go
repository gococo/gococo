package cmd

import (
	"github.com/gococo/gococo/pkg/build"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:                "build",
	DisableFlagParsing: true,
	Run:                buildAction,
}

func buildAction(cmd *cobra.Command, args []string) {
	b := build.NewBuild(
		build.WithBuild(),
		build.WithArgs(args...),
	)

	b.Build()
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
