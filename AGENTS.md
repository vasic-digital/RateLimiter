# AGENTS.md

Instructions for AI agents working on this codebase.

## Module Structure

This is a pure Go library module (`digital.vasic.ratelimiter`). It has no `main` package and produces no binary. All public API is under `pkg/`.

## Key Interfaces

The `limiter.Limiter` interface is the central contract. Any new rate limiter backend must implement:
- `Allow(ctx context.Context, key string) (*Result, error)` - Check and record a request.
- `Reset(ctx context.Context, key string) error` - Clear rate limit state for a key.

## Adding a New Backend

1. Create a new package under `pkg/` (e.g., `pkg/memcached/`).
2. Implement the `limiter.Limiter` interface.
3. Add tests in the same package.
4. Ensure all tests pass: `go test ./... -count=1`.

## Testing

- In-memory tests use the `sliding` package directly.
- Redis tests use `alicebob/miniredis/v2` for a real Redis protocol without a running server.
- Middleware tests use `httptest` for HTTP handler testing.
- All tests must pass with `-count=1` (no caching).

## Dependencies

Keep dependencies minimal. Current external dependencies:
- `github.com/redis/go-redis/v9` - Redis client (only in `pkg/redis/`)
- `github.com/stretchr/testify` - Test assertions (test only)
- `github.com/alicebob/miniredis/v2` - In-process Redis (test only)


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**

