package process

import (
	"anodik/internal/platform"
	"errors"
	"fmt"
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

func (m *Manager) Run(name string, args []string, background bool) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = m.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if background {
		if err := platform.ConfigureBackground(cmd); err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}

		if err := m.writeSession(name, cmd.Process.Pid, ""); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return err
		}

		go func() {
			_ = cmd.Wait()
			if !platform.ProcessAlive(cmd.Process.Pid) {
				_ = m.removeSession(name)
			}
		}()

		return nil
	}

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

func (m *Manager) RunDetachedLogged(name string, args []string, logPath, metadata string) (pid int, alive bool, err error) {
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

	if err := m.writeSession(name, pid, metadata); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 0, false, err
	}

	go func() {
		_ = cmd.Wait()
		if !platform.ProcessAlive(pid) {
			_ = m.removeSession(name)
		}
	}()

	const startupGrace = time.Second
	time.Sleep(startupGrace)

	return pid, platform.ProcessAlive(pid), nil
}

func (m *Manager) Kill(name string) error {
	pid, _, err := m.readSession(name)
	if err != nil {
		return err
	}

	if err := platform.KillProcess(pid); err != nil {
		if !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("kill %s: %w", name, err)
		}
	}

	return m.removeSession(name)
}

func (m *Manager) Session(name string) (pid int, metadata string, ok bool) {
	pid, metadata, err := m.readSession(name)
	if err != nil {
		return 0, "", false
	}
	return pid, metadata, true
}

func (m *Manager) IsAlive(name string) bool {
	pid, _, ok := m.Session(name)
	if !ok {
		return false
	}
	return platform.ProcessAlive(pid)
}

func (m *Manager) sessionPath(name string) string {
	return filepath.Join(
		m.WorkingDir,
		"build",
		sessionName(name),
	)
}

func (m *Manager) writeSession(name string, pid int, metadata string) error {
	path := m.sessionPath(name)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create process session directory: %w", err)
	}

	content := strconv.Itoa(pid) + "\n"
	if metadata != "" {
		content += metadata + "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write process session: %w", err)
	}

	return nil
}

func (m *Manager) readSession(name string) (pid int, metadata string, err error) {
	data, err := os.ReadFile(m.sessionPath(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, "", fmt.Errorf("%s is not running", name)
		}
		return 0, "", fmt.Errorf("read process session: %w", err)
	}

	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)

	pid, convErr := strconv.Atoi(strings.TrimSpace(lines[0]))
	if convErr != nil || pid <= 0 {
		return 0, "", fmt.Errorf("invalid %s process session", name)
	}

	if len(lines) > 1 {
		metadata = strings.TrimSpace(lines[1])
	}

	return pid, metadata, nil
}

func (m *Manager) removeSession(name string) error {
	err := os.Remove(m.sessionPath(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func sessionName(name string) string {
	name = filepath.Base(name)

	if ext := filepath.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}

	return name + ".session"
}
