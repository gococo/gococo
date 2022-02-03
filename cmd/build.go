package cmd

import (
	"os"

	"github.com/gococo/gococo/pkg/build/cache"
	"github.com/gococo/gococo/pkg/log"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:    "build",
	Short:  "build the main package",
	PreRun: preBuild,
	Run:    build,
}

func preBuild(cmd *cobra.Command, args []string) {

}

func build(cmd *cobra.Command, args []string) {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("fail to get current working directory: %v", err)
	}

	c, err := cache.NewBuildCache(pwd, cache.WithSkip(".git"))
	if err != nil {
		log.Fatalf("fail to init build cache: %v", err)
	}

	c.Cache()
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
