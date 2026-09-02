#!/usr/bin/env bash

set -euo pipefail

APP_NAME="anodik"
OUT_DIR="dist"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

# GOOS / GOARCH targets.
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"

    "darwin/amd64"
    "darwin/arm64"

    "windows/amd64"
    "windows/arm64"
)

for target in "${TARGETS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "$target"

    if [[ "$GOOS" == "windows" ]]; then
        EXT=".exe"
    else
        EXT=""
    fi

    OUTPUT="$OUT_DIR/${APP_NAME}-${GOOS}-${GOARCH}${EXT}"

    echo "==> Building $GOOS/$GOARCH"
    echo "    $OUTPUT"

    CGO_ENABLED=0 \
    GOOS="$GOOS" \
    GOARCH="$GOARCH" \
    go build \
        -trimpath \
        -ldflags="-s -w" \
        -o "$OUTPUT" \
        .

    echo
done

echo "Build complete:"
ls -lh "$OUT_DIR"
