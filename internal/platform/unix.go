//go:build !windows

package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func ConfigureBackground(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	return nil
}

func RunForeground(cmd *exec.Cmd) error {
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	return cmd.Run()
}

func KillProcess(pid int) error {
	process, err := findProcess(pid)
	if err != nil {
		return err
	}

	return process.Signal(syscall.SIGTERM)
}

func findProcess(pid int) (*os.Process, error) {
	return os.FindProcess(pid)
}

func ProcessAlive(pid int) bool {
	process, err := findProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

const CMAKE_PACKAGE_PATH = ".cmake/packages/RiscvSdk/riscv-sdk"
const CMAKE_PACKAGE_DIR_PATH = ".cmake/packages/RiscvSdk"

func readSdkRegistration() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find user home directory, %w", err)
	}

	path := filepath.Join(home, CMAKE_PACKAGE_PATH)

	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return "", ErrSdkRootNotRegistered
	}

	file, err := os.Open(path)
	if err != nil {
		return "", ErrSdkRootNotRegistered
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read SDK registration: %w", err)
		}
		return "", ErrSdkRootNotRegistered
	}

	registeredPath := strings.TrimSpace(scanner.Text())
	if registeredPath == "" {
		return "", ErrSdkRootNotRegistered
	}

	return registeredPath, nil
}

func RegisterSdk(target string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find user home directory, %w", err)
	}

	dir := filepath.Join(home, CMAKE_PACKAGE_DIR_PATH)
	err = os.MkdirAll(dir, 0755)

	if err != nil {
		return fmt.Errorf("cannot create directory %s: %w", CMAKE_PACKAGE_DIR_PATH, err)
	}

	path := filepath.Join(home, CMAKE_PACKAGE_PATH)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("cannot open file %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(target)
	if err != nil {
		return fmt.Errorf("cannot write to file %w", err)
	}

	return nil
}

func UnregisterSdk() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find user home directory, %w", err)
	}
	path := filepath.Join(home, CMAKE_PACKAGE_DIR_PATH)
	return os.RemoveAll(path)
}
