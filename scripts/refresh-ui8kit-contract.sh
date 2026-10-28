#!/usr/bin/env bash
# Refresh the vendored @ui8kit/contract pin (schemas + official canonical vectors).
# Usage: scripts/refresh-ui8kit-contract.sh [commit]
set -euo pipefail

COMMIT="${1:-8a09c8fbf9229a00b3d16850afc17808a74e8366}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/conformance/contract"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if ! command -v git >/dev/null; then
  echo "git is required" >&2
  exit 1
fi

git clone --filter=blob:none https://github.com/ui8kit/contract.git "$TMP/contract"
# Short SHAs are not remote refs; clone origin/main first, then fetch the commit object if needed.
if ! git -C "$TMP/contract" cat-file -e "${COMMIT}^{commit}" 2>/dev/null; then
  git -C "$TMP/contract" fetch --depth=1 origin "$COMMIT"
fi
git -C "$TMP/contract" checkout --detach "$COMMIT"
COMMIT="$(git -C "$TMP/contract" rev-parse HEAD)"

mkdir -p "$DEST/schemas" "$DEST/fixtures/vectors"
cp "$TMP/contract"/schemas/*.json "$DEST/schemas/"
for f in canonical.json manifest.json term.json settings.json; do
  if [[ -f "$TMP/contract/fixtures/vectors/$f" ]]; then
    cp "$TMP/contract/fixtures/vectors/$f" "$DEST/fixtures/vectors/$f"
  fi
done

PARENT="$(git -C "$TMP/contract" rev-parse "${COMMIT}^" 2>/dev/null || true)"
VERSION="$(python3 -c "import json, pathlib; print(json.loads(pathlib.Path(r'''$TMP/contract/package.json''').read_text()).get('version', 'unknown'))" 2>/dev/null || true)"
if [[ -z "$VERSION" ]]; then
  VERSION="unknown"
fi

cat > "$DEST/VERSION" <<EOF
package: @ui8kit/contract
version: ${VERSION}
commit: ${COMMIT}
parent: ${PARENT}
source: https://github.com/ui8kit/contract
note: schemas and fixtures/vectors/*.json only; do not tighten vendored JSON Schema
EOF

echo "pinned @ui8kit/contract ${COMMIT} -> ${DEST}"
