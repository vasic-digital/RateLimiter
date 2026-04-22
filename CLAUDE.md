# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Overview

`digital.vasic.ratelimiter` is a standalone Go module providing rate limiting implementations. It offers an in-memory sliding window limiter for single-instance deployments and a Redis-backed distributed limiter for multi-instance environments. An HTTP middleware adapter is included for easy integration with any `net/http` compatible router.

## Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./... -count=1

# Run tests with verbose output
go test -v ./... -count=1

# Run tests for a specific package
go test -v ./pkg/memory/ -count=1

# Run a single test
go test -v -run TestAllowWithinLimit ./pkg/memory/
```

## Architecture

```
pkg/limiter/     - Core interfaces (Limiter, Config, Result)
pkg/sliding/     - Sliding window counter algorithm
pkg/memory/      - In-memory rate limiter (uses sliding window)
pkg/redis/       - Redis-backed distributed rate limiter (Lua script)
pkg/middleware/   - HTTP middleware adapter
```

**Data flow:** HTTP Middleware -> Limiter interface -> Memory or Redis implementation -> Sliding window algorithm

**Key design decisions:**
- The `Limiter` interface is the central abstraction; all implementations satisfy it.
- The sliding window algorithm divides time into sub-windows for smooth rate limiting.
- The Redis implementation uses an atomic Lua script to avoid race conditions.
- The HTTP middleware is fail-open: on limiter errors, requests are allowed through.

## Conventions

- Functional options pattern for Redis limiter configuration (`WithPrefix`).
- Table-driven tests throughout.
- `*_test.go` files beside source files.
- `context.Context` passed through all interface methods.
- `EffectiveBurst()` centralizes the burst-defaults-to-rate logic.


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**


