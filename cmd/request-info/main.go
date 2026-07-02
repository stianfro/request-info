package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/stianfro/request-info/internal/requestinfo"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	port, err := portFromEnv(os.Getenv("PORT"))
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestinfo.Handler(time.Now),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting request-info server", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func portFromEnv(value string) (string, error) {
	if value == "" {
		return "8080", nil
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		return "", fmt.Errorf("invalid PORT %q: %w", value, err)
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid PORT %q: must be between 1 and 65535", value)
	}
	return value, nil
}
