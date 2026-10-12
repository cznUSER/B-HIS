// Package bootstrap is the composition root for the headless API.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cznUSER/B-HIS/internal/adapter/http"
	"github.com/cznUSER/B-HIS/internal/app"
	"github.com/cznUSER/B-HIS/internal/auth"
	"github.com/cznUSER/B-HIS/internal/config"
	"github.com/cznUSER/B-HIS/internal/logging"
	"github.com/cznUSER/B-HIS/internal/redact"
	"github.com/cznUSER/B-HIS/internal/store"
)

// Run serves the API until ctx is cancelled. It logs the listen address and driver, never the DSN.
func Run(ctx context.Context, cfg config.Config) error {
	log, err := logging.New(cfg.Logging.Dir, cfg.Logging.Level)
	if err != nil {
		return err
	}
	defer log.Close()

	users, revocations, audit, closer, err := store.Open(cfg)
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
		Tokens:      issuer,
		Limiter:     auth.NewLimiter(cfg.MaxFailures(), window),
		Log:         log,
		TTL:         ttl,
	}
	handler := httpapi.New(svc, log, cfg.CORS.AllowedOrigins, cfg.Database.Driver, window)
	srv := &http.Server{
		Addr:              cfg.Host.ApiAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Log(logging.Event{
		Level:   "info",
		Action:  "server.start",
		Outcome: "SUCCESS",
		Message: "addr=" + cfg.Host.ApiAddr + " driver=" + cfg.Database.Driver,
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
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		log.Log(logging.Event{Level: "info", Action: "server.stop", Outcome: "SUCCESS"})
		return err
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("%s", redact.String(err.Error()))
		}
		return nil
	}
}
