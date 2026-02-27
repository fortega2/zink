package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
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

	router, err := proxy.NewRouter(cfg)
	if err != nil {
		logger.Error("failed to initialize router", "error", err)
		return
	}

	router.Use(middleware.Logging(logger))

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
