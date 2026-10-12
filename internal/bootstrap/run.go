// Package bootstrap is the composition root for the headless API.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/adapter/http"
	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/redact"
	"github.com/cznUSER/B-HIS/internal/store"
	"github.com/cznUSER/B-HIS/internal/webui"
)

// Options selects process extras. The zero value is the headless API.
type Options struct {
	// UI serves the embedded login SPA on this listener. /api and /health stay on the API.
	UI bool
	// OnReady runs after the port is accepting connections. listenAddr is host:port.
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

	users, revocations, audit, patients, closer, err := store.Open(cfg)
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
	svc := &app.Service{
		Users:       users,
		Revocations: revocations,
		Audit:       audit,
		Patients:    patients,
		Tokens:      issuer,
		Limiter:     auth.NewLimiter(cfg.MaxFailures(), window),
		Log:         log,
		TTL:         ttl,
	}
	handler := httpapi.New(svc, log, cfg.CORS.AllowedOrigins, cfg.Database.Driver, window)
	if opt.UI {
		handler = webui.Mount(handler)
	}
	ln, err := net.Listen("tcp", cfg.Host.ApiAddr)
	if err != nil {
		return fmt.Errorf("%s", redact.String(err.Error()))
	}
	readyLn := &readyListener{Listener: ln, ready: make(chan struct{})}
	srv := &http.Server{
		Addr:              cfg.Host.ApiAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	msg := "addr=" + ln.Addr().String() + " driver=" + cfg.Database.Driver
	if opt.UI {
		msg += " ui=embedded"
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

	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(readyLn)
		if errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	select {
	case <-readyLn.ready:
		if opt.OnReady != nil {
			opt.OnReady(ln.Addr().String())
		}
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("%s", redact.String(err.Error()))
		}
		return nil
	case <-ctx.Done():
		return shutdown(srv, log)
	}

	select {
	case <-ctx.Done():
		return shutdown(srv, log)
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("%s", redact.String(err.Error()))
		}
		return nil
	}
}

func shutdown(srv *http.Server, log *logging.Logger) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(shutdownCtx)
	log.Log(logging.Event{Level: "info", Action: "server.stop", Outcome: "SUCCESS"})
	return err
}

// readyListener closes ready when Serve begins accepting.
type readyListener struct {
	net.Listener
	ready chan struct{}
	once  sync.Once
}

func (l *readyListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.ready) })
	return l.Listener.Accept()
}
