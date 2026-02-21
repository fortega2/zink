package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/proxy"
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
		os.Exit(1)
	}

	logger.Info("configuration loaded successfully", "port", cfg.Server.Port, "host", cfg.Server.Host)

	for _, svc := range cfg.Services {
		logger.Info("service registered", "name", svc.Name, "path_prefix", svc.PathPrefix, "targets", len(svc.Target))
	}

	router, err := proxy.NewRouter(cfg)
	if err != nil {
		logger.Error("failed to initialize router", "error", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("zink Gateway is running", "address", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server crashed", "error", err)
		os.Exit(1)
	}
}
