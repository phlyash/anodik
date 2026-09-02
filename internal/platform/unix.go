//go:build !windows

package platform

import (
	"bufio"
	"errors"
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

var ErrSdkRootNotRegistered = errors.New("sdk root not registered")

func ResolveSdkRoot() (string, error) {

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find user home directory, %w", err)
	}

	path := filepath.Join(home, CMAKE_PACKAGE_PATH)

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrSdkRootNotRegistered
		}

		return "", fmt.Errorf("cannot read SDK registration: %w", err)
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

	registeredPath, err = filepath.Abs(registeredPath)
	if err != nil {
		return "", fmt.Errorf("resolve SDK registration path: %w", err)
	}

	root := registeredPath
	for i := 0; i < 3; i++ {
		root = filepath.Dir(root)
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", ErrSdkRootNotRegistered
	}

	return root, nil
}

func RegisterSdk(target string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find user home directory, %w", err)
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
