# Lafke Conan example

Walk through the core loop on a tiny Conan 2 `conan.lock`: lock, change a
dependency, inspect trust drift, then verify.

```bash
cd examples/conan
lafke lock --offline --no-fail
lafke verify --offline
```

`--offline` does not query anything. The lockfile does not record which
remote served a recipe and OSV has no ConanCenter ecosystem, so
vulnerabilities stay `unknown` even online. The recipe revision is kept as
the resolved reference, not as a digest. The default policy treats
unchecked evidence as `unknown`, so `lock` exits `1` unless `--no-fail` is set.
`lafke.lock` in this directory is that unknown snapshot for
`zlib@1.3.1`.

Simulate a dependency change:

```bash
cp after/conan.lock conan.lock
lafke diff --offline
lafke verify --offline
```

`diff` reports the version and recipe revision change. `verify` fails until you
accept the new tree:

```bash
lafke lock --offline
lafke verify --offline
```

Restore the original lockfile with `git checkout -- conan.lock`.
