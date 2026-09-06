package ocd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	sessionFileName = "openocd.session"
	logFileName     = "openocd.log"
)

func RuntimeDir(workingDirectory string) string {
	/// maybe move openocd runtime dir with logs and session file to somewhere like
	/// .local/share/niiet/openocd/run or smth like this.
	return filepath.Join(workingDirectory, "build")
}

func sessionPath(workingDirectory string) string {
	return filepath.Join(RuntimeDir(workingDirectory), sessionFileName)
}

func LogPath(workingDirectory string) string {
	return filepath.Join(RuntimeDir(workingDirectory), logFileName)
}

type Session struct {
	PID     int
	GDBPort string
}

func WriteSession(workingDirectory string, pid int, gdbPort string) error {
	if err := os.MkdirAll(RuntimeDir(workingDirectory), 0755); err != nil {
		return fmt.Errorf("create openocd runtime dir: %w", err)
	}

	content := fmt.Sprintf("pid=%d\ngdb_port=%s\n", pid, gdbPort)
	if err := os.WriteFile(sessionPath(workingDirectory), []byte(content), 0644); err != nil {
		return fmt.Errorf("write openocd session: %w", err)
	}

	return nil
}

// ReadSession returns the running OpenOCD session, or ok=false if no
// session file exists (nothing is running, or a stale file was cleaned up).
func ReadSession(workingDirectory string) (session Session, ok bool, err error) {
	file, err := os.Open(sessionPath(workingDirectory))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, false, nil
		}
		return Session{}, false, fmt.Errorf("read openocd session: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if key, value, found := strings.Cut(line, "="); found {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return Session{}, false, fmt.Errorf("read openocd session: %w", err)
	}

	pidStr, hasPID := values["pid"]
	if !hasPID {
		return Session{}, false, nil
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return Session{}, false, nil
	}

	return Session{PID: pid, GDBPort: values["gdb_port"]}, true, nil
}

func RemoveSession(workingDirectory string) error {
	err := os.Remove(sessionPath(workingDirectory))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// SessionGDBPort returns the gdb port of a currently running OpenOCD
// session, or "" if none is running. Used by gdb/flash-go to attach to an
// already-started server without re-specifying the port.
func SessionGDBPort(workingDirectory string) string {
	session, ok, err := ReadSession(workingDirectory)
	if err != nil || !ok {
		return ""
	}
	return session.GDBPort
}
