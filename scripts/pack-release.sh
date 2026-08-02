#!/usr/bin/env bash
# Package opsgraph release archives from built binaries.
# Shared by CI and the release workflow to avoid packaging drift.
#
# Usage:
#   scripts/pack-release.sh <version> <bin-dir> <out-dir>
#
# <bin-dir> may contain either naming scheme:
#   opsgraph-<goos>-<goarch>[.exe]              (CI / make cross)
#   opsgraph_<version>_<goos>_<goarch>[.exe]    (release build)
#
# Writes to <out-dir>:
#   6 archives + LICENSE + DEPENDENCIES.txt + sbom.gomod.json + install.sh + install.ps1 + SHA256SUMS
set -euo pipefail

VERSION="${1:?version required (e.g. v0.1.5)}"
BIN_DIR="${2:?bin-dir required}"
OUT_DIR="${3:?out-dir required}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/opsgraph_*.tar.gz "$OUT_DIR"/opsgraph_*.zip \
  "$OUT_DIR"/SHA256SUMS "$OUT_DIR"/DEPENDENCIES.txt "$OUT_DIR"/sbom.gomod.json \
  "$OUT_DIR"/LICENSE "$OUT_DIR"/install.sh "$OUT_DIR"/install.ps1

resolve_src() {
  local goos="$1" goarch="$2"
  local a="${BIN_DIR}/opsgraph_${VERSION}_${goos}_${goarch}"
  local b="${BIN_DIR}/opsgraph-${goos}-${goarch}"
  if [ "$goos" = windows ]; then
    a="${a}.exe"
    b="${b}.exe"
  fi
  if [ -s "$a" ]; then
    printf '%s' "$a"
  elif [ -s "$b" ]; then
    printf '%s' "$b"
  else
    echo "missing binary for ${goos}/${goarch} in ${BIN_DIR}" >&2
    return 1
  fi
}

for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  GOOS="${pair%/*}"
  GOARCH="${pair#*/}"
  src="$(resolve_src "$GOOS" "$GOARCH")"
  # Cheap integrity: confirm embed GOOS/GOARCH match the filename.
  meta="$(go version -m "$src")"
  # go version -m lines look like: build<TAB>GOOS=linux
  echo "$meta" | grep -F "GOOS=${GOOS}" >/dev/null
  echo "$meta" | grep -F "GOARCH=${GOARCH}" >/dev/null

  base="opsgraph_${VERSION}_${GOOS}_${GOARCH}"
  # Stable archive metadata so same inputs rebuild to matching SHA256SUMS.
  export SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-0}"
  if [ "$GOOS" = windows ]; then
    cp "$src" "${OUT_DIR}/${base}.exe"
    # Normalize mtime so zip payloads are SOURCE_DATE_EPOCH-stable.
    touch -d "@${SOURCE_DATE_EPOCH}" "${OUT_DIR}/${base}.exe" 2>/dev/null || \
      touch -t 197001010000.00 "${OUT_DIR}/${base}.exe" 2>/dev/null || true
    (cd "$OUT_DIR" && zip -q -X "${base}.zip" "${base}.exe")
    rm -f "${OUT_DIR}/${base}.exe"
  else
    cp "$src" "${OUT_DIR}/${base}"
    # GNU tar: fixed mtime/owner; gzip -n drops timestamp from header.
    if tar --version 2>/dev/null | grep -qi gnu; then
      (cd "$OUT_DIR" && tar --sort=name --owner=0 --group=0 --numeric-owner \
        --mtime="@${SOURCE_DATE_EPOCH}" -cf - "${base}" | gzip -n > "${base}.tar.gz")
    else
      (cd "$OUT_DIR" && tar -czf "${base}.tar.gz" "${base}")
    fi
    rm -f "${OUT_DIR}/${base}"
  fi
done

cp "${ROOT}/LICENSE" "${OUT_DIR}/LICENSE"
cp "${ROOT}/scripts/install.sh" "${OUT_DIR}/install.sh"
cp "${ROOT}/scripts/install.ps1" "${OUT_DIR}/install.ps1"
chmod +x "${OUT_DIR}/install.sh"
(cd "$ROOT" && go list -m all) > "${OUT_DIR}/DEPENDENCIES.txt"
# Module SBOM as a single JSON array (go list -m -json emits pretty multi-object stream).
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 required to build sbom.gomod.json" >&2
  exit 1
fi
(cd "$ROOT" && go list -m -json all) | python3 -c '
import json, sys
dec = json.JSONDecoder()
data = sys.stdin.read()
objs, idx = [], 0
while True:
    while idx < len(data) and data[idx].isspace():
        idx += 1
    if idx >= len(data):
        break
    obj, end = dec.raw_decode(data, idx)
    objs.append(obj)
    idx = end
json.dump(objs, sys.stdout, indent=2)
print()
' > "${OUT_DIR}/sbom.gomod.json"

if command -v sha256sum >/dev/null 2>&1; then
  HASH_CMD=(sha256sum)
  CHECK_CMD=(sha256sum -c)
elif command -v shasum >/dev/null 2>&1; then
  HASH_CMD=(shasum -a 256)
  CHECK_CMD=(shasum -a 256 -c)
else
  echo "need sha256sum or shasum to write SHA256SUMS" >&2
  exit 1
fi

(
  cd "$OUT_DIR"
  # -- stops option parsing so a leading-dash archive name cannot break hashing.
  "${HASH_CMD[@]}" -- *.tar.gz *.zip LICENSE DEPENDENCIES.txt sbom.gomod.json install.sh install.ps1 > SHA256SUMS
  test "$(grep -c . SHA256SUMS)" -eq 11
  "${CHECK_CMD[@]}" SHA256SUMS
  echo "--- SHA256SUMS ---"
  cat SHA256SUMS
)

echo "packaged release ${VERSION} -> ${OUT_DIR}"
