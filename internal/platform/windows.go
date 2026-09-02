//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func ConfigureBackground(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return nil
}

func RunForeground(cmd *exec.Cmd) error {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	return cmd.Run()
}

func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return process.Kill()
}

const WIN32_STILL_ACTIVE = 259

func ProcessAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}

	return exitCode == WIN32_STILL_ACTIVE
}

const SDK_REGISTRY_KEY = `Software\Kitware\CMake\Packages\RiscvSdk`

var ErrSdkRootNotRegistered = errors.New("sdk root not registered")

func ResolveSdkRoot() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, SDK_REGISTRY_KEY, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrSdkRootNotRegistered
		}
		return "", fmt.Errorf("open SDK registration: %w", err)
	}
	defer key.Close()

	registeredPath, _, err := key.GetStringValue("Path")
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return "", ErrSdkRootNotRegistered
		}
		return "", fmt.Errorf("read SDK registration: %w", err)
	}

	registeredPath = strings.TrimSpace(registeredPath)
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

func RegisterSdk(path string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, SDK_REGISTRY_KEY, registry.SET_VALUE)
	if err != nil {
		panic(err)
	}
	defer k.Close()

	if err := k.SetStringValue("Path", path); err != nil {
		return err
	}

	return nil
}

func UnregisterSdk() error {
	return registry.DeleteKey(registry.CURRENT_USER, SDK_REGISTRY_KEY)
}
