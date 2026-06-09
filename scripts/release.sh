#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-$(cat "${ROOT_DIR}/VERSION" 2>/dev/null || echo v0.1.0)}"
APP_NAME="wecom-go"
OUT_DIR="${ROOT_DIR}/dist/release"
BUILD_DIR="${ROOT_DIR}/dist/build"

targets=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/arm64"
  "linux/amd64"
  "windows/amd64"
)

rm -rf "${BUILD_DIR}"
mkdir -p "${OUT_DIR}" "${BUILD_DIR}"

for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  package_dir="${BUILD_DIR}/${APP_NAME}-${VERSION}-${os}-${arch}"
  mkdir -p "${package_dir}"

  binary="${APP_NAME}"
  if [[ "${os}" == "windows" ]]; then
    binary="${APP_NAME}.exe"
  fi

  GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 go -C "${ROOT_DIR}" build -o "${package_dir}/${binary}" ./cmd/wecom-go
  cp "${ROOT_DIR}/README.md" "${package_dir}/README.md"
  cat > "${package_dir}/manifest.json" <<EOF
{
  "name": "${APP_NAME}",
  "version": "${VERSION}",
  "os": "${os}",
  "arch": "${arch}",
  "binary": "${binary}"
}
EOF

  if [[ "${os}" == "windows" ]]; then
    (cd "${BUILD_DIR}" && zip -qr "${OUT_DIR}/${APP_NAME}-${VERSION}-${os}-${arch}.zip" "$(basename "${package_dir}")")
  else
    tar -C "${BUILD_DIR}" -czf "${OUT_DIR}/${APP_NAME}-${VERSION}-${os}-${arch}.tar.gz" "$(basename "${package_dir}")"
  fi
done

(cd "${OUT_DIR}" && shasum -a 256 "${APP_NAME}-${VERSION}"-* > SHA256SUMS)
rm -rf "${BUILD_DIR}"
printf 'Created release artifacts in %s\n' "${OUT_DIR}"
