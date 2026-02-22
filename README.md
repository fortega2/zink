# Zink

Zink is a lightweight API Gateway written in Go (1.25+) that acts as a reverse proxy, configurable entirely via YAML.

## Features

- **YAML Configuration**: Easily define server settings and routing rules in a single configuration file.
- **Reverse Proxy**: Built on top of Go's robust `httputil.NewSingleHostReverseProxy`.
- **Prefix-Based Routing**: Match incoming requests by exact path prefixes and route them to specific backend services.
- **Structured Logging**: Includes standard structured logging using `log/slog`.
- **Extensible Middleware**: Architecture designed to support custom middlewares (authentication, rate limiting, logging).

## Prerequisites

- Go 1.25.6 or higher

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
```

## Configuration

Zink is configured using a YAML file. By default, it looks for `zink.yml`.

```yaml
server:
  port: 8080               # Port to listen on (default 80 if not specified, 8080 in dev)
  host: localhost
  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s

services:
  - name: "user-service"
    path_prefix: "/api/v1/users" # Requests starting with this prefix will be routed here
    target:
      - "http://localhost:8081"  # Backend service URL
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
├── internal/           # Private packages
│   ├── config/         # YAML configuration loading and validation
│   ├── proxy/          # Reverse proxy logic
│   └── middleware/     # Middlewares (auth, rate limiting, logging)
├── zink.yml            # Example configuration file
└── README.md           # This file
```

## Contributing

Please refer to the `AGENTS.md` file for project-specific conventions, code style guidelines, and detailed build/test instructions.
