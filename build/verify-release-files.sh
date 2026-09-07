#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 1 || ! -d $1 || -L $1 ]]; then
  printf 'usage: %s RELEASE_DIRECTORY\n' "$0" >&2
  exit 1
fi

RELEASE_DIR=$(cd -- "$1" && pwd -P)
expected_files=(
  anodik-linux-x86_64.tar.gz
  anodik-linux-x86_64.zip
  anodik-darwin-aarch64.tar.gz
  anodik-darwin-aarch64.zip
  anodik-windows-x86_64.tar.gz
  anodik-windows-x86_64.zip
)

mapfile -t actual < <(find "$RELEASE_DIR" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)
mapfile -t expected < <(printf '%s\n' "${expected_files[@]}" | sort)

if [[ ${#actual[@]} != ${#expected[@]} ]]; then
  printf 'release directory must contain exactly six archives\n' >&2
  exit 1
fi

for index in "${!expected[@]}"; do
  if [[ ${actual[index]} != "${expected[index]}" ]]; then
    printf 'unexpected release archive set\n' >&2
    exit 1
  fi

  path="$RELEASE_DIR/${actual[index]}"
  if [[ ! -f $path || -L $path ]]; then
    printf 'release archive must be a regular non-symlink file: %s\n' "${actual[index]}" >&2
    exit 1
  fi
done

python3 - "$RELEASE_DIR" <<'PY'
import pathlib
import stat
import sys
import tarfile
import zipfile


PROFILES = {
    "anodik-linux-x86_64": ("bin/anodik", True),
    "anodik-darwin-aarch64": ("bin/anodik", True),
    "anodik-windows-x86_64": ("bin/anodik.exe", False),
}


def normalized_tar_name(name):
    return name[2:] if name.startswith("./") else name


def verify_tar(path, payload, require_executable):
    with tarfile.open(path, "r:gz") as archive:
        members = archive.getmembers()
        files = []
        for member in members:
            name = normalized_tar_name(member.name)
            if member.isdir() and name in ("", ".", "bin"):
                continue
            if member.isfile() and name == payload:
                files.append(member)
                continue
            raise ValueError("unexpected tar entry: " + member.name)

        if len(files) != 1 or files[0].size <= 0:
            raise ValueError("tar must contain one non-empty executable: " + path.name)
        if require_executable and not files[0].mode & stat.S_IXUSR:
            raise ValueError("Unix tar payload is not executable: " + path.name)


def verify_zip(path, payload, require_executable):
    with zipfile.ZipFile(path) as archive:
        entries = archive.infolist()
        if [entry.filename for entry in entries] != ["bin/", payload]:
            raise ValueError("unexpected zip entries: " + path.name)

        directory, executable = entries
        if not directory.is_dir() or executable.is_dir() or executable.file_size <= 0:
            raise ValueError("invalid zip payload: " + path.name)

        mode = executable.external_attr >> 16
        if stat.S_ISLNK(mode):
            raise ValueError("zip payload must not be a symlink: " + path.name)
        if require_executable and not mode & stat.S_IXUSR:
            raise ValueError("Unix zip payload is not executable: " + path.name)


def main():
    root = pathlib.Path(sys.argv[1])
    for base, (payload, require_executable) in PROFILES.items():
        verify_tar(root / (base + ".tar.gz"), payload, require_executable)
        verify_zip(root / (base + ".zip"), payload, require_executable)


try:
    main()
except (OSError, ValueError, tarfile.TarError, zipfile.BadZipFile) as error:
    print(error, file=sys.stderr)
    raise SystemExit(1)
PY

printf 'Verified six Anodik release archives\n'
