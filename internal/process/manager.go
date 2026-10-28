package process

import (
	"anodik/internal/platform"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	WorkingDir string
}

func (m *Manager) Run(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = m.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}

	return nil
}

func (m *Manager) RunForeground(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = m.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return platform.RunForeground(cmd)
}

// RunForegroundCaptured behaves like RunForeground, but also returns everything
// the process wrote to stdout and stderr so the caller can inspect it.
func (m *Manager) RunForegroundCaptured(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = m.WorkingDir
	cmd.Stdin = os.Stdin

	var captured bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &captured)
	cmd.Stderr = io.MultiWriter(os.Stderr, &captured)

	err := platform.RunForeground(cmd)

	return captured.String(), err
}

func (m *Manager) RunDetachedLogged(session, name string, args []string, logPath, metadata string) (pid int, alive bool, err error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return 0, false, fmt.Errorf("create log directory: %w", err)
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return 0, false, fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(name, args...)
	cmd.Dir = m.WorkingDir
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := platform.ConfigureBackground(cmd); err != nil {
		return 0, false, err
	}

	if err := cmd.Start(); err != nil {
		return 0, false, fmt.Errorf("start %s: %w", name, err)
	}

	pid = cmd.Process.Pid

	if err := m.writeSession(session, pid, metadata); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 0, false, err
	}

	go func() {
		_ = cmd.Wait()
		if !platform.ProcessAlive(pid) {
			_ = m.removeSession(session)
		}
	}()

	const startupGrace = time.Second
	time.Sleep(startupGrace)

	return pid, platform.ProcessAlive(pid), nil
}

func (m *Manager) Kill(sessionPath string) error {
	pid, _, err := m.readSession(sessionPath)
	if err != nil {
		return err
	}

	if err := platform.KillProcess(pid); err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("kill process %d: %w", pid, err)
		}
	}

	const killTimeout = 5 * time.Second
	if err := waitForExit(pid, killTimeout); err != nil {
		return err
	}

	return m.removeSession(sessionPath)
}

func (m *Manager) Session(sessionPath string) (pid int, metadata string, ok bool) {
	pid, metadata, err := m.readSession(sessionPath)
	if err != nil {
		return 0, "", false
	}
	return pid, metadata, true
}

func (m *Manager) IsAlive(sessionPath string) bool {
	pid, _, ok := m.Session(sessionPath)
	if !ok {
		return false
	}
	return platform.ProcessAlive(pid)
}

func (m *Manager) writeSession(sessionPath string, pid int, metadata string) error {
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0700); err != nil {
		return fmt.Errorf("create process session directory: %w", err)
	}

	content := strconv.Itoa(pid) + "\n"
	if metadata != "" {
		content += metadata + "\n"
	}

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write process session: %w", err)
	}

	return nil
}

func (m *Manager) readSession(sessionPath string) (pid int, metadata string, err error) {
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, "", fmt.Errorf("no process session at %s", sessionPath)
		}
		return 0, "", fmt.Errorf("read process session: %w", err)
	}

	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)

	pid, convErr := strconv.Atoi(strings.TrimSpace(lines[0]))
	if convErr != nil || pid <= 0 {
		return 0, "", fmt.Errorf("invalid process session %s", sessionPath)
	}

	if len(lines) > 1 {
		metadata = strings.TrimSpace(lines[1])
	}

	return pid, metadata, nil
}

func (m *Manager) removeSession(sessionPath string) error {
	err := os.Remove(sessionPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func waitForExit(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for platform.ProcessAlive(pid) {
		if time.Now().After(deadline) {
			return fmt.Errorf("process %d did not exit within %s", pid, timeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
