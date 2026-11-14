# Lafke Cabal example

Walk through the core loop on a tiny `cabal.project.freeze`: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/cabal
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query OSV. A freeze file carries no hashes, so no
digest is recorded. The default policy treats unchecked vulnerabilities and
malicious reports as `unknown`, so `lock` exits `1` unless `--no-fail` is set.
`lafke.lock` in this directory is that unknown snapshot for
`aeson@2.2.1.0`.

Simulate a dependency change:

```bash
cp after/cabal.project.freeze cabal.project.freeze
lafke diff --offline
lafke verify --offline
```

`diff` reports the version change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- cabal.project.freeze`.
