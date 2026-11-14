# Security Policy

## Reporting a vulnerability

Please report security issues privately through GitHub Security Advisories
on [lafke/lafke](https://github.com/lafke/lafke/security/advisories/new).

Do not open a public issue for an unreleased vulnerability.

We aim to acknowledge reports within 5 business days.

Lafke may send public dependency names and versions to OSV and
the npm registry. Private registries are not queried in the default
`public-only` network mode. See [Privacy](docs/privacy.md).

## Supported versions

Only the latest tagged release is supported until v1.0.

## What Lafke verifies about itself

Release artifacts are published with:

- `checksums.txt`
- SPDX SBOMs
- GitHub artifact attestations
- Immutable releases (tag and assets locked after publish)

Consumers can verify either the release archive or the extracted binary:

```bash
gh attestation verify lafke_darwin_arm64.tar.gz --repo lafke/lafke
gh attestation verify ./lafke --repo lafke/lafke
```

`scripts/install.sh` checks the published SHA-256 before install. If `gh`
is available, it also verifies the GitHub attestation.
