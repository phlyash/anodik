package discover

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ExamplesDirName = "examples"

func ExamplesDir(sdkRoot string) string {
	return filepath.Join(sdkRoot, ExamplesDirName)
}

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

func BuildDir(sdkRoot, workingDir, exampleName string) string {
	return filepath.Join(ExampleDir(sdkRoot, workingDir, exampleName), "build")
}

func ExampleExists(sdkRoot, name string) bool {
	if sdkRoot == "" || name == "" {
		return false
	}
	ok, _ := IsExecutableProject(filepath.Join(ExamplesDir(sdkRoot), name))
	return ok
}

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
