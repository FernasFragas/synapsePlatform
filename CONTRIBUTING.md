Here's what a `CONTRIBUTING.md` would look like, tailored to your project's actual tooling, conventions, and architecture:

---

# Contributing to Synapse Platform

## Before You Start

1. Read the [README](Readme.md) and run the project locally.
2. Check existing issues and PRs to avoid duplicating work.
3. For non-trivial changes, open an issue first to discuss the approach.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | [go.dev/doc/install](https://go.dev/doc/install) |
| Docker + Compose | any recent | [docs.docker.com](https://docs.docker.com/get-docker/) |
| golangci-lint | v2.4.0+ | `make install-tools` |
| mockgen | latest | `make install-tools` |

## Development Workflow

```bash
# 1. Fork and clone the repository
git clone <your-fork>
cd synapsePlatform

# 2. Create a feature branch from main
git checkout -b <category>/<short-description>

# 3. Install tools
make install-tools

# 4. Make your changes, then validate
make fmt
make lint
make test

# 5. Run the full local CI pipeline before pushing
./test-ci.sh
```

## Branch Naming

Use a prefix that describes the type of change:

| Prefix | Use for |
|--------|---------|
| `feat/` | New functionality |
| `fix/` | Bug fixes |
| `perf/` | Performance improvements |
| `refactor/` | Code restructuring with no behavior change |
| `test/` | Adding or improving tests |
| `docs/` | Documentation only |
| `ci/` | CI/CD and build tooling |

Examples: `feat/websocket-events`, `perf/batch-inserts`, `fix/cursor-pagination-off-by-one`

## Pull Request Rules

### Every PR must

- **Target `main`** and branch from the latest `main`.
- **Pass CI** -- lint, test, and build must all succeed.
- **Be focused** -- one logical change per PR. Don't mix a feature with an unrelated refactor.
- **Include tests** for any new or changed behavior. Aim to maintain or increase coverage.
- **Not break existing tests** -- run `make test` locally before pushing.
- **Follow existing patterns** -- this project uses decorator-based composition (logging, metrics, failure handling). New components should follow the same layering.

### Commit messages

Write commits in imperative mood. 

e.g: Add cursor-based pagination for event listing

### PR description

Use this template:


## Summary
<!-- 1-3 bullet points: what changed and why -->

## Code Guidelines

### Architecture

Every pipeline component follows the **decorator pattern** through interfaces:

core implementation -> log decorator -> metrics decorator

When adding a new component:
1. Define the interface in `internal/ingestor/` (or the relevant domain package).
2. Write the core implementation.
3. Add a logging decorator in `internal/log/`.
4. Add a metrics decorator in `internal/metrics/` with OpenTelemetry spans and counters.
5. Wire it in `cmd/main.go`.

### Testing

- **Unit tests** live next to the code (`foo_test.go` in the same package, or `foo_test` for black-box).
- **Integration tests** use the `//go:build integration` build tag and go in a `_integration_test.go` file.
- **Mocks** are generated with `go.uber.org/mock`. Add a `//go:generate mockgen` directive to the interface file, then run `make generate`.
- **Test helpers** live in `internal/utilstest/`. Use the fluent builder style that exists there (`WithEvents(...)`, `WithError(...)`, etc.).

### Linting

The project uses golangci-lint v2 with **all linters enabled** by default (see `.golangci.yaml`). Notable settings:

- Max cyclomatic complexity: **9**
- Max function arguments: **4** (enforced by revive)
- Some linters are relaxed in `_test.go` files

If the linter flags something you believe is a false positive, disable it with a `//nolint` comment that includes the linter name and a reason:

```go
//nolint:errcheck
_ = writer.Flush()
```

## Performance and Benchmark Contributions

Performance work is welcome but must be evidence-driven.

### PR description for performance changes

## Summary
<!-- What was slow and why -->

## Test plan
- [ ] `make test` passes
- [ ] `go test -race ./...` passes

## Adding New Features

### Checklist

- [ ] Core implementation with unit tests
- [ ] Log decorator in `internal/log/`
- [ ] Metrics decorator in `internal/metrics/` (histogram + counter + trace span)
- [ ] Wired in `cmd/main.go` following the existing composition order
- [ ] Config fields added to `internal/config.go` and `config.yaml`
- [ ] Health probe added if the component has external dependencies
- [ ] `make generate` run if new interfaces were added
- [ ] `./test-ci.sh` passes

### What not to do

- Don't add global state. Every dependency is injected.
- Don't import `internal/log` or `internal/metrics` from core packages (`internal/ingestor`, `internal/kafka`, `internal/sqllite`). Decorators wrap; they don't get imported by the thing they decorate.
- Don't commit `data.db`, `coverage.out`, or binary artifacts.

## Review Process

1. A maintainer will review your PR within a few days.
2. Address review comments by pushing new commits (don't force-push during review).
3. Once approved, the maintainer squash-merges into `main`.
```