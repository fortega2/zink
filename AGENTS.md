# Zink - AGENTS.md

Zink is an API Gateway written in Go 1.26.0 that acts as a reverse proxy
configurable via YAML. This document establishes the norms and conventions
for developers (humans and agents) working on this project.

## 1. Build, Lint, and Test Commands

### Main commands

```bash
# Run the application (development)
go run ./cmd/zink/main.go

# Run with custom config
go run ./cmd/zink/main.go --config=path/to/config.yml

# Build binary
go build -o zink ./cmd/zink/main.go

# Run all tests
go test ./...

# Run short tests (used in pre-commit)
go test -short ./...

# Run a single test
go test -v ./path/to/package -run TestName

# Run linter (golangci-lint)
golangci-lint run

# Check security vulnerabilities
govulncheck ./...

# Format code
go fmt ./...

# Verify static errors
go vet ./...
```

### Pre-commit hooks

The project uses lefthook. Hooks run automatically on every commit:

```bash
# Install hooks (only needed once)
lefthook install

# Run hooks manually
lefthook run pre-commit
```

Pre-commit hooks run in parallel:
1. `golangci-lint run` (linter)
2. `govulncheck ./...` (security)
3. `go test -short ./...` (tests)

## 2. Code Style and Conventions

### General

- Go version: 1.26.0 (use only features available up to this version)
- Maximum lines per function: linter guideline (gocyclo: 15 recommended)
- Maximum cyclomatic complexity: linter guideline (gocognit: 30)
- Avoid files larger than 300 lines when possible
- Never write comments in code
- The application logger uses `slog.NewJSONHandler` writing to stdout at `slog.LevelDebug`
- Tests use `slog.New(slog.DiscardHandler)` to silence logging output (Go 1.24+ feature)

### Imports

Sort imports in three groups (gofmt/goimports standard):

1. Standard Go packages (`net/http`, `fmt`, `os`, etc.)
2. Third-party external packages (`gopkg.in/yaml.v3`, `github.com/...`)
3. Internal project packages (`github.com/fortega2/zink/internal/...`)

Example:

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/fortega2/zink/internal/config"
)
```

Do not use import aliases unless necessary to avoid name collisions.

### Types and Declarations

- Use concrete types over interfaces unless polymorphism is needed
- Prefer embedded `structs` over explicit composition when there is no name collision
- Use pointers (`*Type`) only when modifying the value or when nil has semantic meaning
- Declare variables close to their first use

Example:

```go
// Correct
func process(data []byte) error {
    cfg := &Config{} // Pointer because it is modified
    if err := json.Unmarshal(data, cfg); err != nil {
        return fmt.Errorf("unmarshal failed: %w", err)
    }
    return nil
}

// Avoid if pointer is not necessary
func sum(a, b int) int {
    return a + b
}
```

### Naming Conventions

- **Packages:** short names, lowercase, no underscores. E.g.: `config`, `proxy`, `middleware`
- **Files:** snake_case.go or descriptive component names. E.g.: `load.go`, `router.go`
- **Variables and Functions:** camelCase
- **Constants:** Uppercase if exported, lowercase if not. E.g.: `MaxRetries`, `defaultTimeout`
- **Interfaces:** Name + `er` suffix when simple. E.g.: `Reader`, `Handler`, `Proxy`
- **Errors:** Prefix with "Err" for exported errors. E.g.: `ErrInvalidConfig`

### Error Handling

- DO NOT use `_` to discard errors unless explicitly intentional
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use `errors.Is()` and `errors.As()` for error inspection
- Prefer errors declared as variables/constants over literal errors
- Do not create errors with `errors.New("literal string")` inside functions that run in hot loops

Example:

```go
var ErrServiceNotFound = errors.New("service not found")

