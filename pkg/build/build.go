package build

import (
	"log"
	"os"
)

const (
	GOCOCO_DO_BUILD = iota
	GOCOCO_DO_INSTALL
)

type Build struct {
	OriArgs   []string
	NewFlags  []string
	NewArgs   []string
	BuildType int

	GOPATH           string
	GOBIN            string
	CurWd            string
	CacheWd          string
	CurProjectPath   string
	CacheProjectPath string

	ImportPath string
}

func NewBuild(opts ...Option) *Build {
	b := &Build{
		OriArgs: make([]string, 0),
	}

	for _, o := range opts {
		o(b)
	}

	// 1. get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get current working directory: %v", wd)
	}
	b.CurWd = wd

	// 2. parse flags and args
	// we only have official Go flags and args
	b.parseArgs()

	return b
}
