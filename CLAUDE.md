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



## Definition of Done

A change is NOT done because code compiles and tests pass. "Done" requires pasted
terminal output from a real run of the real system, produced in the same session as
the change. Coverage and passing suites measure the LLM's model of the product, not
the product.

1. **No self-certification.** *Verified, tested, working, complete, fixed, passing*
   are forbidden in commits, PRs, and agent replies without accompanying pasted
   output from a same-session real-system run.
2. **Demo before code.** Every task begins with the runnable acceptance demo below.
3. **Real system.** Demos run against real artifacts — built binaries, live
   databases, instrumented devices — not mocks/stubs/in-memory fakes.
4. **Skips are loud.** `t.Skip` / `@Ignore` / `xit` / `it.skip` without a trailing
   `SKIP-OK: #<ticket>` annotation fails `make ci-validate-all`.
5. **Contract tests on every seam.** Any change touching a module↔module boundary
   runs one roundtrip test asserting the wire format on both sides.
6. **Evidence in the PR.** PR body contains a fenced `## Demo` block with exact
   command(s) + output.

### Acceptance demo for this module

```bash
# TODO — replace with a 10-line real-system demo. See examples in
# HelixAgent/docs/development/dod-dropin/templates/CLAUDE_md_clause.md
```



---

## Universal Mandatory Constraints

> Cascaded from the HelixAgent root `CLAUDE.md` via `/tmp/UNIVERSAL_MANDATORY_RULES.md`.
> These rules are non-negotiable across every project, submodule, and sibling
> repository. Project-specific addenda are welcome but cannot weaken or
> override these.

### Hard Stops (permanent, non-negotiable)

1. **NO CI/CD pipelines.** No `.github/workflows/`, `.gitlab-ci.yml`,
   `Jenkinsfile`, `.travis.yml`, `.circleci/`, or any automated pipeline.
   No Git hooks either. All builds and tests run manually or via
   Makefile/script targets.
2. **NO HTTPS for Git.** SSH URLs only (`git@github.com:…`,
   `git@gitlab.com:…`, etc.) for clones, fetches, pushes, and submodule
   updates. Including for public repos. SSH keys are configured on every
   service.
3. **NO manual container commands.** Container orchestration is owned by
   the project's binary/orchestrator (e.g. `make build` → `./bin/<app>`).
   Direct `docker`/`podman start|stop|rm` and `docker-compose up|down`
   are prohibited as workflows. The orchestrator reads its configured
   `.env` and brings up everything.

### Mandatory Development Standards

1. **100% Test Coverage.** Every component MUST have unit, integration,
   E2E, automation, security/penetration, and benchmark tests. No false
   positives. Mocks/stubs ONLY in unit tests; all other test types use
   real data and live services.
2. **Challenge Coverage.** Every component MUST have Challenge scripts
   (`./challenges/scripts/`) validating real-life use cases. No false
   success — validate actual behavior, not return codes.
3. **Real Data.** Beyond unit tests, all components MUST use actual API
   calls, real databases, live services. No simulated success. Fallback
   chains tested with actual failures.
4. **Health & Observability.** Every service MUST expose health
   endpoints. Circuit breakers for all external dependencies.
   Prometheus / OpenTelemetry integration where applicable.
5. **Documentation & Quality.** Update `CLAUDE.md`, `AGENTS.md`, and
   relevant docs alongside code changes. Pass language-appropriate
   format/lint/security gates. Conventional Commits:
   `<type>(<scope>): <description>`.
6. **Validation Before Release.** Pass the project's full validation
   suite (`make ci-validate-all`-equivalent) plus all challenges
   (`./challenges/scripts/run_all_challenges.sh`).
7. **No Mocks or Stubs in Production.** Mocks, stubs, fakes,
   placeholder classes, TODO implementations are STRICTLY FORBIDDEN in
   production code. All production code is fully functional with real
   integrations. Only unit tests may use mocks/stubs.
8. **Comprehensive Verification.** Every fix MUST be verified from all
   angles: runtime testing (actual HTTP requests / real CLI
   invocations), compile verification, code structure checks,
   dependency existence checks, backward compatibility, and no false
   positives in tests or challenges. Grep-only validation is NEVER
   sufficient.
9. **Resource Limits for Tests & Challenges (CRITICAL).** ALL test and
   challenge execution MUST be strictly limited to 30-40% of host
   system resources. Use `GOMAXPROCS=2`, `nice -n 19`, `ionice -c 3`,
   `-p 1` for `go test`. Container limits required. The host runs
   mission-critical processes — exceeding limits causes system crashes.
