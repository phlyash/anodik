package env

import (
	"os"
	"path/filepath"
)

const (
	configDirName    = "riscv-sdk"
	toolchainEnvFile = "toolchain.env"
	packagesFile     = "packages.json"
)

func ConfigDir() (string, error) {
	if base, err := os.UserConfigDir(); err == nil {
		return filepath.Join(base, configDirName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDirName), nil
}

func ToolchainEnvPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, toolchainEnvFile), nil
}

func PackagesJSONPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, packagesFile), nil
}
