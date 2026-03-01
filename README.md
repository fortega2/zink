# Zink

Zink is a lightweight API Gateway written in Go (1.26+) that acts as a reverse proxy, configurable entirely via YAML.

## Features

- **YAML Configuration**: Easily define server settings and routing rules in a single configuration file.
- **Reverse Proxy**: Built on top of Go's robust `httputil.NewSingleHostReverseProxy`.
- **Prefix-Based Routing**: Match incoming requests by exact path prefixes and route them to specific backend services.
- **Round-Robin Load Balancing**: Distribute traffic across multiple backend instances with atomic, lock-free round-robin selection.
- **Structured Logging**: Includes standard structured logging using `log/slog`.
- **Extensible Middleware**: Architecture designed to support custom middlewares (authentication, rate limiting, logging).
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals, waiting up to 5 seconds for in-flight requests to complete.

## Prerequisites

- Go 1.26.0 or higher

## Getting Started

### Running the application

By default, Zink looks for a `zink.yml` configuration file in the current working directory.

```bash
# Run the application (development)
go run ./cmd/zink/main.go
```

To run with a custom configuration file:

```bash
go run ./cmd/zink/main.go --config=path/to/custom_config.yml
```

### Building the binary

```bash
go build -o zink ./cmd/zink/main.go
./zink

# Run with a custom config file
./zink --config=path/to/config.yml
```

## Configuration

Zink is configured using a YAML file. By default, it looks for `zink.yml`.

```yaml
server:
  port: 8080               # Required — startup fails if missing or zero
  host: localhost          # Defaults to 0.0.0.0 if not specified
  read_timeout: 15s        # Default: 15s
  write_timeout: 15s       # Default: 15s
  idle_timeout: 60s        # Default: 60s

services:
  - name: "user-service"
    path_prefix: "/api/v1/users"   # Requests with this prefix are routed here
    load_balancer: "round_robin"   # Optional; defaults to round_robin if omitted
    target:
      - "http://localhost:8081"    # Backend instance 1
      - "http://localhost:8082"    # Backend instance 2
```

## Development

The project uses several tools to ensure code quality:

### Useful Commands

```bash
# Run all tests
go test ./...

# Run linter (golangci-lint)
golangci-lint run

# Check security vulnerabilities
govulncheck ./...

# Format code
go fmt ./...
```

### Pre-commit hooks

This project uses `lefthook` for pre-commit checks. It automatically runs tests, linting, and vulnerability checks on every commit.

```bash
# Install hooks (only needed once)
lefthook install

# Run hooks manually
lefthook run pre-commit
```

## Project Structure

```
zink/
├── cmd/zink/           # Application entry point
│   └── main.go
├── docker/             # Container build files
│   └── Dockerfile      # Multi-stage build (golang:1.26.0-alpine3.23 → scratch)
├── internal/           # Private packages
│   ├── config/         # YAML configuration loading and validation
│   ├── middleware/     # Middlewares (logging; extensible for auth, rate limiting)
│   ├── proxy/          # Reverse proxy logic and load balancing
│   └── server/         # HTTP server with graceful shutdown
├── zink.yml            # Example configuration file
└── README.md           # This file
```

## Docker

A multi-stage `Dockerfile` is included in `docker/Dockerfile`.

```bash
# Build the Docker image
docker build -f docker/Dockerfile -t zink .

# Run with a config file
docker run -v $(pwd)/zink.yml:/zink.yml zink --config=/zink.yml
```

## Contributing

Please refer to the `AGENTS.md` file for project-specific conventions, code style guidelines, and detailed build/test instructions.
