package cmakebuild

import (
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func compileCommandsLinkPath(projectDir string) string {
	return filepath.Join(projectDir, "build", "compile_commands.json")
}

// LinkCompileCommands exposes buildDir/compile_commands.json at
// projectDir/build/compile_commands.json (symlinked on Unix, copied on
// Windows where unprivileged symlinks aren't reliably available), first
// patching it with toolchain include directories so clangd can resolve
// standard library headers.
func LinkCompileCommands(s *env.Settings, buildDir, projectDir string) error {
	src := filepath.Join(buildDir, "compile_commands.json")
	if _, err := os.Stat(src); err != nil {
		return nil
	}

	dst := compileCommandsLinkPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create project build dir: %w", err)
	}

	if includeDirs := toolchainIncludeDirs(s); len(includeDirs) > 0 {
		if err := patchCompileCommands(src, includeDirs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not patch compile_commands.json: %v\n", err)
		}
	}

	if src == dst {
	/// standalone project
		return nil
	}

	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale compile_commands.json: %w", err)
	}

	if isWindows() {
		return copyFile(src, dst)
	}
	return os.Symlink(src, dst)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// toolchainIncludeDirs returns the include directories clangd needs to
// resolve libc/libstdc++ headers: CLANGD_EXTRA_INCLUDES if set, otherwise
// auto-detected by asking the configured gcc for its search dirs.
func toolchainIncludeDirs(s *env.Settings) []string {
	if s.ClangdExtraIncludes != "" {
		var dirs []string
		for _, p := range strings.Split(s.ClangdExtraIncludes, ";") {
			if p = strings.TrimSpace(p); p != "" {
				dirs = append(dirs, p)
			}
		}
		if len(dirs) > 0 {
			return dirs
		}
	}

	return detectGCCIncludeDirs(s)
}

func detectGCCIncludeDirs(s *env.Settings) []string {
	if s.GCCToolchainBinDir == "" {
		return nil
	}

	gccBin := toolchain.GCC(s)
	if _, err := os.Stat(gccBin); err != nil {
		return nil
	}

	out, err := exec.Command(gccBin, "-print-search-dirs").Output()
	if err != nil {
		return nil
	}

	installPath := parseInstallPath(string(out))
	if installPath == "" {
		return nil
	}

	prefix := toolchain.Prefix(s)
	root := findToolchainRoot(installPath, prefix)

	var dirs []string
	for _, candidate := range []string{
		filepath.Join(root, prefix, "include"),
		filepath.Join(installPath, "include"),
		filepath.Join(installPath, "include-fixed"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			dirs = append(dirs, candidate)
		}
	}

	return dirs
}

func parseInstallPath(gccSearchDirsOutput string) string {
	for _, line := range strings.Split(gccSearchDirsOutput, "\n") {
		if rest, ok := strings.CutPrefix(line, "install:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// findToolchainRoot walks up from installPath (bounded to a few levels,
// matching the python tool's search depth) looking for a directory that
// looks like the toolchain root: one containing bin/ or <prefix>/.
func findToolchainRoot(installPath, prefix string) string {
	root := installPath
	for i := 0; i < 6; i++ {
		parent := filepath.Dir(root)
		if parent == root {
			break
		}

		if isDir(filepath.Join(root, "bin")) || isDir(filepath.Join(root, prefix)) {
			break
		}
		root = parent
	}
	return root
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

type compileCommandEntry map[string]any

func patchCompileCommands(path string, includeDirs []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var commands []compileCommandEntry
	if err := json.Unmarshal(data, &commands); err != nil {
		return err
	}

	modified := false
	for _, cmd := range commands {
		if patchCommandEntry(cmd, includeDirs) {
			modified = true
		}
	}

	if !modified {
		return nil
	}

	patched, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, patched, 0644)
}

// patchCommandEntry inserts -I<dir> flags for any missing includeDirs into
// a single compile_commands.json entry, preferring the "arguments" array
// form and falling back to splitting the "command" string form.
func patchCommandEntry(cmd compileCommandEntry, includeDirs []string) bool {
	if rawArgs, ok := cmd["arguments"].([]any); ok {
		args := make([]string, len(rawArgs))
		for i, a := range rawArgs {
			args[i], _ = a.(string)
		}

		modified := false
		for _, dir := range includeDirs {
			flag := "-I" + dir
			if containsString(args, flag) {
				continue
			}
			args = insertAfterFlag(args, "-xc", flag)
			modified = true
		}

		if modified {
			newArgs := make([]any, len(args))
			for i, a := range args {
				newArgs[i] = a
			}
			cmd["arguments"] = newArgs
		}
		return modified
	}

	commandStr, ok := cmd["command"].(string)
	if !ok {
		return false
	}

	modified := false
	for _, dir := range includeDirs {
		flag := "-I" + dir
		if strings.Contains(commandStr, flag) {
			continue
		}
		if strings.Contains(commandStr, "-xc") {
			commandStr = strings.Replace(commandStr, "-xc", "-xc "+flag, 1)
		} else {
			parts := strings.Fields(commandStr)
			if len(parts) > 1 {
				parts = append(parts[:2], append([]string{flag}, parts[2:]...)...)
				commandStr = strings.Join(parts, " ")
			}
		}
		modified = true
	}

	if modified {
		cmd["command"] = commandStr
	}
	return modified
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func insertAfterFlag(args []string, flag, insert string) []string {
	for i, a := range args {
		if a == flag {
			result := make([]string, 0, len(args)+1)
			result = append(result, args[:i+1]...)
			result = append(result, insert)
			result = append(result, args[i+1:]...)
			return result
		}
	}

	if len(args) > 2 {
		result := make([]string, 0, len(args)+1)
		result = append(result, args[:2]...)
		result = append(result, insert)
		result = append(result, args[2:]...)
		return result
	}

	return append(args, insert)
}
