package discover

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ExamplesDirName = "examples"

// ExamplesDir returns <sdkRoot>/examples. sdkRoot may be empty if the SDK
// is not registered; callers must handle that case before calling this.
func ExamplesDir(sdkRoot string) string {
	return filepath.Join(sdkRoot, ExamplesDirName)
}

// ExampleDir resolves the source directory for a named example: an SDK
// example directory if the name matches one, otherwise the working
// directory itself (the project the user is currently standing in).
func ExampleDir(sdkRoot, workingDir, exampleName string) string {
	if sdkRoot != "" && exampleName != "" {
		candidate := filepath.Join(ExamplesDir(sdkRoot), exampleName)
		if isDir(candidate) {
			if ok, _ := IsExecutableProject(candidate); ok {
				return candidate
			}
		}
	}

	return workingDir
}

// ProjectDir returns the base directory used for shared artifacts such as
// compile_commands.json: the SDK root if the example lives inside
// <sdkRoot>/examples, otherwise the example's own source directory.
func ProjectDir(sdkRoot, workingDir, exampleName string) string {
	srcDir := ExampleDir(sdkRoot, workingDir, exampleName)

	if sdkRoot == "" {
		return srcDir
	}

	rel, err := filepath.Rel(ExamplesDir(sdkRoot), srcDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return srcDir
	}

	return sdkRoot
}

// BuildDir returns <exampleSrcDir>/build for the resolved example.
func BuildDir(sdkRoot, workingDir, exampleName string) string {
	return filepath.Join(ExampleDir(sdkRoot, workingDir, exampleName), "build")
}

// ExampleExists reports whether name is a valid executable project inside
// <sdkRoot>/examples, without falling back to the working directory the way
// ExampleDir does. Used to check "is this word actually an example name"
// before committing to it (e.g. the unknown-command build fallback).
func ExampleExists(sdkRoot, name string) bool {
	if sdkRoot == "" || name == "" {
		return false
	}
	ok, _ := IsExecutableProject(filepath.Join(ExamplesDir(sdkRoot), name))
	return ok
}

// DiscoverExamples lists valid executable projects directly inside dir:
// dir itself if it qualifies, plus any immediate non-hidden subdirectory
// that qualifies.
func DiscoverExamples(dir string) []string {
	var names []string

	if !isDir(dir) {
		return names
	}

	if ok, _ := IsExecutableProject(dir); ok {
		names = append(names, filepath.Base(dir))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return names
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		sub := filepath.Join(dir, entry.Name())
		if ok, _ := IsExecutableProject(sub); ok {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)
	return names
}

// DetectCurrentExample walks up from workingDir looking for the nearest
// directory that is a valid executable project, stopping once it reaches
// sdkRoot (inclusive) or the filesystem root. Returns "" if none is found.
func DetectCurrentExample(sdkRoot, workingDir string) string {
	dir := workingDir

	for {
		if ok, _ := IsExecutableProject(dir); ok {
			return filepath.Base(dir)
		}

		if dir == sdkRoot {
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
