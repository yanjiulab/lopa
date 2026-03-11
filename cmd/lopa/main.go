package main

import (
	"log"

	"github.com/yanjiulab/lopa/internal/cli"
)

// lopa: CLI front-end, talks to lopad daemon over HTTP.
func main() {
	if err := cli.Execute(); err != nil {
		log.Fatalf("lopa exit with error: %v", err)
	}
}

