#!/usr/bin/env bash
# Install oncallgraph from a GitHub Release (or a local dist/ directory).
# Free, no accounts, verifies SHA256 when SHA256SUMS is present.
set -euo pipefail

REPO="${ONCALLGRAPH_REPO:-sanjeev0120test/oncallgraph}"
VERSION="${ONCALLGRAPH_VERSION:-latest}"
INSTALL_DIR="${ONCALLGRAPH_INSTALL_DIR:-${HOME}/.local/bin}"
DIST_DIR="${ONCALLGRAPH_DIST_DIR:-}"

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

tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

if [[ -n "$DIST_DIR" ]]; then
  # Local/CI mode: install from already-built dist artifacts.
  src="$(ls -1 "${DIST_DIR}"/oncallgraph-"${os}"-"${arch}"* 2>/dev/null | head -n1 || true)"
  if [[ -z "$src" ]]; then
    echo "no local binary for ${os}/${arch} in ${DIST_DIR}" >&2
    exit 1
  fi
  bin="$tmp/oncallgraph"
  cp "$src" "$bin"
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
  asset="oncallgraph_${tag}_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
  echo "downloading ${url}"
  curl -fsSL -o "$tmp/${asset}" "$url"
  curl -fsSL -o "$tmp/SHA256SUMS" "https://github.com/${REPO}/releases/download/${tag}/SHA256SUMS" || true
  if [[ -f "$tmp/SHA256SUMS" ]]; then
    (cd "$tmp" && sha256sum -c SHA256SUMS --ignore-missing)
  fi
  tar -xzf "$tmp/${asset}" -C "$tmp"
  bin="$(ls -1 "$tmp"/oncallgraph_${tag}_${os}_${arch} | head -n1)"
fi

chmod +x "$bin"
mkdir -p "$INSTALL_DIR"
dest="${INSTALL_DIR}/oncallgraph"
cp "$bin" "$dest"
echo "installed ${dest}"
"$dest" version
