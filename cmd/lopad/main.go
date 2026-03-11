package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yanjiulab/lopa/internal/config"
	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/node"
	"github.com/yanjiulab/lopa/internal/reflector"
	"github.com/yanjiulab/lopa/internal/server"
)

var noReflector bool

func init() {
	flag.BoolVar(&noReflector, "no-reflector", false, "disable reflector (UDP echo server)")
}

// lopad: Lopa daemon, runs measurement engine and HTTP API in background.
func main() {
	flag.Parse()
	if _, err := config.Load(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := logger.Init(); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-node"
	}
	node.SetNodeID(hostname)

	// Start HTTP API server
	server.Start()

	// Start reflector by default (unless disabled by config or --no-reflector)
	reflectorEnabled := !noReflector && config.Global().Reflector.Enabled
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if reflectorEnabled {
		go func() {
			_ = reflector.Run(ctx, config.Global().Reflector.Addr)
		}()
	} else {
		logger.S().Info("reflector disabled by config or --no-reflector")
	}

	<-ctx.Done()
}

