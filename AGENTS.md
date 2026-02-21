# Zink - AGENTS.md

Zink is an API Gateway written in Go 1.25.6 that acts as a reverse proxy
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

- Go version: 1.25.6 (use only features available up to this version)
- Maximum lines per function: linter guideline (gocyclo: 15 recommended)
- Maximum cyclomatic complexity: linter guideline (gocognit: 30)
- Avoid files larger than 300 lines when possible

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
├── cmd/zink/           # Application entry point
│   └── main.go
├── internal/           # Private packages (not externally importable)
│   ├── config/         # YAML configuration loading and validation
│   ├── proxy/         # Reverse proxy logic
│   └── middleware/    # Middlewares (auth, rate limiting, logging)
├── zink.yml           # Example configuration file
├── go.mod
├── go.sum
├── .golangci.yml      # Linter configuration
├── lefthook.yml       # Pre-commit hooks configuration
└── AGENTS.md          # This file
```

## 4. Project-Specific Rules

### Configuration

- Default configuration file is `zink.yml` in the current directory
- Support `--config` flag to specify custom path
- Default port is 80 if not specified (or 8080 for development)

### Proxy and Routing

- Use `net/http/httputil.NewSingleHostReverseProxy` as base
- Implement an `http.Handler` that acts as a router (multiplexer)
- Route matching should be by exact prefix for this version
- Support multiple backends per service (round-robin balancing in future phases)

### Middlewares

- Implement using the standard pattern `func(http.Handler) http.Handler`
- Include a structured logger (slog) by default in all requests
- Validate auth and rate limiting configuration at load time, not per request

## 5. Enabled Linters (Reference)

The project uses golangci-lint with the following rules enabled, among others:
gosec, bodyclose, noctx, revive, gocritic, misspell, whitespace,
unconvert, tagalign, predeclared, modernize, sloglint, usestdlibvars,
perfsprint, prealloc, errorlint, errname, nilnil, nilerr, nestif,
mnd, copyloopvar, gocyclo, gocognit, ireturn, iotamixing, iface,
godoclint, funcorder, embeddedstructfieldcheck.

- Always check with "golangci-lint run" if a file has a issue when it's been modify or created.
- Ensure code passes all linters before committing.
