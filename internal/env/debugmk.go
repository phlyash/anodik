package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type BuildConfig struct {
	TargetName      string // TARGET_NAME
	SoC             string // TARGET_SOC
	Linker          string // TARGET_LINKER
	Interface       string // TARGET_INTERFACE
	GDBPort         string // TARGET_GDB_PORT
	OCDTargetCfg    string // TARGET_OCD_TARGET_CFG
	OCDInterfaceCfg string // TARGET_OCD_INTERFACE_CFG
	OCDExtraArgs    string // TARGET_OCD_EXTRA_ARGS
}

var ErrDebugMkNotFound = fmt.Errorf("debug.mk not found: project not configured yet")

func ReadDebugMK(buildDir string) (*BuildConfig, error) {
	path := filepath.Join(buildDir, "debug.mk")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDebugMkNotFound
		}
		return nil, fmt.Errorf("open debug.mk: %w", err)
	}
	defer f.Close()

	raw, err := parseDebugMk(f)
	if err != nil {
		return nil, fmt.Errorf("parse debug.mk: %w", err)
	}

	cfg := &BuildConfig{
		TargetName:      raw["TARGET_NAME"],
		SoC:             raw["TARGET_SOC"],
		Linker:          raw["TARGET_LINKER"],
		Interface:       raw["TARGET_INTERFACE"],
		GDBPort:         raw["TARGET_GDB_PORT"],
		OCDTargetCfg:    raw["TARGET_OCD_TARGET_CFG"],
		OCDInterfaceCfg: raw["TARGET_OCD_INTERFACE_CFG"],
		OCDExtraArgs:    raw["TARGET_OCD_EXTRA_ARGS"],
	}

	return cfg, nil
}

// parseDebugMk разбирает KEY := VALUE строки.
// Комментарии (#) и пустые строки пропускаются.
func parseDebugMk(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, ":=")
		if !ok {
			continue
		}

		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
