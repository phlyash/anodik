#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PACKAGE="$ROOT/build/package-release.sh"
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
CAPTURE="$TMP/go-calls"
mkdir -p "$BIN"
FAKE_GO="$BIN/go"

cat >"$FAKE_GO" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

output=
previous=
for argument in "$@"; do
  if [[ $previous == -o ]]; then
    output=$argument
    break
  fi
  previous=$argument
done

[[ -n $output ]]
printf '%s/%s/%s\t' "$CGO_ENABLED" "$GOOS" "$GOARCH" >>"$GO_CAPTURE"
printf '%q ' "$@" >>"$GO_CAPTURE"
printf '\n' >>"$GO_CAPTURE"
printf '%s/%s\n' "$GOOS" "$GOARCH" >"$output"
EOF
chmod 700 "$FAKE_GO"

expect_failure env GO_BIN="$FAKE_GO" GO_CAPTURE="$CAPTURE" "$PACKAGE"

mkdir "$TMP/non-empty"
printf 'keep\n' >"$TMP/non-empty/existing"
expect_failure env GO_BIN="$FAKE_GO" GO_CAPTURE="$CAPTURE" \
  "$PACKAGE" "$TMP/non-empty"
[[ $(<"$TMP/non-empty/existing") == keep ]] || fail 'non-empty output was modified'

mkdir "$TMP/symlink-target"
ln -s "$TMP/symlink-target" "$TMP/symlink-output"
expect_failure env GO_BIN="$FAKE_GO" GO_CAPTURE="$CAPTURE" \
  "$PACKAGE" "$TMP/symlink-output"

GO_BIN="$FAKE_GO" GO_CAPTURE="$CAPTURE" "$PACKAGE" "$TMP/out"

expected=(
  anodik-linux-x86_64.tar.gz
  anodik-linux-x86_64.zip
  anodik-darwin-aarch64.tar.gz
  anodik-darwin-aarch64.zip
  anodik-windows-x86_64.tar.gz
  anodik-windows-x86_64.zip
)
mapfile -t actual < <(find "$TMP/out" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)
mapfile -t wanted < <(printf '%s\n' "${expected[@]}" | sort)
[[ ${actual[*]} == "${wanted[*]}" ]] || fail 'wrong archive set'

mapfile -t calls <"$CAPTURE"
[[ ${#calls[@]} == 3 ]] || fail "expected three Go builds, got ${#calls[@]}"
[[ ${calls[0]} == $'0/linux/amd64\t'*'/bin/anodik . ' ]] ||
  fail "unexpected Linux build: ${calls[0]}"
[[ ${calls[1]} == $'0/darwin/arm64\t'*'/bin/anodik . ' ]] ||
  fail "unexpected Darwin build: ${calls[1]}"
[[ ${calls[2]} == $'0/windows/amd64\t'*'/bin/anodik.exe . ' ]] ||
  fail "unexpected Windows build: ${calls[2]}"
for call in "${calls[@]}"; do
  [[ $call == *'build -trimpath -ldflags=-s\ -w -o '* ]] ||
    fail "missing release build flags: $call"
done

python3 - "$TMP/out" <<'PY'
import pathlib
import stat
import sys
import tarfile
import zipfile

root = pathlib.Path(sys.argv[1])
targets = {
    "anodik-linux-x86_64": ("bin/anodik", b"linux/amd64\n", True),
    "anodik-darwin-aarch64": ("bin/anodik", b"darwin/arm64\n", True),
    "anodik-windows-x86_64": ("bin/anodik.exe", b"windows/amd64\n", False),
}

for base, (entry, payload, executable) in targets.items():
    with tarfile.open(root / f"{base}.tar.gz", "r:gz") as archive:
        files = [item for item in archive.getmembers() if item.isfile()]
        assert [item.name.removeprefix("./") for item in files] == [entry]
        extracted = archive.extractfile(files[0])
        assert extracted is not None
        assert extracted.read() == payload
        if executable:
            assert files[0].mode & stat.S_IXUSR

    with zipfile.ZipFile(root / f"{base}.zip") as archive:
        files = [name for name in archive.namelist() if not name.endswith("/")]
        assert files == [entry]
        assert archive.read(entry) == payload
PY

expect_failure env GO_BIN="$FAKE_GO" GO_CAPTURE="$CAPTURE" \
  "$PACKAGE" "$TMP/out"

printf 'PASS: release packaging contract\n'
