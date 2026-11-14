# Lafke

Lafke records the trust decision for each dependency in `lafke.lock`, fails CI
when that state drifts, and analyzes your own source code for vulnerabilities, committed
secrets and unsafe infrastructure settings. Analysis runs locally; your source code is
never uploaded.

## Install

```sh
curl -fsSL https://lafke.sh/install | sh
# a specific version
curl -fsSL https://lafke.sh/install | sh -s -- --version v0.1.0
```

The installer puts two executables next to each other:

- `lafke`: the command-line interface (this repository).
- `lafke-engine`: the source-code analysis engine that `lafke` starts for
  `sast`, `security`, `chain` and related commands.

`lafke` looks for the engine in the directory of its own executable, then on `PATH`.
Set `LAFKE_ENGINE` to use an engine at another location.

To remove everything the installer added, run `lafke uninstall` (`--dry-run` lists
what would go, `--yes` skips the prompt, `--keep-cache` keeps the response cache). Project
files such as `lafke.lock` are never touched.

## Usage

```sh
lafke scan .                 # dependency trust report
lafke lock .                 # write lafke.lock
lafke verify .               # fail when the lock drifts
lafke sast .                 # find vulnerabilities in your source
lafke security . --format sarif
```

Run `lafke --help` for all commands. Documentation is in [`docs/`](docs/).

## Development

```sh
go vet ./...
go test ./...
```

Tests that need the analysis engine are skipped unless `LAFKE_ENGINE` points to a
`lafke-engine` executable.

## License

Apache-2.0. See [LICENSE](LICENSE).
