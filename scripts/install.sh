#!/bin/sh
set -eu

REPO="${REPO:-lafke/lafke}"
BINARY="${BINARY:-lafke}"
ENGINE_BINARY="${ENGINE_BINARY:-lafke-engine}"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION=$2
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

resolve_latest() {
  if command -v gh >/dev/null 2>&1; then
    gh release view --repo "$REPO" --json tagName --jq .tagName
    return 0
  fi
  url="https://github.com/${REPO}/releases/latest"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSLI -o /dev/null -w '%{url_effective}' "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget --max-redirect=0 --server-response "$url" 2>&1 | awk 'tolower($1) == "location:" { print $2; exit }'
  else
    echo "curl, wget, or gh is required" >&2
    exit 1
  fi
}

if [ -z "$VERSION" ]; then
  latest=$(resolve_latest)
  VERSION=${latest##*/}
fi
if [ -z "$VERSION" ]; then
  echo "could not resolve latest release" >&2
  exit 1
fi

archive="${BINARY}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${VERSION}"
archive_url="${base}/${archive}"
checksums_url="${base}/checksums.txt"

if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR=/usr/local/bin
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

say() {
  printf '%s\n' "$*"
}

say "lafke ${VERSION} (${os}/${arch})"
say "install dir: ${INSTALL_DIR}"

mkdir -p "$INSTALL_DIR"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

download() {
  url=$1
  dest=$2
  token="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
  if command -v curl >/dev/null 2>&1; then
    if [ -n "$token" ]; then
      curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/octet-stream" "$url" -o "$dest"
    else
      curl -fsSL "$url" -o "$dest"
    fi
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

fetch_assets() {
  if command -v gh >/dev/null 2>&1; then
    gh release download "$VERSION" --repo "$REPO" --pattern "$archive" --pattern checksums.txt --dir "$tmp"
    return 0
  fi
  download "$checksums_url" "${tmp}/checksums.txt"
  download "$archive_url" "${tmp}/${archive}"
}

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    echo "sha256sum or shasum is required" >&2
    exit 1
  fi
}

say "downloading ${archive}"
fetch_assets

expected=$(awk -v name="$archive" '$2 == name { print $1; exit }' "${tmp}/checksums.txt")
if [ -z "$expected" ]; then
  echo "no checksum for ${archive}" >&2
  exit 1
fi

got=$(sha256 "${tmp}/${archive}")
if [ "$got" != "$expected" ]; then
  echo "checksum mismatch for ${archive}" >&2
  echo "expected ${expected}" >&2
  echo "got      ${got}" >&2
  exit 1
fi

say "checksum ok: sha256:${got}"

# With gh available, the archive must also carry a valid GitHub attestation from REPO.
if command -v gh >/dev/null 2>&1; then
  if gh auth status >/dev/null 2>&1; then
    say "verifying attestation from ${REPO}"
    if ! gh attestation verify "${tmp}/${archive}" --repo "$REPO"; then
      echo "attestation verification failed for ${archive}" >&2
      exit 1
    fi
    say "attestation ok"
  else
    say "attestation not verified: gh is not authenticated"
  fi
else
  say "attestation not verified: gh is not installed"
fi

tar -xzf "${tmp}/${archive}" -C "$tmp"
install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
install -m 0755 "${tmp}/${ENGINE_BINARY}" "${INSTALL_DIR}/${ENGINE_BINARY}"

say ""
say "installed:"
say "  ${INSTALL_DIR}/${BINARY}"
say "  ${INSTALL_DIR}/${ENGINE_BINARY}"
"${INSTALL_DIR}/${BINARY}" version

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    say ""
    say "${INSTALL_DIR} is not on your PATH; add it with:"
    say "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

say ""
say "get started: lafke --help"
