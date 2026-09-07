#!/usr/bin/env python3
"""Security and data-flow contract tests for the release workflow."""

import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"
ARCHIVES = (
    "anodik-linux-x86_64.tar.gz",
    "anodik-linux-x86_64.zip",
    "anodik-darwin-aarch64.tar.gz",
    "anodik-darwin-aarch64.zip",
    "anodik-windows-x86_64.tar.gz",
    "anodik-windows-x86_64.zip",
)


def job_block(workflow, name):
    match = re.search(
        rf"(?ms)^  {re.escape(name)}:\n.*?(?=^  [A-Za-z0-9_-]+:\n|\Z)",
        workflow,
    )
    if match is None:
        raise AssertionError("missing workflow job: " + name)
    return match.group(0)


def assert_active_line(test_case, text, line):
    test_case.assertRegex(text, rf"(?m)^{re.escape(line)}$")


def assert_workflow_contract(test_case, workflow):
    assert_active_line(test_case, workflow, "  workflow_dispatch:")
    test_case.assertRegex(
        workflow,
        r"(?m)^      release_tag:\n"
        r"^        description: Git tag used for the published GitHub Release\n"
        r"^        required: true\n"
        r"^        type: string$",
    )
    test_case.assertRegex(
        workflow, r"(?m)^  push:\n^    branches:\n^      - master$"
    )
    test_case.assertRegex(
        workflow, r"(?m)^  pull_request:\n^    branches:\n^      - master$"
    )
    test_case.assertRegex(
        workflow, r"(?m)^permissions:\n^  contents: read$"
    )

    build = job_block(workflow, "build")
    for required in (
        "actions/checkout@v6",
        "actions/setup-go@v7",
        "go-version-file: go.mod",
        "CGO_ENABLED=0 go test ./...",
        "bash build/test-package-release.sh",
        "bash build/test-release-files.sh",
        "python3 build/test-beget-release.py",
        "bash build/test-deploy-beget.sh",
        "python3 build/test-release-workflow.py",
        "bash build/package-release.sh dist/release",
        "bash build/verify-release-files.sh dist/release",
        "dist/smoke/bin/anodik help",
    ):
        test_case.assertIn(required, build)
    test_case.assertEqual(build.count("uses: actions/upload-artifact@v7"), 6)
    for archive in ARCHIVES:
        test_case.assertIn("name: " + archive, build)
        test_case.assertIn("path: dist/release/" + archive, build)

    release = job_block(workflow, "release")
    assert_active_line(
        test_case, release, "    if: github.event_name == 'workflow_dispatch'"
    )
    assert_active_line(test_case, release, "    needs: build")
    assert_active_line(test_case, release, "      contents: write")
    test_case.assertIn("uses: actions/download-artifact@v8", release)
    test_case.assertIn("pattern: anodik-*", release)
    test_case.assertIn("merge-multiple: true", release)
    test_case.assertIn("bash build/verify-release-files.sh release-assets", release)
    test_case.assertIn("python3 build/prepare-beget-release.py", release)
    test_case.assertIn("--type anodik", release)
    test_case.assertIn('--tag "$RELEASE_TAG"', release)
    test_case.assertIn("--output release-check", release)
    test_case.assertIn('gh release create "$RELEASE_TAG" release-assets/*', release)
    test_case.assertIn("RELEASE_TAG: ${{ inputs.release_tag }}", release)

    deploy = job_block(workflow, "deploy-beget")
    assert_active_line(
        test_case, deploy, "    if: github.event_name == 'workflow_dispatch'"
    )
    assert_active_line(test_case, deploy, "    needs: release")
    test_case.assertIn("uses: actions/download-artifact@v8", deploy)
    test_case.assertIn("bash build/verify-release-files.sh release-assets", deploy)
    test_case.assertIn("--type anodik", deploy)
    test_case.assertIn('--tag "$RELEASE_TAG"', deploy)
    test_case.assertIn('--repository "$GITHUB_REPOSITORY"', deploy)
    test_case.assertIn(
        '--run-id "$GITHUB_RUN_ID-$GITHUB_RUN_ATTEMPT"', deploy
    )
    test_case.assertIn("run: bash build/deploy-beget-release.sh aspect-upload", deploy)
    for secret in (
        "BEGET_HOST",
        "BEGET_PORT",
        "BEGET_USER",
        "BEGET_SSH_PRIVATE_KEY",
        "BEGET_KNOWN_HOSTS",
    ):
        test_case.assertIn(f"{secret}: ${{{{ secrets.{secret} }}}}", deploy)

    test_case.assertEqual(workflow.count("contents: write"), 1)
    test_case.assertNotIn("ssh-keyscan", workflow)


