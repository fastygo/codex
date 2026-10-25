#!/usr/bin/env bash
# Refresh the vendored @ui8kit/contract pin (schemas + official canonical vectors).
# Usage: scripts/refresh-ui8kit-contract.sh [commit]
set -euo pipefail

COMMIT="${1:-a70c243}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/conformance/contract"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if ! command -v git >/dev/null; then
  echo "git is required" >&2
  exit 1
fi

git clone --filter=blob:none https://github.com/ui8kit/contract.git "$TMP/contract"
git -C "$TMP/contract" fetch --depth=1 origin "$COMMIT"
git -C "$TMP/contract" checkout --detach "$COMMIT"

mkdir -p "$DEST/schemas" "$DEST/fixtures/vectors"
cp "$TMP/contract"/schemas/*.json "$DEST/schemas/"
for f in canonical.json manifest.json term.json settings.json; do
  if [[ -f "$TMP/contract/fixtures/vectors/$f" ]]; then
    cp "$TMP/contract/fixtures/vectors/$f" "$DEST/fixtures/vectors/$f"
  fi
done

PARENT="$(git -C "$TMP/contract" rev-parse "${COMMIT}^" 2>/dev/null || true)"
VERSION="$(python3 - <<'PY' 2>/dev/null || true
import json, pathlib
p = pathlib.Path("'"$TMP"'/contract/package.json")
print(json.loads(p.read_text()).get("version", "unknown"))
PY
)"
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
