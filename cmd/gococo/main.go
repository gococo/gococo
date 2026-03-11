package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gococo/gococo/internal/instrument"
	"github.com/gococo/gococo/internal/server"
)

const usage = `gococo - real-time Go coverage visualization

Usage:
  gococo server [--addr HOST:PORT]     Start the relay server
  gococo build  [--host HOST:PORT] [BUILD_FLAGS...] [PACKAGES]
                                       Instrument and build a Go project
  gococo version                       Show version

Environment:
  GOCOCO_HOST   Override the server address in instrumented binaries
`

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		runServer()
	case "build":
		runBuild()
	case "version":
		fmt.Printf("gococo %s\n", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(1)
	}
}

func runServer() {
	addr := "127.0.0.1:7778"
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr", "-addr":
			if i+1 < len(args) {
				addr = args[i+1]
				i++
			}
		}
	}

	// Embedded web UI will be wired in later; for now serve a placeholder.
	s := server.New(addr, http.Dir("web/dist"))
	if err := s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func runBuild() {
	host := "127.0.0.1:7778"
	debug := false
	var goFlags []string
	var packages []string
	outputDir := ""

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host", "-host":
			if i+1 < len(args) {
				host = args[i+1]
				i++
			}
		case "--debug":
			debug = true
		case "-o":
			if i+1 < len(args) {
				outputDir = args[i+1]
				goFlags = append(goFlags, "-o", args[i+1])
				i++
			}
		default:
			a := args[i]
			if len(a) > 0 && a[0] == '-' {
				goFlags = append(goFlags, a)
				// Check if this flag takes a value
				if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
					goFlags = append(goFlags, args[i+1])
					i++
				}
			} else {
				packages = append(packages, a)
			}
		}
	}

	opts := instrument.Options{
		Host:      host,
		Packages:  packages,
		GoFlags:   goFlags,
		OutputDir: outputDir,
		Debug:     debug,
	}

	if err := instrument.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}
}
