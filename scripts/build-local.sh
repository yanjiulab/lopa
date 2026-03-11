#!/usr/bin/env bash
set -euo pipefail

# Build lopa & lopad locally with version info injected,
# using the same ldflags scheme as CI.
#
# Usage:
#   scripts/build-local.sh [version]
# Example:
#   scripts/build-local.sh v0.1.0
#   scripts/build-local.sh          # auto-detect from git (fallback: dev)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

LDFLAGS="-s -w \
  -X github.com/yanjiulab/lopa/internal/version.Version=${VERSION} \
  -X github.com/yanjiulab/lopa/internal/version.Commit=${COMMIT} \
  -X github.com/yanjiulab/lopa/internal/version.Date=${DATE} \
  -X github.com/yanjiulab/lopa/internal/version.BuiltBy=local"

echo "Building lopa & lopad locally with:"
echo "  VERSION=${VERSION}"
echo "  COMMIT=${COMMIT}"
echo "  DATE=${DATE}"
echo "  GOOS=${GOOS:-$(go env GOOS)}"
echo "  GOARCH=${GOARCH:-$(go env GOARCH)}"

export CGO_ENABLED="${CGO_ENABLED:-0}"

go build -trimpath -ldflags "${LDFLAGS}" -o lopa ./cmd/lopa
go build -trimpath -ldflags "${LDFLAGS}" -o lopad ./cmd/lopad

echo "Done."
echo "  ./lopa version"
echo "  ./lopad --version"

