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

	"github.com/ckchessmaster/zitadel-actions-api/internal/config"
	"github.com/ckchessmaster/zitadel-actions-api/internal/handler"
	"github.com/ckchessmaster/zitadel-actions-api/internal/middleware"
	"github.com/ckchessmaster/zitadel-actions-api/internal/service"
)

func main() {
	cfg := config.Load()

	// Initialize structured logger
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("starting zitadel-actions-api",
		slog.String("port", cfg.Port),
		slog.String("log_level", cfg.LogLevel),
		slog.String("claim_name", cfg.ClaimName),
		slog.String("role_format", cfg.RoleFormat),
		slog.Bool("lowercase_roles", cfg.LowercaseRoles),
		slog.String("filter_project_id", cfg.FilterProjectID),
		slog.Bool("signature_verification", cfg.ZitadelSigningKey != ""),
	)

	// Initialize dispatcher and action processors
	flattenerService := service.NewRoleFlattener()
	actionDispatcher := service.NewDispatcher(flattenerService)
	unifiedActionHandler := handler.NewActionsHandler(actionDispatcher, cfg, logger)

	// Actions router with signature validation
	actionsMux := http.NewServeMux()
	actionsMux.Handle("/actions", unifiedActionHandler)

	// Wrap actions with signature validation middleware
	validatedActionsHandler := middleware.SignatureValidator(cfg.ZitadelSigningKey, logger)(actionsMux)

	// Main root router
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /healthz", handler.Healthz())
	rootMux.HandleFunc("GET /readyz", handler.Readyz())
	rootMux.Handle("/actions", validatedActionsHandler)

	// Apply global middleware: recoverer, then logger
	rootHandler := middleware.Recoverer(logger)(middleware.Logger(logger)(rootMux))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      rootHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server shutdown channel
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	sig := <-shutdownChan
	logger.Info("received shutdown signal", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server exited cleanly")
}
