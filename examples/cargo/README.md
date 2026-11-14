# Lafke Cargo example

Walk through the core loop on a tiny Cargo lockfile: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/cargo
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query OSV. The default policy treats unchecked
vulnerabilities and malicious reports as `unknown`, so `lock` exits
`1` unless `--no-fail` is set. `lafke.lock` in this directory is
that unknown snapshot for `cfg-if@1.0.0`.

Simulate a dependency change:

```bash
cp after/Cargo.lock Cargo.lock
lafke diff --offline
lafke verify --offline
```

`diff` reports the version and digest change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- Cargo.lock`.
