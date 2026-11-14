# Lafke uv example

Walk through the core loop on a tiny uv lockfile: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/uv
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query OSV or the PyPI Integrity API. The default
policy treats unchecked vulnerabilities and malicious reports as
`unknown`, so `lock` exits `1` unless `--no-fail` is set.
`lafke.lock` in this directory is that unknown snapshot for
`idna@3.20`.

Simulate a dependency change:

```bash
cp after/uv.lock uv.lock
lafke diff --offline
lafke verify --offline
```

`diff` reports the version and digest change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- uv.lock`.
