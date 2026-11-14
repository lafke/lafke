# Lafke

Lafke records the trust decision for each dependency in `lafke.lock`, fails CI
when that state drifts, and analyzes your own source code for vulnerabilities, committed
secrets and unsafe infrastructure settings. Analysis runs locally; your source code is
never uploaded.

- **Dependency trust**: `lock` records who and what you trust, `verify` fails when it drifts.
- **Vulnerabilities**: `audit` checks dependencies against OSV and ranks findings by severity.
- **Source code**: `sast` finds vulnerabilities, secrets and unsafe infrastructure settings.
- **One report**: `security` combines all of the above; output as text, JSON, SARIF, JUnit
  or Markdown.

## Install

```sh
curl -fsSL https://lafke.sh/install | sh
# a specific version
curl -fsSL https://lafke.sh/install | sh -s -- --version v0.1.1
```

The installer supports macOS and Linux (amd64 and arm64), checks the SHA-256 of the
download, verifies its GitHub attestation, and installs to `/usr/local/bin` or, if that is
not writable, `~/.local/bin`. Set `INSTALL_DIR` to choose another directory.

Attestation verification needs the [GitHub CLI](https://cli.github.com) (`gh`), signed in
with `gh auth login`; without it the installer stops. Pass `--skip-attestation` to install
without verifying. Windows builds are published as zip archives on the
[releases page](https://github.com/lafke/lafke/releases).

Each release archive holds two executables that belong together:

- `lafke`: the command-line interface (this repository).
- `lafke-engine`: the source-code analysis engine that `lafke` starts for
  `sast`, `security`, `chain` and related commands.

`lafke` looks for the engine in the directory of its own executable, then on `PATH`.
Set `LAFKE_ENGINE` to use an engine at another location. Commands that do not analyze
source code, such as `scan`, `lock` and `verify`, work without the engine.

## Usage

```sh
lafke scan .                 # dependency trust report
lafke lock .                 # write lafke.lock
lafke verify .               # fail when the lock drifts
lafke audit .                # dependency vulnerabilities by severity
lafke sast .                 # find vulnerabilities in your source
lafke security . --format sarif
```

Run `lafke --help` for all commands, and `--offline` to skip remote lookups.
Exit code 0 means success, 1 a trust violation or finding, 2 an operational error.
[`examples/`](examples/) walks the lock, update and verify loop for each package manager.

### In CI

```yaml
- uses: lafke/lafke@v0.1.0
  with:
    command: security
```

A [pre-commit](https://pre-commit.com) hook is provided in
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml).

## Uninstall

```sh
lafke uninstall
```

This removes the `lafke` executable, the `lafke-engine` next to it and the
response cache. `--dry-run` lists what would go, `--yes` skips the prompt, `--keep-cache`
keeps the cache. Project files such as `lafke.lock` are never touched.

## Documentation

- [Trust model](docs/trust-model.md) and [threat model](docs/threat-model.md)
- [Lockfile format](docs/lockfile.md) and [specification](docs/specification.md)
- [Privacy](docs/privacy.md): what leaves your machine, and what does not
- [Enterprise](docs/enterprise.md)

## Development

```sh
go vet ./...
go test ./...
```

Building this repository gives you the CLI only. The engine lives in a separate,
private repository; tests that need it are skipped unless `LAFKE_ENGINE` points to a
`lafke-engine` executable. `scripts/package.sh <goos> <goarch> <version>` builds both
into a release archive when the engine is checked out in `engine/`.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request, and
[SECURITY.md](SECURITY.md) to report a vulnerability.

## License

Apache-2.0. See [LICENSE](LICENSE).
