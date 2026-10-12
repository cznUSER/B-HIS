// Command host is a thin desktop-host stub. The API also runs on its own via cmd/api.
// A Windows tray can call these subcommands later; this build only starts, checks health, and prints the log directory.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cznUSER/B-HIS/internal/bootstrap"
	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/redact"
)

func main() {
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	configPath := fs.String("config", "", "YAML config path")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	args := fs.Args()
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "start":
		cfg, err := config.Load(*configPath)
		if err != nil {
			fail(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := bootstrap.Run(ctx, cfg); err != nil {
			fail(err)
		}
	case "health":
		cfg, err := config.Load(*configPath)
		if err != nil {
			fail(err)
		}
		url := "http://" + cfg.Host.ApiAddr + "/health"
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "health: %s\n", redact.String(err.Error()))
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		fmt.Printf("%s %d\n%s\n", url, resp.StatusCode, body)
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
	case "logs":
		cfg, err := config.Load(*configPath)
		if err != nil {
			fail(err)
		}
		dir, err := filepath.Abs(cfg.Logging.Dir)
		if err != nil {
			fail(err)
		}
		fmt.Printf("log dir: %s\ndriver: %s\n", dir, cfg.Database.Driver)
	default:
		usage()
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "host: %s\n", redact.String(err.Error()))
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `h-host is a thin stub around the API process (tray UI is not in this build).

The API does not need the host. Headless:

  go run ./cmd/api -config config.example.yaml

Host subcommands (flags before the subcommand):

  go run ./cmd/host -config config.example.yaml start
  go run ./cmd/host -config config.example.yaml health
  go run ./cmd/host -config config.example.yaml logs

Database driver defaults to memory. sqlserver reads BHIS_DB_DSN from the environment
or a gitignored config.local.yaml / .env.local. Do not commit that file.
`)
}
