// Command host is the local demo launcher. Double-click the Windows exe to
// start the API (memory by default), serve the login page, and open a browser.
// 2026-10-12 无参数即启动；cmd/api 仍是无头入口。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/cznUSER/B-HIS/internal/bootstrap"
	"github.com/cznUSER/B-HIS/internal/redact"
)

func main() {
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	configPath := fs.String("config", "", "YAML config path")
	noBrowser := fs.Bool("no-browser", false, "do not open a browser")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	args := fs.Args()
	cmd := "start"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "start":
		if err := start(*configPath, browserEnabled(*noBrowser)); err != nil {
			fail(err)
		}
	case "health":
		cfg, err := loadDemo(discoverConfig(*configPath))
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
		cfg, err := loadDemo(discoverConfig(*configPath))
		if err != nil {
			fail(err)
		}
		dir, err := filepath.Abs(cfg.Logging.Dir)
		if err != nil {
			fail(err)
		}
		fmt.Printf("log dir: %s\ndriver: %s\n", dir, cfg.Database.Driver)
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func start(configFlag string, open bool) error {
	cfg, err := loadDemo(discoverConfig(configFlag))
	if err != nil {
		return err
	}
	page := loginURL(cfg.Host.ApiAddr)
	fmt.Printf("B-HIS local demo\nLogin: %s\nDatabase: %s\nDemo account: admin / Admin@123\nKeep this window open. Close it or press Ctrl+C to stop.\n", page, cfg.Database.Driver)
	if cfg.PlaceholderSecret() {
		fmt.Fprintf(os.Stderr, "host: jwt secret is a placeholder; set BHIS_JWT_SECRET outside this demo\n")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return bootstrap.RunWith(ctx, cfg, bootstrap.Options{
		UI: true,
		OnReady: func(string) {
			if !open {
				return
			}
			if err := openBrowser(page); err != nil {
				fmt.Fprintf(os.Stderr, "browser: %s\n", redact.String(err.Error()))
			}
		},
	})
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "host: %s\n", redact.String(err.Error()))
	pauseIfDoubleClick()
	os.Exit(1)
}

// pauseIfDoubleClick keeps the console visible when Explorer launches the exe
// and startup fails. A terminal session (extra args) is left alone.
func pauseIfDoubleClick() {
	if runtime.GOOS != "windows" || len(os.Args) != 1 {
		return
	}
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "Press Enter to close this window.")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func usage() {
	fmt.Fprintf(os.Stderr, `b-his-host serves the login page and the API in one process.

Double-click bin\b-his-host.exe (or run it with no arguments). A browser opens
the local login page. Default database is memory. Demo account: admin / Admin@123
(this is not a SQL Server password).

  b-his-host.exe
  b-his-host.exe -no-browser
  b-his-host.exe -config config.local.yaml start
  b-his-host.exe health
  b-his-host.exe logs

Flags belong before the subcommand. BHIS_OPEN_BROWSER=0 skips the browser.
No config file is required for the memory demo. Optional config.local.yaml
(next to the exe, or in the working directory) can set sqlserver via
BHIS_DB_DSN. Do not commit that file.

Headless API, without the login page:

  go run ./cmd/api -config config.example.yaml

Vite mock UI for development stays in web/: npm run dev
(mock account admin / admin123). Rebuild the exe with scripts\build-windows.ps1.
`)
}
