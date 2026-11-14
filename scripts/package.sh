#!/bin/sh
# Builds the CLI and the engine for one platform and packages them in one archive.
#
# usage: package.sh <goos> <goarch> <version>
# env:   CLI_DIR (default .), ENGINE_DIR (default ./engine), OUT_DIR (default ./dist)
set -eu

goos=${1:?goos}
goarch=${2:?goarch}
version=${3:?version}

CLI_DIR=$(cd "${CLI_DIR:-.}" && pwd)
ENGINE_DIR=$(cd "${ENGINE_DIR:-engine}" && pwd)
mkdir -p "${OUT_DIR:-dist}"
OUT_DIR=$(cd "${OUT_DIR:-dist}" && pwd)

ext=""
if [ "$goos" = windows ]; then ext=".exe"; fi

commit=$(git -C "$CLI_DIR" rev-parse --short HEAD 2>/dev/null || echo none)
date=$(git -C "$CLI_DIR" log -1 --format=%cI 2>/dev/null || echo unknown)

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

export CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch"
(cd "$CLI_DIR" && go build -trimpath \
  -ldflags "-s -w -X main.version=${version} -X main.commit=${commit} -X main.date=${date}" \
  -o "$stage/lafke${ext}" ./cmd/lafke)
(cd "$ENGINE_DIR" && go build -trimpath \
  -ldflags "-s -w -X main.version=${version}" \
  -o "$stage/lafke-engine${ext}" ./cmd/lafke-engine)

cp "$CLI_DIR/LICENSE" "$stage/LICENSE"

name="lafke_${goos}_${goarch}"
if [ "$goos" = windows ]; then
  (cd "$stage" && zip -q "$OUT_DIR/${name}.zip" lafke.exe lafke-engine.exe LICENSE)
  echo "$OUT_DIR/${name}.zip"
else
  tar -C "$stage" -czf "$OUT_DIR/${name}.tar.gz" lafke lafke-engine LICENSE
  echo "$OUT_DIR/${name}.tar.gz"
fi
