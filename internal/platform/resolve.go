package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrSdkRootNotRegistered = errors.New("sdk root not registered")

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

func ResolveSdkRoot() (string, error) {
	registeredPath, err := readSdkRegistration()
	if err != nil {
		return "", err
	}

	return sdkRootFromRegisteredPath(registeredPath)
}

func sdkRootFromRegisteredPath(registeredPath string) (string, error) {
	registeredPath, err := filepath.Abs(registeredPath)
	if err != nil {
		return "", fmt.Errorf("resolve SDK registration path: %w", err)
	}

	if info, err := os.Stat(registeredPath); err != nil || !info.IsDir() {
		return "", ErrSdkRootNotRegistered
	}

	root := registeredPath
	for i := 0; i < 3; i++ {
		root = filepath.Dir(root)
	}

	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return "", ErrSdkRootNotRegistered
	}

	return root, nil
}
