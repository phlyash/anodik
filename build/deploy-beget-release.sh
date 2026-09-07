#!/usr/bin/env bash
set -euo pipefail

require_env() {
  local name=$1
  if [[ -z ${!name:-} ]]; then
    printf 'missing required environment variable: %s\n' "$name" >&2
    exit 1
  fi
}

for name in \
  BEGET_HOST BEGET_PORT BEGET_USER BEGET_SSH_PRIVATE_KEY BEGET_KNOWN_HOSTS \
  GITHUB_REPOSITORY GITHUB_RUN_ID GITHUB_RUN_ATTEMPT RUNNER_TEMP; do
  require_env "$name"
done

if [[ $# != 1 || ! -d $1 || -L $1 ]]; then
  printf 'usage: %s BUNDLE_DIRECTORY\n' "$0" >&2
  exit 1
fi
BUNDLE_DIR=$(realpath -- "$1")

if [[ ! $BEGET_HOST =~ ^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$ || $BEGET_HOST == *..* ]]; then
  printf 'unsafe BEGET_HOST\n' >&2
  exit 1
fi
if [[ ! $BEGET_USER =~ ^[a-z_][a-z0-9_-]{0,31}$ ]]; then
  printf 'unsafe BEGET_USER\n' >&2
  exit 1
fi
if [[ ! $BEGET_PORT =~ ^[1-9][0-9]{0,4}$ || $BEGET_PORT -gt 65535 ]]; then
  printf 'unsafe BEGET_PORT\n' >&2
  exit 1
fi
if [[ $GITHUB_REPOSITORY != phlyash/anodik ]]; then
  printf 'unauthorized GITHUB_REPOSITORY\n' >&2
  exit 1
fi
if [[ ! $GITHUB_RUN_ID =~ ^[1-9][0-9]*$ || ! $GITHUB_RUN_ATTEMPT =~ ^[1-9][0-9]*$ ]]; then
  printf 'unsafe GitHub run values\n' >&2
  exit 1
fi

expected_files=(
  anodik-linux-x86_64.tar.gz
  anodik-linux-x86_64.zip
  anodik-darwin-aarch64.tar.gz
  anodik-darwin-aarch64.zip
  anodik-windows-x86_64.tar.gz
  anodik-windows-x86_64.zip
  release.json
)
mapfile -t bundle_files < <(
  find "$BUNDLE_DIR" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort
)
if [[ ${#bundle_files[@]} != 7 ]]; then
  printf 'bundle must contain exactly seven files\n' >&2
  exit 1
fi
mapfile -t expected_sorted < <(printf '%s\n' "${expected_files[@]}" | sort)
for index in "${!expected_sorted[@]}"; do
  if [[ ${bundle_files[index]} != "${expected_sorted[index]}" ]]; then
    printf 'unexpected bundle contents\n' >&2
    exit 1
  fi
  path="$BUNDLE_DIR/${bundle_files[index]}"
  if [[ ! -f $path || -L $path ]]; then
    printf 'bundle entry must be a regular non-symlink file\n' >&2
    exit 1
  fi
done

upload_id="$GITHUB_RUN_ID-$GITHUB_RUN_ATTEMPT"
umask 077
SSH_DIR=$(mktemp -d "$RUNNER_TEMP/aspect-ssh.XXXXXX")
chmod 700 "$SSH_DIR"
trap 'rm -rf -- "$SSH_DIR"' EXIT
printf '%s\n' "$BEGET_SSH_PRIVATE_KEY" >"$SSH_DIR/deploy_key"
printf '%s\n' "$BEGET_KNOWN_HOSTS" >"$SSH_DIR/known_hosts"
chmod 600 "$SSH_DIR/deploy_key" "$SSH_DIR/known_hosts"

SSH_BIN=${SSH_BIN:-ssh}
SCP_BIN=${SCP_BIN:-scp}
SSH_ARGS=(
  -p "$BEGET_PORT"
  -i "$SSH_DIR/deploy_key"
  -o BatchMode=yes
  -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=yes
  -o "UserKnownHostsFile=$SSH_DIR/known_hosts"
)
SCP_ARGS=(
  -P "$BEGET_PORT"
  -i "$SSH_DIR/deploy_key"
  -o BatchMode=yes
  -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=yes
  -o "UserKnownHostsFile=$SSH_DIR/known_hosts"
  -O
)
target="$BEGET_USER@$BEGET_HOST"

"$SSH_BIN" "${SSH_ARGS[@]}" "$target" "aspect-publisher prepare $upload_id"

upload_paths=()
for file in "${expected_sorted[@]}"; do
  upload_paths+=("$BUNDLE_DIR/$file")
done
"$SCP_BIN" "${SCP_ARGS[@]}" "${upload_paths[@]}" \
  "$target:aspect-upload/$upload_id/"

"$SSH_BIN" "${SSH_ARGS[@]}" "$target" "aspect-publisher publish $upload_id"
