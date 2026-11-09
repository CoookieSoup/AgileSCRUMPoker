package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CoookieSoup/AgileSCRUMPoker/backend/app"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/config"
	"github.com/CoookieSoup/AgileSCRUMPoker/backend/infra/mysql"
)

const (
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := run(logger)
	if err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	settings, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := mysql.Open(settings.DatabaseDSN)
	if err != nil {
		return err
	}

	err = mysql.Migrate(database)
	if err != nil {
		return err
	}

	application := app.New(app.Options{
		Database:       database,
		Logger:         logger,
		AllowedOrigins: settings.AllowedOrigins,
		OriginPatterns: settings.OriginPatterns(),
	})

	server := &http.Server{
		Addr:              settings.HTTPAddress,
		Handler:           application.Handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go application.Cleanup.Run(ctx, settings.CleanupInterval)

	go func() {
		<-ctx.Done()

		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		_ = server.Shutdown(shutdownContext)
	}()

	logger.Info("server listening", "address", settings.HTTPAddress)

	err = server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}

	return nil
}
