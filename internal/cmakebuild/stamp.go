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
