#!/usr/bin/env bash
# Install opsgraph from a GitHub Release (or a local dist/ directory).
# Free, no accounts, verifies SHA256 when SHA256SUMS is present.
# Set OPSGRAPH_INSECURE=1 to skip checksum verification (not recommended).
set -euo pipefail

# Prefer OPSGRAPH_*; accept ONCALLGRAPH_* aliases from the brief rename period.
REPO="${OPSGRAPH_REPO:-${ONCALLGRAPH_REPO:-sanjeev0120test/opsgraph}}"
VERSION="${OPSGRAPH_VERSION:-${ONCALLGRAPH_VERSION:-latest}}"
INSTALL_DIR="${OPSGRAPH_INSTALL_DIR:-${ONCALLGRAPH_INSTALL_DIR:-${HOME}/.local/bin}}"
DIST_DIR="${OPSGRAPH_DIST_DIR:-${ONCALLGRAPH_DIST_DIR:-}}"
# Local release-layout dir (archives + SHA256SUMS), same shape as GitHub Releases.
RELEASE_DIR="${OPSGRAPH_RELEASE_DIR:-${ONCALLGRAPH_RELEASE_DIR:-}}"
INSECURE="${OPSGRAPH_INSECURE:-${ONCALLGRAPH_INSECURE:-0}}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) os=windows ;;
  *) echo "unsupported os: $os" >&2; exit 1 ;;
esac

verify_asset_sha256() {
  local sums_file="$1"
  local asset_name="$2"
  local asset_path="$3"
  local want="" got="" hash name
  while read -r hash name; do
    name="${name#\*}"
    name="$(printf '%s' "$name" | tr -d '\r')"
    if [[ "$name" == "$asset_name" ]]; then
      want="$hash"
      break
    fi
  done < "$sums_file"
  if [[ -z "$want" ]]; then
    echo "SHA256SUMS has no entry for ${asset_name}" >&2
    return 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum "$asset_path" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 "$asset_path" | awk '{print $1}')"
  else
    echo "no sha256sum/shasum available for verification" >&2
    return 1
  fi
  if [[ "$want" != "$got" ]]; then
    echo "SHA256 mismatch for ${asset_name}" >&2
    echo "  want: $want" >&2
    echo "  got:  $got" >&2
    return 1
  fi
  echo "checksum ok: ${asset_name}"
}

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

install_from_archive() {
  local tag="$1"
  local asset_path="$2"
  local sums_path="$3"
  local asset
  asset="$(basename "$asset_path")"
  if [[ "$INSECURE" != "1" ]]; then
    if [[ ! -s "$sums_path" ]]; then
      echo "SHA256SUMS missing or empty (set OPSGRAPH_INSECURE=1 to skip)" >&2
      exit 1
    fi
    verify_asset_sha256 "$sums_path" "$asset" "$asset_path"
  else
    echo "warning: skipping checksum verification (OPSGRAPH_INSECURE=1)" >&2
  fi
  if [[ "$os" == windows ]]; then
    if command -v unzip >/dev/null 2>&1; then
      unzip -q -o "$asset_path" -d "$tmp"
    else
      powershell.exe -NoProfile -Command "Expand-Archive -Path '$asset_path' -DestinationPath '$tmp' -Force"
    fi
    bin="$(ls -1 "$tmp"/opsgraph_${tag}_${os}_${arch}.exe | head -n1)"
  else
    tar -xzf "$asset_path" -C "$tmp"
    bin="$(ls -1 "$tmp"/opsgraph_${tag}_${os}_${arch} | head -n1)"
  fi
  if [[ -z "${bin:-}" || ! -e "$bin" ]]; then
    echo "extracted binary missing for ${os}/${arch}" >&2
    exit 1
  fi
}

if [[ -n "$DIST_DIR" ]]; then
  # Local/CI mode: install from already-built raw dist binaries.
  src="$(ls -1 "${DIST_DIR}"/opsgraph-"${os}"-"${arch}"* 2>/dev/null | head -n1 || true)"
  if [[ -z "$src" ]]; then
    echo "no local binary for ${os}/${arch} in ${DIST_DIR}" >&2
    exit 1
  fi
  bin="$tmp/opsgraph"
  if [[ "$os" == windows ]]; then
    bin="$tmp/opsgraph.exe"
  fi
  cp "$src" "$bin"
elif [[ -n "$RELEASE_DIR" ]]; then
  # Local/CI mode: install from a release-shaped directory (archives + SHA256SUMS).
  if [[ "$VERSION" == "latest" || -z "$VERSION" ]]; then
    echo "OPSGRAPH_VERSION must be set when using OPSGRAPH_RELEASE_DIR" >&2
    exit 1
  fi
  tag="$VERSION"
  if [[ "$os" == windows ]]; then
    asset="opsgraph_${tag}_${os}_${arch}.zip"
  else
    asset="opsgraph_${tag}_${os}_${arch}.tar.gz"
  fi
  if [[ ! -s "${RELEASE_DIR}/${asset}" ]]; then
    echo "missing release asset ${asset} in ${RELEASE_DIR}" >&2
    exit 1
  fi
  install_from_archive "$tag" "${RELEASE_DIR}/${asset}" "${RELEASE_DIR}/SHA256SUMS"
else
  if [[ "$VERSION" == "latest" ]]; then
    tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)"
  else
    tag="$VERSION"
  fi
  if [[ -z "${tag}" ]]; then
    echo "could not resolve release tag for ${REPO}" >&2
    exit 1
  fi
  if [[ "$os" == windows ]]; then
    asset="opsgraph_${tag}_${os}_${arch}.zip"
  else
    asset="opsgraph_${tag}_${os}_${arch}.tar.gz"
  fi
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
  echo "downloading ${url}"
  curl -fsSL -o "$tmp/${asset}" "$url"
  if ! curl -fsSL -o "$tmp/SHA256SUMS" "https://github.com/${REPO}/releases/download/${tag}/SHA256SUMS"; then
    if [[ "$INSECURE" != "1" ]]; then
      echo "failed to download SHA256SUMS (set OPSGRAPH_INSECURE=1 to skip)" >&2
      exit 1
    fi
    : > "$tmp/SHA256SUMS"
  fi
  install_from_archive "$tag" "$tmp/${asset}" "$tmp/SHA256SUMS"
fi

chmod +x "$bin" 2>/dev/null || true
mkdir -p "$INSTALL_DIR"
dest="${INSTALL_DIR}/opsgraph"
if [[ "$os" == windows ]]; then
  dest="${INSTALL_DIR}/opsgraph.exe"
fi
cp "$bin" "$dest"
echo "installed ${dest}"
"$dest" version
