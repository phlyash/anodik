#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PACKAGE="$ROOT/build/package-release.sh"
VERIFY="$ROOT/build/verify-release-files.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

expect_failure() {
  if "$@" >/dev/null 2>&1; then
    fail "expected failure: $*"
  fi
}

BIN="$TMP/bin"
mkdir -p "$BIN"
FAKE_GO="$BIN/go"
cat >"$FAKE_GO" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
output=
previous=
for argument in "$@"; do
  if [[ $previous == -o ]]; then output=$argument; break; fi
  previous=$argument
done
[[ -n $output ]]
printf '%s/%s\n' "$GOOS" "$GOARCH" >"$output"
EOF
chmod 700 "$FAKE_GO"

VALID="$TMP/valid"
GO_BIN="$FAKE_GO" "$PACKAGE" "$VALID"
"$VERIFY" "$VALID"

cp -a "$VALID" "$TMP/missing"
rm "$TMP/missing/anodik-linux-x86_64.zip"
expect_failure "$VERIFY" "$TMP/missing"

cp -a "$VALID" "$TMP/extra"
printf 'unexpected\n' >"$TMP/extra/unexpected"
expect_failure "$VERIFY" "$TMP/extra"

cp -a "$VALID" "$TMP/symlink"
mv "$TMP/symlink/anodik-linux-x86_64.zip" "$TMP/real.zip"
ln -s "$TMP/real.zip" "$TMP/symlink/anodik-linux-x86_64.zip"
expect_failure "$VERIFY" "$TMP/symlink"

cp -a "$VALID" "$TMP/wrapper"
python3 - "$TMP/wrapper/anodik-windows-x86_64.zip" <<'PY'
import pathlib
import sys
import zipfile

path = pathlib.Path(sys.argv[1])
with zipfile.ZipFile(path, "w") as archive:
    archive.writestr("anodik-windows-x86_64/bin/anodik.exe", b"bad")
PY
expect_failure "$VERIFY" "$TMP/wrapper"

cp -a "$VALID" "$TMP/extra-entry"
python3 - "$TMP/extra-entry/anodik-darwin-aarch64.tar.gz" <<'PY'
import io
import pathlib
import sys
import tarfile

path = pathlib.Path(sys.argv[1])
with tarfile.open(path, "w:gz") as archive:
    for name in ("bin/anodik", "notes.txt"):
        info = tarfile.TarInfo(name)
        info.mode = 0o755 if name == "bin/anodik" else 0o644
        payload = name.encode()
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
PY
expect_failure "$VERIFY" "$TMP/extra-entry"

cp -a "$VALID" "$TMP/non-executable"
python3 - "$TMP/non-executable/anodik-linux-x86_64.tar.gz" <<'PY'
import io
import pathlib
import sys
import tarfile

path = pathlib.Path(sys.argv[1])
with tarfile.open(path, "w:gz") as archive:
    info = tarfile.TarInfo("bin/anodik")
    info.mode = 0o644
    payload = b"linux/amd64\n"
    info.size = len(payload)
    archive.addfile(info, io.BytesIO(payload))
PY
expect_failure "$VERIFY" "$TMP/non-executable"

cp -a "$VALID" "$TMP/malformed"
printf 'not a zip\n' >"$TMP/malformed/anodik-darwin-aarch64.zip"
expect_failure "$VERIFY" "$TMP/malformed"

expect_failure "$VERIFY"
expect_failure "$VERIFY" "$TMP/not-present"

printf 'PASS: exact release archive set and payloads\n'
