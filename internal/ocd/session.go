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

func RuntimeDir(sdkRoot string) string {
	return filepath.Join(sdkRoot, "build")
}

func sessionPath(sdkRoot string) string {
	return filepath.Join(RuntimeDir(sdkRoot), sessionFileName)
}

func LogPath(sdkRoot string) string {
	return filepath.Join(RuntimeDir(sdkRoot), logFileName)
}

type Session struct {
	PID     int
	GDBPort string
}

func WriteSession(sdkRoot string, pid int, gdbPort string) error {
	if err := os.MkdirAll(RuntimeDir(sdkRoot), 0755); err != nil {
		return fmt.Errorf("create openocd runtime dir: %w", err)
	}

	content := fmt.Sprintf("pid=%d\ngdb_port=%s\n", pid, gdbPort)
	if err := os.WriteFile(sessionPath(sdkRoot), []byte(content), 0644); err != nil {
		return fmt.Errorf("write openocd session: %w", err)
	}

	return nil
}

// ReadSession returns the running OpenOCD session, or ok=false if no
// session file exists (nothing is running, or a stale file was cleaned up).
func ReadSession(sdkRoot string) (session Session, ok bool, err error) {
	file, err := os.Open(sessionPath(sdkRoot))
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

func RemoveSession(sdkRoot string) error {
	err := os.Remove(sessionPath(sdkRoot))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// SessionGDBPort returns the gdb port of a currently running OpenOCD
// session, or "" if none is running. Used by gdb/flash-go to attach to an
// already-started server without re-specifying the port.
func SessionGDBPort(sdkRoot string) string {
	session, ok, err := ReadSession(sdkRoot)
	if err != nil || !ok {
		return ""
	}
	return session.GDBPort
}
