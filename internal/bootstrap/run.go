// Package bootstrap is the composition root for the headless API.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/adapter/http"
	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/port"
	"github.com/cznUSER/B-HIS/internal/redact"
	"github.com/cznUSER/B-HIS/internal/store"
	"github.com/cznUSER/B-HIS/internal/webui"
)

// Options selects process extras. The zero value is the headless API.
type Options struct {
	// UI serves the embedded login SPA on Host.WebAddr and proxies API traffic to Host.ApiAddr.
	UI bool
	// OnReady runs after all enabled listeners accept connections. listenAddr is host:port.
	OnReady func(listenAddr string)
}

// Run serves the API until ctx is cancelled. It logs the listen address and driver, never the DSN.
func Run(ctx context.Context, cfg config.Config) error {
	return RunWith(ctx, cfg, Options{})
}

// RunWith serves the API until ctx is cancelled. 2026-10-12 可选同源登录页。
func RunWith(ctx context.Context, cfg config.Config, opt Options) error {
	log, err := logging.New(cfg.Logging.Dir, cfg.Logging.Level)
	if err != nil {
		return err
	}
	defer log.Close()

	users, revocations, audit, patients, registers, orders, charges, dispenses, closer, err := store.Open(cfg)
	if err != nil {
		return fmt.Errorf("%s", redact.String(err.Error()))
	}
	defer closer.Close()

	ttl, err := cfg.TokenTTL()
	if err != nil {
		return err
	}
	window, err := cfg.RateWindow()
	if err != nil {
		return err
	}
	issuer, err := auth.NewIssuer(cfg.Auth.JWTSecret, ttl)
	if err != nil {
		return err
	}
	master, ok := users.(port.MasterStore)
	if !ok {
		return fmt.Errorf("store does not implement master data")
	}
	diagnoses, ok := any(registers).(port.DiagnoseStore)
	if !ok {
		return fmt.Errorf("store does not implement diagnoses")
	}
	dashboard, _ := any(charges).(port.DashboardStore)
	batchStore, ok := any(dispenses).(port.InventoryBatchStore)
	if !ok {
		return fmt.Errorf("store does not implement inventory batch reads")
	}
	svc := &app.Service{
		Users:               users,
		Revocations:         revocations,
		Audit:               audit,
		Patients:            patients,
		Registers:           registers,
		Diagnoses:           diagnoses,
		Orders:              orders,
		Charges:             charges,
		Dispenses:           dispenses,
		InventoryBatches:    batchStore,
		Master:              master,
		Dashboard:           dashboard,
		Tokens:              issuer,
		Limiter:             auth.NewLimiter(cfg.MaxFailures(), window),
		Log:                 log,
		TTL:                 ttl,
		RegisterExpireHours: cfg.RegisterExpireHours,
	}
	apiHandler := httpapi.New(svc, log, cfg.CORS.AllowedOrigins, cfg.Database.Driver, window)
	apiLn, err := net.Listen("tcp", cfg.Host.ApiAddr)
	if err != nil {
		return fmt.Errorf("%s", redact.String(err.Error()))
	}
	servers := []serverBinding{{
		server:   newHTTPServer(cfg.Host.ApiAddr, apiHandler),
		listener: apiLn,
	}}
	msg := "api_addr=" + apiLn.Addr().String() + " driver=" + cfg.Database.Driver
	readyAddr := apiLn.Addr().String()
	if opt.UI {
		apiURL, err := localHTTPURL(cfg.Host.ApiAddr)
		if err != nil {
			_ = apiLn.Close()
			return err
		}
		webLn, err := net.Listen("tcp", cfg.Host.WebAddr)
		if err != nil {
			_ = apiLn.Close()
			return fmt.Errorf("%s", redact.String(err.Error()))
		}
		servers = append(servers, serverBinding{
			server:   newHTTPServer(cfg.Host.WebAddr, webui.Mount(apiURL)),
			listener: webLn,
		})
		msg += " web_addr=" + webLn.Addr().String() + " ui=embedded"
		readyAddr = webLn.Addr().String()
	}
	log.Log(logging.Event{
		Level:   "info",
		Action:  "server.start",
		Outcome: "SUCCESS",
		Message: msg,
	})
	if cfg.PlaceholderSecret() {
		log.Log(logging.Event{
			Level:   "warn",
			Action:  "server.config",
			Outcome: "SUCCESS",
			Message: "jwt secret is a placeholder; set BHIS_JWT_SECRET for any shared environment",
		})
	}

	return serveServers(ctx, servers, readyAddr, opt.OnReady, log)
}

type serverBinding struct {
	server   *http.Server
	listener net.Listener
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func localHTTPURL(addr string) (string, error) {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid Host.ApiAddr %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	} else if host == "::" {
		host = "::1"
	}
	return "http://" + net.JoinHostPort(host, port), nil
}

func serveServers(ctx context.Context, bindings []serverBinding, readyAddr string, onReady func(string), log *logging.Logger) error {
	readyCh := make(chan string, len(bindings))
	errCh := make(chan error, len(bindings))
	for _, binding := range bindings {
		readyListener := &readyListener{Listener: binding.listener, ready: readyCh}
		go func(server *http.Server, listener net.Listener) {
			err := server.Serve(listener)
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			errCh <- err
		}(binding.server, readyListener)
	}

	readyCount := 0
	for readyCount < len(bindings) {
		select {
		case <-readyCh:
			readyCount++
		case err := <-errCh:
			_ = shutdownServers(bindings, log)
			if err == nil {
				err = fmt.Errorf("HTTP server stopped before all listeners were ready")
			}
			return fmt.Errorf("%s", redact.String(err.Error()))
		case <-ctx.Done():
			return shutdownServers(bindings, log)
		}
	}
	if onReady != nil {
		onReady(readyAddr)
	}

	select {
	case <-ctx.Done():
		return shutdownServers(bindings, log)
	case err := <-errCh:
		_ = shutdownServers(bindings, log)
		if err == nil {
			return fmt.Errorf("HTTP server stopped unexpectedly")
		}
		return fmt.Errorf("%s", redact.String(err.Error()))
	}
}

func shutdownServers(bindings []serverBinding, log *logging.Logger) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var firstErr error
	for _, binding := range bindings {
		if err := binding.server.Shutdown(shutdownCtx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	log.Log(logging.Event{Level: "info", Action: "server.stop", Outcome: "SUCCESS"})
	return firstErr
}

// readyListener closes ready when Serve begins accepting.
type readyListener struct {
	net.Listener
	ready chan<- string
	once  sync.Once
}

func (l *readyListener) Accept() (net.Conn, error) {
	l.once.Do(func() { l.ready <- l.Addr().String() })
	return l.Listener.Accept()
}