class ReleaseWorkflowTests(unittest.TestCase):
    def workflow(self):
        self.assertTrue(WORKFLOW.is_file(), "release workflow is missing")
        return WORKFLOW.read_text()

    def test_ci_and_manual_publication_data_flow(self):
        """Missing build gates or manual-only dependencies can publish unsafe input."""
        assert_workflow_contract(self, self.workflow())

    def test_commented_manual_gate_is_not_accepted(self):
        """A commented deploy guard must not count as a publication gate."""
        workflow = self.workflow().replace(
            "    if: github.event_name == 'workflow_dispatch'\n"
            "    needs: release\n",
            "    # if: github.event_name == 'workflow_dispatch'\n"
            "    needs: release\n",
            1,
        )
        with self.assertRaises(AssertionError):
            assert_workflow_contract(self, workflow)

    def test_commented_release_dependency_is_not_accepted(self):
        """A deploy job without active needs: release could upload failed releases."""
        workflow = self.workflow().replace(
            "    needs: release\n",
            "    # needs: release\n",
            1,
        )
        with self.assertRaises(AssertionError):
            assert_workflow_contract(self, workflow)

    def test_release_tag_is_one_quoted_argument_to_the_real_builder(self):
        """Shell metacharacters in a manual tag cannot execute before validation."""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "input"
            output = root / "output"
            executable_directory = root / "bin"
            source.mkdir()
            executable_directory.mkdir()
            for archive in ARCHIVES:
                (source / archive).write_bytes(b"fixture\n")

            marker = root / "shell-marker"
            capture = root / "python-arguments"
            wrapper = executable_directory / "python3"
            wrapper.write_text(
                "#!/usr/bin/env bash\n"
                "printf '%s\\n' \"$@\" >\"$ARGUMENT_CAPTURE\"\n"
                "exec \"$REAL_PYTHON\" \"$@\"\n"
            )
            wrapper.chmod(0o700)
            malicious_tag = 'v1.2.3"; touch "$MARKER"; $(touch "$MARKER"); #'
            command = """python3 build/prepare-beget-release.py \\
  --type anodik \\
  --tag "$RELEASE_TAG" \\
  --repository "$GITHUB_REPOSITORY" \\
  --run-id "$GITHUB_RUN_ID-$GITHUB_RUN_ATTEMPT" \\
  --input "$INPUT" \\
  --output "$OUTPUT"
"""
            result = subprocess.run(
                ["bash", "-c", command],
                cwd=ROOT,
                env={
                    **os.environ,
                    "PATH": str(executable_directory) + os.pathsep + os.environ["PATH"],
                    "ARGUMENT_CAPTURE": str(capture),
                    "REAL_PYTHON": sys.executable,
                    "RELEASE_TAG": malicious_tag,
                    "GITHUB_REPOSITORY": "phlyash/anodik",
                    "GITHUB_RUN_ID": "123456",
                    "GITHUB_RUN_ATTEMPT": "1",
                    "INPUT": str(source),
                    "OUTPUT": str(output),
                    "MARKER": str(marker),
                },
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue(capture.is_file(), result.stdout + result.stderr)
            arguments = capture.read_text().splitlines()
            self.assertEqual(arguments[arguments.index("--tag") + 1], malicious_tag)
            self.assertEqual(arguments.count(malicious_tag), 1)
            self.assertFalse(marker.exists())
            self.assertFalse((output / "release.json").exists())
            self.assertNotIn(malicious_tag, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
