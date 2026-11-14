# Lafke Go example

Walk through the core loop on a tiny Go module: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/go
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query OSV. The default policy treats unchecked
vulnerabilities and malicious reports as `unknown`, so `lock` exits
`1` unless `--no-fail` is set. `lafke.lock` in this directory is
that unknown snapshot for `rsc.io/quote@v1.5.2`.

Simulate a dependency change:

```bash
cp after/go.mod go.mod
cp after/go.sum go.sum
lafke diff --offline
lafke verify --offline
```

`diff` reports the version and digest change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original files with `git checkout -- go.mod go.sum`.
