package discover

import (
	"os"
	"path/filepath"
	"regexp"
)

var projectNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^\s*project\s*\(\s*([A-Za-z_][A-Za-z0-9_-]*)`),
	regexp.MustCompile(`(?im)^\s*project\s*\(\s*"([^"]+)"`),
	regexp.MustCompile(`(?im)^\s*project\s*\(\s*'([^']+)'`),
}

// ParseCMakeProjectName extracts the name passed to project(...) in a
// CMakeLists.txt file. It returns "" if the file doesn't exist or has no
// project() call.
func ParseCMakeProjectName(cmakeListsPath string) string {
	content, err := os.ReadFile(cmakeListsPath)
	if err != nil {
		return ""
	}

	for _, pattern := range projectNamePatterns {
		if match := pattern.FindSubmatch(content); match != nil {
			return string(match[1])
		}
	}

	return ""
}

var executablePatterns = []func(projectName string) *regexp.Regexp{
	func(string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)add_executable\s*\(\s*\$\{PROJECT_NAME\}`)
	},
	func(projectName string) *regexp.Regexp {
		quoted := regexp.QuoteMeta(projectName)
		// The name may or may not be quoted in add_executable(...), same
		// as in project(...); match either form.
		return regexp.MustCompile(`(?i)add_executable\s*\(\s*(?:"` + quoted + `"|'` + quoted + `'|` + quoted + `\b)`)
	},
}

// IsExecutableProject reports whether dir contains a CMakeLists.txt that
// declares a project() and builds it as an executable, either directly by
// name or via ${PROJECT_NAME}. Returns the project name on success.
func IsExecutableProject(dir string) (bool, string) {
	cmakeListsPath := filepath.Join(dir, "CMakeLists.txt")

	projectName := ParseCMakeProjectName(cmakeListsPath)
	if projectName == "" {
		return false, ""
	}

	content, err := os.ReadFile(cmakeListsPath)
	if err != nil {
		return false, ""
	}

	for _, buildPattern := range executablePatterns {
		if buildPattern(projectName).Match(content) {
			return true, projectName
		}
	}

	return false, ""
}