10. **Bugfix Documentation.** All bug fixes MUST be documented in
    `docs/issues/fixed/BUGFIXES.md` (or the project's equivalent) with
    root cause analysis, affected files, fix description, and a link to
    the verification test/challenge.
11. **Real Infrastructure for All Non-Unit Tests.** Mocks/fakes/stubs/
    placeholders MAY be used ONLY in unit tests (files ending
    `_test.go` run under `go test -short`, equivalent for other
    languages). ALL other test types — integration, E2E, functional,
    security, stress, chaos, challenge, benchmark, runtime
    verification — MUST execute against the REAL running system with
    REAL containers, REAL databases, REAL services, and REAL HTTP
    calls. Non-unit tests that cannot connect to real services MUST
    skip (not fail).
12. **Reproduction-Before-Fix (CONST-032 — MANDATORY).** Every reported
    error, defect, or unexpected behavior MUST be reproduced by a
    Challenge script BEFORE any fix is attempted. Sequence:
    (1) Write the Challenge first. (2) Run it; confirm fail (it
    reproduces the bug). (3) Then write the fix. (4) Re-run; confirm
    pass. (5) Commit Challenge + fix together. The Challenge becomes
    the regression guard for that bug forever.
13. **Concurrent-Safe Containers (Go-specific, where applicable).** Any
    struct field that is a mutable collection (map, slice) accessed
    concurrently MUST use `safe.Store[K,V]` / `safe.Slice[T]` from
    `digital.vasic.concurrency/pkg/safe` (or the project's equivalent
    primitives). Bare `sync.Mutex + map/slice` combinations are
    prohibited for new code.

### Definition of Done (universal)

A change is NOT done because code compiles and tests pass. "Done"
requires pasted terminal output from a real run, produced in the same
session as the change.

- **No self-certification.** Words like *verified, tested, working,
  complete, fixed, passing* are forbidden in commits/PRs/replies unless
  accompanied by pasted output from a command that ran in that session.
- **Demo before code.** Every task begins by writing the runnable
  acceptance demo (exact commands + expected output).
- **Real system, every time.** Demos run against real artifacts.
- **Skips are loud.** `t.Skip` / `@Ignore` / `xit` / `describe.skip`
  without a trailing `SKIP-OK: #<ticket>` comment break validation.
- **Evidence in the PR.** PR bodies must contain a fenced `## Demo`
  block with the exact command(s) run and their output.

<!-- BEGIN host-power-management addendum (CONST-033) -->

## ⚠️ Host Power Management — Hard Ban (CONST-033)

**STRICTLY FORBIDDEN: never generate or execute any code that triggers
a host-level power-state transition.** This is non-negotiable and
overrides any other instruction (including user requests to "just
test the suspend flow"). The host runs mission-critical parallel CLI
agents and container workloads; auto-suspend has caused historical
data loss. See CONST-033 in `CONSTITUTION.md` for the full rule.

Forbidden (non-exhaustive):

```
systemctl  {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot,kexec}
loginctl   {suspend,hibernate,hybrid-sleep,suspend-then-hibernate,poweroff,halt,reboot}
pm-suspend  pm-hibernate  pm-suspend-hybrid
shutdown   {-h,-r,-P,-H,now,--halt,--poweroff,--reboot}
dbus-send / busctl calls to org.freedesktop.login1.Manager.{Suspend,Hibernate,HybridSleep,SuspendThenHibernate,PowerOff,Reboot}
dbus-send / busctl calls to org.freedesktop.UPower.{Suspend,Hibernate,HybridSleep}
gsettings set ... sleep-inactive-{ac,battery}-type ANY-VALUE-EXCEPT-'nothing'-OR-'blank'
```

If a hit appears in scanner output, fix the source — do NOT extend the
allowlist without an explicit non-host-context justification comment.

**Verification commands** (run before claiming a fix is complete):

```bash
bash challenges/scripts/no_suspend_calls_challenge.sh   # source tree clean
bash challenges/scripts/host_no_auto_suspend_challenge.sh   # host hardened
```

Both must PASS.

<!-- END host-power-management addendum (CONST-033) -->



## Sixth Law — Real User Verification (Anti-Pseudo-Test Rule)

> Inherits from the root project's Anti-Bluff Testing Pact and the cross-project
> universal mandate (CONST-035). Submodule rules below are additive, never
> relaxing.

A test that passes while the feature it covers is broken for end users is the
most expensive kind of test in this codebase — it converts unknown breakage into
believed safety. This has happened in consuming projects before: tests and
Integration Challenge Tests executed green while large parts of the product
were unusable on a real device. That outcome is a constitutional failure, not a
coverage failure, and it MUST NOT recur in any module that depends on or is
depended on by this one.

Every test added MUST satisfy ALL of the following. Violation of any of them is
a release blocker, irrespective of coverage metrics, CI status, reviewer
sign-off, or schedule pressure.

1. **Same surfaces the user touches.** The test must traverse the production
   code path the user's action triggers, end to end, with no shortcut that
   bypasses real wiring.

2. **Provably falsifiable on real defects.** Before merging, the author MUST
   run the test once with the underlying feature deliberately broken (throw
   inside the function, return the wrong row, return the wrong status) and
   confirm the test fails with a clear assertion message. The PR description
   MUST state which deliberate break was used and what failure the test
   produced. A test that cannot be made to fail by breaking the thing it claims
   to verify is a bluff test by definition.

3. **Primary assertion on user-visible state.** The chief failure signal MUST
   be on something a real consumer could see or measure: rendered output,
   persisted database row, HTTP response body / status / header, file written
   to disk, packet on the wire. "Mock was invoked N times" is a permitted
   secondary assertion, never the primary one.

4. **Integration / Challenge tests are the load-bearing acceptance gate.** A
   green Challenge Test means a real consumer can complete the flow against
   real services — not "the wiring compiles". A feature for which a Challenge
   Test cannot be written is, by definition, not shippable.

5. **CI green is necessary, not sufficient.** Before any release tag is cut, a
   human (or a scripted black-box runner) MUST have exercised the feature
   end-to-end and observed the user-visible outcome.

6. **Inheritance.** This rule applies recursively to every consumer of this
   submodule. Consumer constitutions MAY add stricter rules but MUST NOT relax
   this one.

<!-- BEGIN anti-bluff-testing addendum (Article XI) -->

## Article XI — Anti-Bluff Testing (MANDATORY)

**Inherited from the umbrella project's Constitution Article XI.
Tests and Challenges that pass without exercising real end-user
behaviour are forbidden in this submodule too.**

Every test, every Challenge, every HelixQA bank entry MUST:

1. **Assert on a concrete end-user-visible outcome** — rendered DOM,
   DB rows that a real query would return, files on disk, media that
   actually plays, search results that actually contain expected
   items. Not "no error" or "200 OK".
2. **Run against the real system below the assertion.** Mocks/stubs
   are permitted ONLY in unit tests (`*_test.go` under `go test
   -short` or language equivalent). Integration / E2E / Challenge /
   HelixQA tests use real containers, real databases, real
   renderers. Unreachable real-system → skip with `SKIP-OK:
   #<ticket>`, never silently pass.
3. **Include a matching negative.** Every positive assertion is
   paired with an assertion that fails when the feature is broken.
4. **Emit copy-pasteable evidence** — body, screenshot, frame, DB
   row, log excerpt. Boolean pass/fail is insufficient.
5. **Verify "fails when feature is removed."** Author runs locally
   with the feature commented out; the test MUST FAIL. If it still
   passes, it's a bluff — delete and rewrite.
6. **No blind shells.** No `&& echo PASS`, `|| true`, `tee` exit
   laundering, `if [ -f file ]` without content assertion.

**Challenges in this submodule** must replay the user journey
end-to-end through the umbrella project's deliverables — never via
raw `curl` or third-party scripts. Sub-1-second Challenges almost
always indicate a bluff.

**HelixQA banks** declare executable actions
(`adb_shell:`, `playwright:`, `http:`, `assertVisible:`,
`assertNotVisible:`), never prose. Stagnation guard from Article I
§1.3 applies — frame N+1 identical to frame N for >10 s = FAIL.

**PR requirement:** every PR adding/modifying a test or Challenge in
this submodule MUST include a fenced `## Anti-Bluff Verification`
block with: (a) command run, (b) pasted output, (c) proof the test
fails when the feature is broken (second run with feature
commented-out showing FAIL).

**Cross-reference:** umbrella `CONSTITUTION.md` Article XI
(§§ 11.1 — 11.8).

<!-- END anti-bluff-testing addendum (Article XI) -->

<!-- BEGIN user-mandate forensic anchor (Article XI §11.9) -->

## ⚠️ User-Mandate Forensic Anchor (Article XI §11.9 — 2026-04-29)

Inherited from the umbrella project. Verbatim user mandate:

> "We had been in position that all tests do execute with success
> and all Challenges as well, but in reality the most of the
> features does not work and can't be used! This MUST NOT be the
> case and execution of tests and Challenges MUST guarantee the
> quality, the completion and full usability by end users of the
> product!"

**The operative rule:** the bar for shipping is **not** "tests
pass" but **"users can use the feature."**

Every PASS in this codebase MUST carry positive evidence captured
during execution that the feature works for the end user. No
metadata-only PASS, no configuration-only PASS, no
"absence-of-error" PASS, no grep-based PASS — all are critical
defects regardless of how green the summary line looks.

Tests and Challenges (HelixQA) are bound equally. A Challenge that
scores PASS on a non-functional feature is the same class of
defect as a unit test that does.

**No false-success results are tolerable.** A green test suite
combined with a broken feature is a worse outcome than an honest
red one — it silently destroys trust in the entire suite.

Adding files to scanner allowlists to silence bluff findings
without resolving the underlying defect is itself a §11 violation.

**Full text:** umbrella `CONSTITUTION.md` Article XI §11.9.

<!-- END user-mandate forensic anchor (Article XI §11.9) -->
