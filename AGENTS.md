# AGENTS.md

Instructions for AI coding agents working in this repository. Humans: read
[CONTRIBUTING.md](CONTRIBUTING.md), which this file builds on. If the two disagree,
CONTRIBUTING.md wins; if a user's request disagrees with this file, the user wins.

## Project

Lafke is a Go CLI that records the _trust decision_ for each dependency in
`lafke.lock`, fails CI when that state drifts, and analyzes source code for
vulnerabilities, secrets and unsafe infrastructure settings. Pre-1.0, Go version in
`go.mod`, no CGO. Docs: https://lafke.dev

Source-code analysis is not part of this repository. It runs in a separate executable,
`lafke-engine`, which the CLI starts through `internal/engine`. The CLI and the engine
speak a versioned JSON protocol; see `ProtocolVersion` in `internal/engine/engine.go`.

## Layout

- `cmd/lafke`: entrypoint. `internal/cli`: one cobra command per file, registered in
  `root.go`.
- `internal/engine`: client for the analysis engine (`Analyze`, `SecretContents`, `Calls`).
  `internal/engine/enginetest.Require` skips a test when no engine is available.
- `internal/ecosystem/<resolver>`: one lockfile parser per package manager.
  `internal/evidence`, `internal/osv`: registry and OSV lookups.
- `internal/{scanner,trust,audit,doctor,finding,exceptions,sbom,vex,license,typosquat,reachability}`:
  the logic behind the commands.
- `pkg/lockfile`, `pkg/policy`, `pkg/exceptions`: public types and validation.
- `schemas/`: JSON schemas, embedded and used to validate command output.
- `docs/`, `examples/`, `action/` + `action.yml`, `scripts/install.sh`.

## Commands

Run these from the repository root. CI runs the same checks, so a failure here is a
failure there.

```bash
gofmt -l cmd internal pkg schemas        # must print nothing
go vet ./...
go test ./...
go test -race ./...
go mod tidy && git diff --exit-code      # no dependency drift
bash action/run_test.sh                  # when action/ or action.yml changed
GOOS=windows go vet ./...                # CI smoke-tests Windows
for d in examples/*/; do go run ./cmd/lafke verify --offline "$d"; done
```

Tests that need the analysis engine are skipped unless `LAFKE_ENGINE` points to a
built `lafke-engine`. Build one from a checkout of the engine repository:
`go build -o /tmp/lafke-engine ./cmd/lafke-engine`.

## Before you commit

Do not hand a failure to CI. Run the checks above that apply to your change, fix
everything they report, and only then commit. Format and `go vet` are always required. If
a check cannot run (no network, missing tool), say so in your final message instead of
committing as if it passed.

## Code style

- Match the surrounding code: naming, comment density, idiom.
- Keep changes small and focused. No drive-by refactors.
- Code, comments, commit messages, and everything published to GitHub are in English.
- A change to a request or response of the engine protocol bumps `ProtocolVersion` here
  and `protocolVersion` in the engine together.
