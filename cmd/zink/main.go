package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/fortega2/zink/internal/config"
)

func main() {
	configPath := flag.String("config", "zink.yml", "path to the configuration file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger.Info("Starting Zink...", "configPath", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully", "port", cfg.Server.Port, "host", cfg.Server.Host)

	for _, svc := range cfg.Services {
		logger.Info("Service registered", "name", svc.Name, "path_prefix", svc.PathPrefix, "targets", len(svc.Target))
	}
}
