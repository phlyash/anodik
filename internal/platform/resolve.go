package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

type Context struct {
	WorkingDir string
	SDKRoot    string
}

func RuntimeBaseDir() string {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "niiet")
}

func Resolve() (Context, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return Context{}, fmt.Errorf("resolve working directory: %w", err)
	}

	sdkRoot, err := ResolveSdkRoot()
	if err != nil {
		return Context{WorkingDir: workingDir}, err
	}

	return Context{WorkingDir: workingDir, SDKRoot: sdkRoot}, nil
}
