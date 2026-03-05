package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 {
		return errors.New("server port must be greater than 0")
	}

	if len(cfg.Services) == 0 {
		return errors.New("at least one service must be defined (services)")
	}

	for i, svc := range cfg.Services {
		if err := validateService(i, svc); err != nil {
			return err
		}
	}

	return nil
}

func validateService(i int, svc ServiceConfig) error {
	if svc.Name == "" {
		return fmt.Errorf("service at index %d is missing name", i)
	}
	if svc.PathPrefix == "" {
		return fmt.Errorf("service '%s' is missing path_prefix", svc.Name)
	}
	if len(svc.Target) == 0 {
		return fmt.Errorf("service '%s' must have at least one target", svc.Name)
	}

	for j, targetURL := range svc.Target {
		if err := validateTargetURL(svc.Name, j, targetURL); err != nil {
			return err
		}
	}

	if err := validateMiddlewares(svc.Name, svc.Middlewares); err != nil {
		return err
	}

	return nil
}

func validateTargetURL(svcName string, j int, targetURL string) error {
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL '%s' in service '%s' (index %d): %w", targetURL, svcName, j, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("target URL '%s' in service '%s' (index %d) has unsupported scheme '%s': only http and https are allowed",
			targetURL, svcName, j, parsed.Scheme)
	}

	return nil
}

func validateMiddlewares(svcName string, middlewares []MiddlewareConfig) error {
	for _, mw := range middlewares {
		switch cfg := mw.Value.(type) {
		case RateLimitMiddleware:
			if err := validateRateLimitMiddleware(svcName, cfg); err != nil {
				return err
			}
		case AuthMiddleware:
			if err := validateAuthMiddleware(svcName, cfg); err != nil {
				return err
			}
		default:
			return fmt.Errorf("service '%s' has unknown middleware type: %T", svcName, mw.Value)
		}
	}
	return nil
}

func validateRateLimitMiddleware(svcName string, cfg RateLimitMiddleware) error {
	if cfg.Rate <= 0 {
		return fmt.Errorf("service '%s': rate_limit.rate must be greater than 0", svcName)
	}
	if cfg.Burst <= 0 {
		return fmt.Errorf("service '%s': rate_limit.burst must be greater than 0", svcName)
	}
	return nil
}

func validateAuthMiddleware(svcName string, cfg AuthMiddleware) error {
	if cfg.PublicKeyPath == "" {
		return fmt.Errorf("service '%s': auth.public_key_path is required", svcName)
	}
	keyData, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("service '%s': auth.public_key_path: %w", svcName, err)
	}
	if _, err := jwt.ParseRSAPublicKeyFromPEM(keyData); err != nil {
		return fmt.Errorf("service '%s': auth.public_key_path is not a valid RSA public key PEM: %w", svcName, err)
	}
	if cfg.Issuer == "" {
		return fmt.Errorf("service '%s': auth.issuer is required", svcName)
	}
	if cfg.Audience == "" {
		return fmt.Errorf("service '%s': auth.audience is required", svcName)
	}
	return nil
}