func (g *Gateway) GetService(path string) (*Service, error) {
    svc, ok := g.services[path]
    if !ok {
        return nil, fmt.Errorf("path %s: %w", path, ErrServiceNotFound)
    }
    return svc, nil
}
```

- **Logs:** Use `log/slog` for structured logs. DO NOT use `log.Print*`
- Prefer `slog.Error` with "error" in the context field, or `slog.Warn` for non-fatal warnings

### Context and Concurrency

- The first argument of functions that may be slow or blocking must be `context.Context`
- Use `context.Background()` or `context.TODO()` at the entry point (main)
- Prefer `sync/atomic` over mutexes for simple counters
- Use `sync.WaitGroup` to coordinate goroutines
- Never launch goroutines without control: ensure there is a shutdown mechanism (graceful shutdown)

### Documentation

- Document all exported functions and types with Go-style comments
- The comment should start with the name of the element being documented
- Document the behavior, not the implementation

```go
// Config represents the root configuration of the API Gateway.
// It is loaded from a YAML file using config.Load().
type Config struct {
    // Server contains the HTTP server configuration.
    Server ServerConfig `yaml:"server"`
    // Services defines the list of backend services.
    Services []ServiceConfig `yaml:"services"`
}
```

### Testing

- Name test files as `name_test.go`
- Use subtests for related test cases: `t.Run("test case name", func(t *testing.T) {...})`
- Tests must be independent of each other
- Prefer table-driven tests when there are multiple input/output cases
- Test names should be formatted with Pascal case "TestName()"
- Use `github.com/stretchr/testify` for assertions: `assert` for non-fatal checks and `require` for checks that should abort the test immediately if they fail

```go
func TestLoad(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        wantErr bool
    }{
        {"valid file", "testdata/valid.yml", false},
        {"nonexistent file", "noexiste.yml", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := config.Load(tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## 3. Project Structure

```
zink/
├── cmd/zink/                   # Application entry point
│   └── main.go
├── docker/                     # Container build files
│   └── Dockerfile              # Multi-stage build (golang:1.26.0-alpine3.23 → scratch)
├── internal/                   # Private packages (not externally importable)
│   ├── balancer/               # Round-robin load balancing director (NewDirector, RoundRobin)
│   ├── config/                 # YAML loading, validation, and type definitions
│   │   ├── load.go             # Load() + default constants
│   │   ├── validate.go         # All validate*() functions
│   │   ├── middleware.go       # MiddlewareConfig, MiddlewareType, UnmarshalYAML
│   │   ├── auth.go             # AuthMiddleware
│   │   ├── rate_limit.go       # RateLimitMiddleware
│   │   ├── server.go           # ServerConfig
│   │   ├── service.go          # ServiceConfig
│   │   └── config.go           # Root Config type
│   ├── middleware/             # Middleware type, Chain(), and Registry
│   │   ├── middleware.go       # Middleware type + Chain()
│   │   ├── registry.go         # Registry, Factory, Entry, NewRegistry(), Register(), Build()
│   │   ├── auth/               # Static token authentication middleware
│   │   ├── logging/            # Structured request/response logging middleware
│   │   └── ratelimit/          # Token-bucket rate limiting middleware
│   ├── proxy/                  # Reverse proxy handler and request router
│   │   ├── proxy.go            # createProxy() + defaultTimeout (5s)
│   │   └── router.go           # NewRouter(), applyServiceMiddlewares(), Use()
│   └── server/                 # HTTP server lifecycle and graceful shutdown
│       └── server.go
├── zink.yml                    # Example configuration file
├── go.mod
├── go.sum
├── .golangci.yml               # Linter configuration (golangci-lint v2)
├── lefthook.yml                # Pre-commit hooks configuration
└── AGENTS.md                   # This file
```

## 4. Project-Specific Rules

### Configuration

- Default configuration file is `zink.yml` in the current directory
- Support `--config` flag to specify custom path
- Port is required — a missing or zero port causes a startup error
- Default host is `0.0.0.0` if not specified
- Default timeouts: `read_timeout: 15s`, `write_timeout: 15s`, `idle_timeout: 60s`

### Proxy and Routing

- Use `net/http/httputil.NewSingleHostReverseProxy` as base
- Implement an `http.Handler` that acts as a router (multiplexer)
- Route matching should be by exact prefix for this version
- Round-robin load balancing is fully implemented. Configure with `load_balancer: round_robin` in each service. If omitted, the router defaults to `round_robin` and logs a warning. The field accepts the string `"round_robin"` (constant `config.LoadBalancerRoundRobin`)
- Each proxied request is wrapped with a `context.WithTimeout` of 5 seconds (`defaultTimeout` constant in `proxy/proxy.go`). Backends that do not respond within this window receive a `502 Bad Gateway`

### Server

- The server supports graceful shutdown triggered by SIGINT or SIGTERM
- On receiving a signal, the server stops accepting new connections and waits up to 5 seconds for in-flight requests to complete (`shutdownServerTimeout` in `server/server.go`)

### Middlewares

- Implement using the standard pattern `func(http.Handler) http.Handler`
- Include a structured logger (slog) by default in all requests (applied globally via `router.Use`)
- Validate auth and rate limiting configuration at load time, not per request
- Each middleware lives in its own sub-package under `internal/middleware/` (e.g. `auth`, `logging`, `ratelimit`)
- Register middleware factories in `cmd/zink/main.go` using `middleware.Registry.Register()` at startup
- `Registry.Build` returns an error for unregistered types — never silently skips them
- `MiddlewareConfig.TypeName` is populated during YAML unmarshalling; no type switch on `Value` is needed outside of `config/`

### Docker

- A `Dockerfile` is provided in `docker/Dockerfile`
- Uses a multi-stage build: builder stage on `golang:1.26.0-alpine3.23`, final image is `scratch`
- Build: `docker build -f docker/Dockerfile -t zink .`

## 5. Enabled Linters (Reference)

The project uses golangci-lint v2 (`version: "2"` in `.golangci.yml`) with the following rules enabled, among others:
gosec, bodyclose, noctx, revive, gocritic, misspell, whitespace,
unconvert, tagalign, predeclared, modernize, sloglint, usestdlibvars,
perfsprint, prealloc, errorlint, errname, nilnil, nilerr, nestif,
mnd, copyloopvar, gocyclo, gocognit, ireturn, iotamixing, iface,
godoclint, funcorder, embeddedstructfieldcheck, unqueryvet.

- Always check with "golangci-lint run" if a file has a issue when it's been modify or created.
- Ensure code passes all linters before committing.
