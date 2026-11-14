# Lafke renv example

Walk through the core loop on a tiny `renv.lock`: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/renv
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query OSV. renv's `Hash` is not an artifact digest, so
none is recorded. The default policy treats unchecked vulnerabilities and
malicious reports as `unknown`, so `lock` exits `1` unless `--no-fail` is set.
`lafke.lock` in this directory is that unknown snapshot for
`jsonlite@1.8.8`.

Simulate a dependency change:

```bash
cp after/renv.lock renv.lock
lafke diff --offline
lafke verify --offline
```

`diff` reports the version change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- renv.lock`.
