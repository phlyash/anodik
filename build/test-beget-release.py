#!/usr/bin/env python3
"""Integration tests for the Anodik Aspect incoming-bundle builder."""

import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parent.parent
BUILDER = ROOT / "build" / "prepare-beget-release.py"
FILES = {
    "anodik-linux-x86_64.tar.gz": ("linux", "x86_64", "tar.gz"),
    "anodik-linux-x86_64.zip": ("linux", "x86_64", "zip"),
    "anodik-darwin-aarch64.tar.gz": ("darwin", "aarch64", "tar.gz"),
    "anodik-darwin-aarch64.zip": ("darwin", "aarch64", "zip"),
    "anodik-windows-x86_64.tar.gz": ("windows", "x86_64", "tar.gz"),
    "anodik-windows-x86_64.zip": ("windows", "x86_64", "zip"),
}


def expected_summary(manifest):
    return [
        "Prepared bundle: type=%s version=%s"
        % (manifest["type"], manifest["version"]),
        *[
            "profile=%s/%s/%s sha256=%s"
            % (entry["os"], entry["arch"], entry["archiv"], entry["sha256"])
            for entry in manifest["files"]
        ],
    ]


class PrepareBegetReleaseTests(unittest.TestCase):
    def make_input(self, directory):
        source = Path(directory) / "input"
        source.mkdir()
        for index, name in enumerate(FILES):
            (source / name).write_bytes(("archive-%d\n" % index).encode() * 100)
        return source

    def run_builder(self, source, output, **overrides):
        options = {
            "type": "anodik",
            "tag": "v1.2.3",
            "repository": "phlyash/anodik",
            "run_id": "123456-1",
        }
        options.update(overrides)
        return subprocess.run(
            [
                sys.executable,
                str(BUILDER),
                "--type",
                options["type"],
                "--tag",
                options["tag"],
                "--repository",
                options["repository"],
                "--run-id",
                options["run_id"],
                "--input",
                str(source),
                "--output",
                str(output),
            ],
            text=True,
            capture_output=True,
            check=False,
        )

    def load_builder(self):
        self.assertTrue(BUILDER.is_file(), "prepare-beget-release.py is missing")
        specification = importlib.util.spec_from_file_location(
            "prepare_beget_release", BUILDER
        )
        builder = importlib.util.module_from_spec(specification)
        specification.loader.exec_module(builder)
        return builder

    def assert_failure_without_manifest(self, result, output):
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertFalse((output / "release.json").exists())

    def test_builds_ordered_manifest_and_byte_identical_archives(self):
        """Wrong metadata, order, hash, or copy bytes break the publisher contract."""
        with tempfile.TemporaryDirectory() as temporary:
            source = self.make_input(temporary)
            output = Path(temporary) / "output"
            result = self.run_builder(source, output)

            self.assertEqual(result.returncode, 0, result.stderr)
            manifest = json.loads((output / "release.json").read_text())
            self.assertEqual(
                list(manifest),
                ["schema", "type", "version", "repository", "run_id", "files"],
            )
            self.assertEqual(manifest["schema"], 1)
            self.assertEqual(manifest["type"], "anodik")
            self.assertEqual(manifest["version"], "1.2.3")
            self.assertEqual(manifest["repository"], "phlyash/anodik")
            self.assertEqual(manifest["run_id"], "123456-1")
            self.assertEqual([entry["name"] for entry in manifest["files"]], list(FILES))
            self.assertEqual(
                [
                    (entry["os"], entry["arch"], entry["archiv"])
                    for entry in manifest["files"]
                ],
                list(FILES.values()),
            )
            for entry in manifest["files"]:
                self.assertEqual(
                    list(entry), ["name", "os", "arch", "archiv", "sha256"]
                )
                source_bytes = (source / entry["name"]).read_bytes()
                output_bytes = (output / entry["name"]).read_bytes()
                self.assertEqual(output_bytes, source_bytes)
                self.assertEqual(
                    entry["sha256"], hashlib.sha256(output_bytes).hexdigest()
                )
            self.assertEqual(
                sorted(path.name for path in output.iterdir()),
                sorted([*FILES, "release.json"]),
            )
            self.assertEqual(result.stdout.splitlines(), expected_summary(manifest))

    def test_normalizes_one_optional_leading_v(self):
        """Tags with and without one leading v publish the same safe version."""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for index, tag in enumerate(("v2.4.0", "2.4.0")):
                case = root / str(index)
                case.mkdir()
                source = self.make_input(case)
                output = case / "output"
                result = self.run_builder(source, output, tag=tag)
                self.assertEqual(result.returncode, 0, result.stderr)
                manifest = json.loads((output / "release.json").read_text())
                self.assertEqual(manifest["version"], "2.4.0")

    def test_rejects_unsafe_versions_without_echoing_them(self):
        """Path-like or shell-like tags cannot become paths or leak to logs."""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            unsafe = ("", "v", "../1.2.3", "1 2", "vv1.2.3", '1.2.3"; echo secret')
            for index, tag in enumerate(unsafe):
                case = root / str(index)
                case.mkdir()
                source = self.make_input(case)
                output = case / "output"
                result = self.run_builder(source, output, tag=tag)
                self.assert_failure_without_manifest(result, output)
                if len(tag) > 1:
                    self.assertNotIn(tag, result.stdout + result.stderr)

    def test_rejects_wrong_identity_and_unsafe_run_ids(self):
        """The bundle builder cannot impersonate a different forced identity."""
        cases = (
            {"type": "compiler"},
            {"repository": "phlyash/riscv-toolchain"},
            {"repository": "someone/anodik"},
            {"run_id": "0-1"},
            {"run_id": "123-0"},
            {"run_id": "123"},
            {"run_id": "123-1/escape"},
        )
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for index, overrides in enumerate(cases):
                case = root / str(index)
                case.mkdir()
                source = self.make_input(case)
                output = case / "output"
                self.assert_failure_without_manifest(
                    self.run_builder(source, output, **overrides), output
                )

    def test_rejects_incomplete_extra_and_symlink_inputs(self):
        """Only six regular, non-symlink archives can enter a bundle."""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)

            missing_root = root / "missing"
            missing_root.mkdir()
            missing = self.make_input(missing_root)
            (missing / "anodik-windows-x86_64.zip").unlink()
            self.assert_failure_without_manifest(
                self.run_builder(missing, missing_root / "output"),
                missing_root / "output",
            )

            extra_root = root / "extra"
            extra_root.mkdir()
            extra = self.make_input(extra_root)
            (extra / "unexpected.txt").write_text("unexpected")
            self.assert_failure_without_manifest(
                self.run_builder(extra, extra_root / "output"), extra_root / "output"
            )

            symlink_root = root / "symlink"
            symlink_root.mkdir()
            symlink = self.make_input(symlink_root)
            archive = symlink / "anodik-linux-x86_64.zip"
            real_archive = symlink_root / "real.zip"
            archive.rename(real_archive)
            archive.symlink_to(real_archive)
            self.assert_failure_without_manifest(
                self.run_builder(symlink, symlink_root / "output"),
                symlink_root / "output",
            )

    def test_rejects_invalid_output_and_overlapping_paths(self):
        """Output preconditions prevent input mutation and mixed bundles."""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = self.make_input(root)

            non_empty = root / "non-empty"
            non_empty.mkdir()
            (non_empty / "keep").write_text("keep")
            self.assert_failure_without_manifest(
                self.run_builder(source, non_empty), non_empty
            )
            self.assertEqual((non_empty / "keep").read_text(), "keep")

            target = root / "target"
            target.mkdir()
            symlink = root / "output-link"
            symlink.symlink_to(target, target_is_directory=True)
            self.assert_failure_without_manifest(self.run_builder(source, symlink), symlink)

            nested = source / "output"
            self.assert_failure_without_manifest(self.run_builder(source, nested), nested)

            regular_file = root / "regular-file"
            regular_file.write_text("not a directory")
            self.assert_failure_without_manifest(
                self.run_builder(regular_file, root / "file-output"),
                root / "file-output",
            )

    def test_hashes_copied_archive_even_if_source_changes_after_copy(self):
        """Manifest hashes describe shipped bytes, not a later source mutation."""
        builder = self.load_builder()
        with tempfile.TemporaryDirectory() as temporary:
            source = self.make_input(temporary)
            output = Path(temporary) / "output"
            name = "anodik-linux-x86_64.tar.gz"
            real_copyfile = builder.shutil.copyfile

            def copy_then_mutate(copy_source, destination):
                result = real_copyfile(copy_source, destination)
                if Path(copy_source).name == name:
                    Path(copy_source).write_bytes(b"mutated source\n")
                return result

            arguments = [
                str(BUILDER),
                "--type", "anodik",
                "--tag", "v1.2.3",
                "--repository", "phlyash/anodik",
                "--run-id", "123456-1",
                "--input", str(source),
                "--output", str(output),
            ]
            with mock.patch.object(sys, "argv", arguments):
                with mock.patch.object(
                    builder.shutil, "copyfile", side_effect=copy_then_mutate
                ):
                    builder.main()

            manifest = json.loads((output / "release.json").read_text())
            entry = next(item for item in manifest["files"] if item["name"] == name)
            destination_hash = hashlib.sha256((output / name).read_bytes()).hexdigest()
            source_hash = hashlib.sha256((source / name).read_bytes()).hexdigest()
            self.assertEqual(entry["sha256"], destination_hash)
            self.assertNotEqual(entry["sha256"], source_hash)

    def test_manifest_is_atomically_replaced_from_output_directory(self):
        """The visible manifest appears through one output-local atomic replace."""
        builder = self.load_builder()
        with tempfile.TemporaryDirectory() as temporary:
            source = self.make_input(temporary)
            output = Path(temporary) / "output"
            real_replace = builder.os.replace
            replacements = []

            def observe_replace(source_path, destination_path):
                replacements.append((Path(source_path), Path(destination_path)))
                return real_replace(source_path, destination_path)

            arguments = [
                str(BUILDER),
                "--type", "anodik",
                "--tag", "v1.2.3",
                "--repository", "phlyash/anodik",
                "--run-id", "123456-1",
                "--input", str(source),
                "--output", str(output),
            ]
            with mock.patch.object(sys, "argv", arguments):
                with mock.patch.object(builder.os, "replace", side_effect=observe_replace):
                    builder.main()

            self.assertEqual(len(replacements), 1)
            temporary_path, destination_path = replacements[0]
            self.assertEqual(temporary_path.parent, output)
            self.assertTrue(temporary_path.name.startswith(".release-"))
            self.assertEqual(destination_path, output / "release.json")
            self.assertFalse(temporary_path.exists())

    def test_serialization_failure_removes_partial_manifest(self):
        """A JSON write failure leaves neither manifest nor temporary fragment."""
        builder = self.load_builder()
        with tempfile.TemporaryDirectory() as temporary:
            source = self.make_input(temporary)
            output = Path(temporary) / "output"

            def write_partial_then_fail(_manifest, stream, **_kwargs):
                stream.write("{")
                raise OSError("simulated JSON serialization failure")

            arguments = [
                str(BUILDER),
                "--type", "anodik",
                "--tag", "v1.2.3",
                "--repository", "phlyash/anodik",
                "--run-id", "123456-1",
                "--input", str(source),
                "--output", str(output),
            ]
            with mock.patch.object(sys, "argv", arguments):
                with mock.patch.object(
                    builder.json, "dump", side_effect=write_partial_then_fail
                ):
                    with self.assertRaisesRegex(
                        OSError, "simulated JSON serialization failure"
                    ):
                        builder.main()

            self.assertFalse((output / "release.json").exists())
            self.assertEqual(list(output.glob(".release-*")), [])

    def test_copy_failure_never_creates_completed_manifest(self):
        """A partial archive copy cannot make the bundle look publishable."""
        builder = self.load_builder()
        with tempfile.TemporaryDirectory() as temporary:
            source = self.make_input(temporary)
            output = Path(temporary) / "output"
            arguments = [
                str(BUILDER),
                "--type", "anodik",
                "--tag", "v1.2.3",
                "--repository", "phlyash/anodik",
                "--run-id", "123456-1",
                "--input", str(source),
                "--output", str(output),
            ]
            with mock.patch.object(sys, "argv", arguments):
                with mock.patch.object(
                    builder.shutil,
                    "copyfile",
                    side_effect=OSError("simulated copy failure"),
                ):
                    with self.assertRaisesRegex(OSError, "simulated copy failure"):
                        builder.main()
            self.assertFalse((output / "release.json").exists())


if __name__ == "__main__":
    unittest.main()
