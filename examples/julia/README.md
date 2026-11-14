# Lafke Julia example

Walk through the core loop on a tiny `Manifest.toml`: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/julia
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query anything. `Manifest.toml` does not say which
registry served a package, so no registry is recorded and OSV is not queried
in `public-only`. `git-tree-sha1` is kept as the resolved reference, not as
a digest, and `Project.toml` supplies the runtime scope. The default policy
treats unchecked evidence as `unknown`, so `lock` exits `1` unless
`--no-fail` is set.
`lafke.lock` in this directory is that unknown snapshot for
`URIs@1.5.1`.

Simulate a dependency change:

```bash
cp after/Manifest.toml Manifest.toml
lafke diff --offline
lafke verify --offline
```

`diff` reports the version and resolved tree change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- Manifest.toml`.
