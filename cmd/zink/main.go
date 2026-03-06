package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
	"github.com/fortega2/zink/internal/middleware/auth"
	"github.com/fortega2/zink/internal/middleware/logging"
	"github.com/fortega2/zink/internal/middleware/ratelimit"
	"github.com/fortega2/zink/internal/proxy"
	"github.com/fortega2/zink/internal/server"
)

func main() {
	configPath := flag.String("config", "zink.yml", "path to the configuration file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger.Info("starting Zink...", "configPath", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		return
	}

	logger.Info("configuration loaded successfully", "host", cfg.Server.Host, "port", cfg.Server.Port)

	for _, svc := range cfg.Services {
		logger.Info("service registered", "name", svc.Name, "path_prefix", svc.PathPrefix, "targets", len(svc.Target))
	}

	registry := middleware.NewRegistry()
	registry.Register(string(config.MiddlewareRateLimit), func(ctx context.Context, value any, _ *slog.Logger) (middleware.Middleware, error) {
		cfg, ok := value.(config.RateLimitMiddleware)
		if !ok {
			return nil, fmt.Errorf("rate_limit: unexpected value type %T", value)
		}
		return ratelimit.New(ctx, cfg.Rate, cfg.Burst), nil
	})
	registry.Register(string(config.MiddlewareAuth), func(_ context.Context, value any, _ *slog.Logger) (middleware.Middleware, error) {
		cfg, ok := value.(config.AuthMiddleware)
		if !ok {
			return nil, fmt.Errorf("auth: unexpected value type %T", value)
		}
		return auth.New(cfg)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router, err := proxy.NewRouter(ctx, cfg, logger, registry)
	if err != nil {
		logger.Error("failed to initialize router", "error", err)
		return
	}

	router.Use(logging.New(logger))

	srvConfig := server.Config{
		Host:         cfg.Server.Host,
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	if err := server.Start(srvConfig, router, logger); err != nil {
		logger.Error("server error", "error", err)
	}
}
