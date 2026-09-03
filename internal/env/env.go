package env

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Settings struct {
	WorkingDir string
	SDKRoot    string

	GCCToolchainBinDir   string `env:"GCC_TOOLCHAIN_BIN_DIR"`
	ClangToolchainBinDir string `env:"CLANG_TOOLCHAIN_BIN_DIR"`
	RiscvToolchainPrefix string `env:"RISCV_TOOLCHAIN_PREFIX"`
	ConfigForceCompiler  string `env:"CONFIG_FORCE_COMPILER"`
	ClangdExtraIncludes  string `env:"CLANGD_EXTRA_INCLUDES"`
	ToolsBinDir          string `env:"TOOLS_BIN_DIR"`
	PacketsInstallPath   string `env:"PACKETS_INSTALL_PATH"`

	BuildType string `env:"BUILD_TYPE"`

	OpenOCD         string `env:"OPENOCD"`
	GDB             string `env:"GDB"`
	CMake           string `env:"CMAKE"`
	Ninja           string `env:"NINJA"`
	SoC             string `env:"SOC"`
	Interface       string `env:"INTERFACE"`
	GDBPort         string `env:"GDB_PORT"`
	OCDTargetCfg    string `env:"OCD_TARGET_CFG"`
	OCDInterfaceCfg string `env:"OCD_INTERFACE_CFG"`
	OCDExtraArgs    string `env:"OCD_EXTRA_ARGS"`
}

var knownOptions = func() map[string]struct{} {
	t := reflect.TypeOf(Settings{})
	m := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if key := t.Field(i).Tag.Get("env"); key != "" {
			m[key] = struct{}{}
		}
	}
	return m
}()

func hydrateSettings(s *Settings, values map[string]string) {
	rv := reflect.ValueOf(s).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		key := rt.Field(i).Tag.Get("env")
		if key == "" {
			continue
		}
		if v, ok := values[key]; ok {
			rv.Field(i).SetString(v)
		}
	}
}

func EnvKey(fieldName string) string {
	t := reflect.TypeOf(Settings{})
	f, ok := t.FieldByName(fieldName)
	if !ok {
		panic(fmt.Sprintf("env: Settings has no field %q", fieldName))
	}
	tag := f.Tag.Get("env")
	if tag == "" {
		panic(fmt.Sprintf("env: Settings field %q has no env tag", fieldName))
	}
	return tag
}

func EnvValue(settings *Settings, key string) string {
	t := reflect.TypeOf(*settings)
	v := reflect.ValueOf(*settings)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if f.Tag.Get("env") == key {
			return v.Field(i).String()
		}
	}

	panic(fmt.Sprintf("env: Settings has no env key %q", key))
}

func ReadSettingsFromEnv(workingDir, sdkRoot string) (*Settings, error) {
	local := filepath.Join(workingDir, "local.env")
	user, err := ToolchainEnvPath()
	if err != nil {
		return nil, fmt.Errorf("resolve toolchain env path: %w", err)
	}

	localValues, err := parseEnvFile(local)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", local, err)
	}

	userValues, err := parseEnvFile(user)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", user, err)
	}

	values := make(map[string]string, len(userValues)+len(localValues))

	copyInto(values, userValues)
	copyInto(values, localValues)

	for _, value := range os.Environ() {
		name, value, ok := strings.Cut(value, "=")
		if ok {
			if _, known := knownOptions[name]; known {
				values[name] = value
			}
		}
	}

	settings := &Settings{
		WorkingDir: workingDir,
		SDKRoot:    sdkRoot,
	}
	hydrateSettings(settings, values)

	return settings, nil
}

func ReadToolchainEnv() (*Settings, error) {
	path, err := ToolchainEnvPath()
	if err != nil {
		return nil, fmt.Errorf("resolve toolchain env path: %w", err)
	}

	values, err := parseEnvFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	settings := &Settings{}
	hydrateSettings(settings, values)

	return settings, nil
}

// writableKeys lists the env tags that Save is allowed to write. These are
// the settings anodik itself resolves and persists (toolchain component
// paths, install root) — everything else in toolchain.env (build/debug
// settings like SOC, INTERFACE, ...) is left untouched even if present on
// Settings, since the user is expected to edit those by hand.
var writableKeys = []string{
	EnvKey("GCCToolchainBinDir"),
	EnvKey("ClangToolchainBinDir"),
	EnvKey("RiscvToolchainPrefix"),
	EnvKey("ClangdExtraIncludes"),
	EnvKey("ToolsBinDir"),
	EnvKey("PacketsInstallPath"),

	EnvKey("OpenOCD"),
	EnvKey("GDB"),
	EnvKey("CMake"),
	EnvKey("Ninja"),
}

func Save(s *Settings) error {
	path, err := ToolchainEnvPath()
	if err != nil {
		return fmt.Errorf("resolve toolchain env path: %w", err)
	}

	lines, err := readLines(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	pending := make(map[string]string, len(writableKeys))
	for _, key := range writableKeys {
		pending[key] = EnvValue(s, key)
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)

		value, isWritable := pending[key]
		if !isWritable {
			continue
		}

		lines[i] = key + "=" + value
		delete(pending, key)
	}

	for _, key := range writableKeys {
		value, notWritten := pending[key]
		if !notWritten {
			continue
		}
		lines = append(lines, key+"="+value)
	}

	if err := writeLines(path, lines); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func writeLines(path string, lines []string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func copyInto(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func parseEnvFile(path string) (map[string]string, error) {
	values := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return values, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if _, ok := knownOptions[key]; !ok {
			return nil, fmt.Errorf("unknown option %q on line %d", key, lineNumber)
		}

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if value != "" {
			values[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return values, nil
}
