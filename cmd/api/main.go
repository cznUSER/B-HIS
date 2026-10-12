// Command api is the headless H API. It does not require the desktop host.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cznUSER/B-HIS/internal/bootstrap"
	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/redact"
)

func main() {
	fs := flag.NewFlagSet("api", flag.ContinueOnError)
	configPath := fs.String("config", "", "YAML config path (optional; BHIS_* env and gitignored local files also apply)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %s\n", redact.String(err.Error()))
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := bootstrap.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "api: %s\n", redact.String(err.Error()))
		os.Exit(1)
	}
}
