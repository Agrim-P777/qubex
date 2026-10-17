#!/usr/bin/env bash
set -euo pipefail
: "${VERSION:=dev}"
mkdir -p dist
python3 scripts/notices.py > dist/THIRD_PARTY_NOTICES.txt
for target_os in darwin linux; do
  for target_arch in amd64 arm64; do
    artifact="dist/qbx-${target_os}-${target_arch}"
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$artifact" ./cmd/qbx
  done
done
python3 scripts/package_release.py "$VERSION"
