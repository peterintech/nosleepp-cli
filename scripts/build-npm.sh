#!/usr/bin/env sh
set -eu

VERSION="${1:-0.2.0}"
ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
BIN_DIR="$ROOT/npm/bin"
GO_CACHE="$ROOT/.gocache"

mkdir -p "$BIN_DIR"
mkdir -p "$GO_CACHE"

build_nosleepp() {
  goos="$1"
  goarch="$2"
  output="$3"
  echo "Building $goos/$goarch -> $output"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 GOCACHE="$GO_CACHE" go build -ldflags "-X nosleepp/cmd.version=$VERSION" -o "$BIN_DIR/$output" .
}

cd "$ROOT"
build_nosleepp windows amd64 nosleepp-win32-x64.exe
build_nosleepp darwin arm64 nosleepp-darwin-arm64
build_nosleepp darwin amd64 nosleepp-darwin-x64

echo "Built npm binaries in $BIN_DIR"
