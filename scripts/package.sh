#!/bin/sh
# Builds the CLI for one platform and packages it in one archive. The engine is released
# separately; see lafke/engine-releases and `lafke engine install`.
#
# usage: package.sh <goos> <goarch> <version>
# env:   CLI_DIR (default .), OUT_DIR (default ./dist)
set -eu

goos=${1:?goos}
goarch=${2:?goarch}
version=${3:?version}

CLI_DIR=$(cd "${CLI_DIR:-.}" && pwd)
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

cp "$CLI_DIR/LICENSE" "$stage/LICENSE"

name="lafke_${goos}_${goarch}"
if [ "$goos" = windows ]; then
  (cd "$stage" && zip -q "$OUT_DIR/${name}.zip" lafke.exe LICENSE)
  echo "$OUT_DIR/${name}.zip"
else
  tar -C "$stage" -czf "$OUT_DIR/${name}.tar.gz" lafke LICENSE
  echo "$OUT_DIR/${name}.tar.gz"
fi
