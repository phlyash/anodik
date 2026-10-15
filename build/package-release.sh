#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

if [[ $# != 1 ]]; then
  printf 'usage: [ANODIK_VERSION=<tag or version>] %s OUTPUT_DIRECTORY\n' "$0" >&2
  exit 1
fi

# The version "anodik version" reports. A release tag such as v0.7 becomes
# 0.7; the rule matches build/prepare-beget-release.py. Without one the
# binaries report "dev".
version=${ANODIK_VERSION:-}
if [[ -z $version ]]; then
  version=dev
elif version=${version#v}; [[ ! $version =~ ^[0-9][A-Za-z0-9._+~-]*$ ]]; then
  printf 'invalid ANODIK_VERSION\n' >&2
  exit 1
fi

OUTPUT_DIR=$1
if [[ -L $OUTPUT_DIR ]]; then
  printf 'output must not be a symlink\n' >&2
  exit 1
fi

if [[ -e $OUTPUT_DIR ]]; then
  if [[ ! -d $OUTPUT_DIR || -n $(find "$OUTPUT_DIR" -mindepth 1 -maxdepth 1 -print -quit) ]]; then
    printf 'output directory must be empty\n' >&2
    exit 1
  fi
else
  mkdir -p -- "$OUTPUT_DIR"
fi

OUTPUT_DIR=$(cd -- "$OUTPUT_DIR" && pwd -P)
GO_BIN=${GO_BIN:-go}

targets=(
  'linux amd64 linux x86_64 anodik'
  'darwin arm64 darwin aarch64 anodik'
  'windows amd64 windows x86_64 anodik.exe'
)

for target in "${targets[@]}"; do
  read -r goos goarch catalog_os catalog_arch executable <<<"$target"
  stage=$(mktemp -d "${TMPDIR:-/tmp}/anodik-package.XXXXXX")
  trap 'rm -rf -- "$stage"' EXIT

  mkdir -p "$stage/bin"
  (
    cd "$ROOT"
    env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
      "$GO_BIN" build \
        -trimpath \
        -ldflags="-s -w -X anodik/internal/cli.Version=$version" \
        -o "$stage/bin/$executable" \
        .
  )
  chmod 755 "$stage/bin/$executable"

  base="anodik-$catalog_os-$catalog_arch"
  tar -czf "$OUTPUT_DIR/$base.tar.gz" -C "$stage" .
  (
    cd "$stage"
    zip -q -r "$OUTPUT_DIR/$base.zip" bin
  )

  rm -rf -- "$stage"
  trap - EXIT
done
