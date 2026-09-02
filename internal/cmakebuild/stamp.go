package cmakebuild

import (
	"os"
	"path/filepath"
	"strings"
)

const stampFileName = ".sdk-configure-stamp"

func stampPath(buildDir string) string {
	return filepath.Join(buildDir, stampFileName)
}

// NeedsConfigure reports whether buildDir must be (re)configured: either it
// has never been configured, or the cmake invocation would differ from the
// one that produced the current build directory.
func NeedsConfigure(buildDir string, args []string) bool {
	existing, err := os.ReadFile(stampPath(buildDir))
	if err != nil {
		return true
	}
	return string(existing) != strings.Join(args, "\n")
}

func WriteStamp(buildDir string, args []string) error {
	return os.WriteFile(stampPath(buildDir), []byte(strings.Join(args, "\n")), 0644)
}
