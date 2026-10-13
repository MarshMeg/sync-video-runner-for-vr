#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
APP_NAME="vr-sync-run"
PKG_PATH="./cmd"
GO_LDFLAGS="-s -w -buildid="
USE_UPX="${USE_UPX:-1}"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed or not in PATH" >&2
  exit 1
fi

mkdir -p "${DIST_DIR}"

targets=(
  "linux amd64"
  "windows amd64"
  "darwin amd64"
)

has_upx=0
if [[ "${USE_UPX}" == "1" ]] && command -v upx >/dev/null 2>&1; then
  has_upx=1
fi

for target in "${targets[@]}"; do
  read -r goos goarch <<<"${target}"

  suffix=""
  if [[ "${goos}" == "windows" ]]; then
    suffix=".exe"
  fi

  output="${DIST_DIR}/${APP_NAME}-${goos}-${goarch}${suffix}"
  echo "Building ${goos}/${goarch} -> ${output}"

  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -buildvcs=false -ldflags="${GO_LDFLAGS}" -o "${output}" "${PKG_PATH}"

  if [[ "${has_upx}" -eq 1 ]]; then
    echo "Compressing with upx -> ${output}"
    if ! upx --best --lzma "${output}" >/dev/null; then
      echo "warning: upx compression failed for ${output}, keeping uncompressed binary" >&2
    fi
  fi
done

if [[ "${USE_UPX}" == "1" ]] && [[ "${has_upx}" -eq 0 ]]; then
  echo "note: upx not found, skipped binary compression" >&2
fi

echo "Done. Artifacts are in ${DIST_DIR}"
