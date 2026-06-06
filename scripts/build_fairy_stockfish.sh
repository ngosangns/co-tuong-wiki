#!/usr/bin/env bash
# Build Fairy-Stockfish with the Xiangqi ruleset enabled.
#
# Fairy-Stockfish is a multi-variant engine derived from Stockfish that
# supports xiangqi when built with the appropriate ruleset. We use it as
# the secondary comparison engine behind Pikafish, which is the primary
# analysis engine for the lesson experience.
#
# Usage:
#   ./scripts/build_fairy_stockfish.sh /absolute/path/to/Fairy-Stockfish-src
#
# Output:
#   ./bin/fairy-stockfish              # the compiled binary
#
# Requirements:
#   - g++ (C++17) or clang++
#   - make
#   - A working clone of https://github.com/fairy-stockfish/Fairy-Stockfish

set -euo pipefail

SRC_DIR="${1:-${FAIRY_STOCKFISH_SRC:-}}"
if [ -z "$SRC_DIR" ] || [ ! -d "$SRC_DIR" ]; then
  echo "ERROR: provide the path to a Fairy-Stockfish source tree as the first argument" >&2
  echo "       (e.g. ~/src/Fairy-Stockfish)" >&2
  echo "       or set FAIRY_STOCKFISH_SRC" >&2
  exit 2
fi

OUT_DIR="$(cd "$(dirname "$0")/.." && pwd)/bin"
mkdir -p "$OUT_DIR"
OUT_BIN="$OUT_DIR/fairy-stockfish"

if [ -x "$OUT_BIN" ]; then
  echo "Replacing existing $OUT_BIN"
fi

cd "$SRC_DIR"

echo ">>> Cleaning previous build"
make -C src clean >/dev/null 2>&1 || true

echo ">>> Building Fairy-Stockfish with Xiangqi ruleset (profile-build)"
make -C src -j"$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)" profile-build \
  EXTRACXXFLAGS="-DUSE_XIANGQI" \
  >/tmp/fairy-stockfish-build.log 2>&1

cp src/stockfish "$OUT_BIN"
chmod +x "$OUT_BIN"

echo ">>> Built $OUT_BIN"
"$OUT_BIN" --help 2>&1 | head -n 1 || true
echo
echo "Set ENGINE_FAIRY_PATH to enable the secondary engine in the API:"
echo "  export ENGINE_FAIRY_PATH=\"$OUT_BIN\""
