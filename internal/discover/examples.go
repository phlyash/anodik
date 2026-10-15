package discover

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const ExamplesDirName = "examples"

func ExamplesDir(sdkRoot string) string {
	return filepath.Join(sdkRoot, ExamplesDirName)
}

func ExampleDir(sdkRoot, workingDir, exampleName string) string {
	current := currentProjectDir(sdkRoot, workingDir)
	if current != "" && filepath.Base(current) == path.Base(exampleName) {
		return current
	}

	if sdkRoot != "" && exampleName != "" {
		candidate := filepath.Join(ExamplesDir(sdkRoot), exampleName)
		if isDir(candidate) {
			if ok, _ := IsExecutableProject(candidate); ok {
				return candidate
			}
		}
	}

	if current != "" {
		return current
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
	const maxDepth = 3

	var names []string

	if !isDir(dir) {
		return names
	}

	type item struct {
		path  string
		name  string // relative name, empty for root
		depth int
	}

	queue := []item{{path: dir}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if ok, _ := IsExecutableProject(cur.path); ok {
			name := cur.name
			if name == "" {
				name = filepath.Base(dir)
			}
			names = append(names, name)
		}

		if cur.depth >= maxDepth {
			continue
		}

		entries, err := os.ReadDir(cur.path)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			queue = append(queue, item{
				path:  filepath.Join(cur.path, e.Name()),
				name:  path.Join(cur.name, e.Name()),
				depth: cur.depth + 1,
			})
		}
	}

	sort.Strings(names)
	return names
}

func DetectCurrentExample(sdkRoot, workingDir string) string {
	if dir := currentProjectDir(sdkRoot, workingDir); dir != "" {
		return filepath.Base(dir)
	}
	return ""
}

// currentProjectDir is the executable project containing workingDir (the
// directory itself or its nearest ancestor, not going above sdkRoot), or "".
func currentProjectDir(sdkRoot, workingDir string) string {
	if workingDir == "" {
		return ""
	}
	dir := workingDir

	for {
		if ok, _ := IsExecutableProject(dir); ok {
			return dir
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
