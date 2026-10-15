package packetmanager

import (
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"time"
)

type DemandState = int

const (
	Required DemandState = iota
	Optional
)

const RepoUrl = "http://aspect.niiet.ru/packages.json"

type Tool struct {
	LogName     string
	PacketName  PacketType
	CommandName func(*env.Settings) string
	Demand      DemandState
	MinVersion  version
	GetVersion  func(command string) version
	Check       func(command string) error
}

var RequiredToolTable = []Tool{
	{"cmake", cmake, toolchain.CMake, Required, version{3, 20}, cmakeVersion, anyCheck},
	{"ninja", ninja, toolchain.Ninja, Required, version{0, 0}, ninjaVersion, anyCheck},
	{"openocd", openocd, toolchain.OpenOCD, Required, version{0, 0}, anyVersion, anyCheck},
	{"riscv gcc", compiler, toolchain.GCC, Required, version{0, 0}, gccVersion, gccCheck},
	{"riscv gdb", compiler, toolchain.GDB, Required, version{0, 0}, gdbVersion, anyCheck},
	{"clangd", compiler, toolchain.Clangd, Optional, version{0, 0}, anyVersion, anyCheck},
	{"clang-format", compiler, toolchain.ClangFormat, Optional, version{0, 0}, anyVersion, anyCheck},
	{"clang-tidy", compiler, toolchain.ClangTidy, Optional, version{0, 0}, anyVersion, anyCheck},
}

type RequiredEnvSetter func(*env.Settings, string) (*env.Settings, error)

type RequiredEnvRemover func(*env.Settings) *env.Settings

var RequiredEnvTable = map[PacketType]RequiredEnvSetter{
	cmake:    cmakeRequiredEnv,
	ninja:    ninjaRequiredEnv,
	openocd:  openocdRequiredEnv,
	compiler: compilerRequiredEnv,
}

var CleanEnvTable = map[PacketType]RequiredEnvRemover{
	cmake:    cmakeRequiredEnvRemove,
	ninja:    ninjaRequiredEnvRemove,
	openocd:  openocdRequiredEnvRemove,
	compiler: compilerRequiredEnvRemove,
}

func cmakeRequiredEnv(s *env.Settings, installDir string) (*env.Settings, error) {
	s.CMake = filepath.Join(installDir, "bin", toolchain.BinName("cmake"))
	return s, nil
}

func cmakeRequiredEnvRemove(s *env.Settings) *env.Settings {
	s.CMake = ""
	return s
}

func ninjaRequiredEnv(s *env.Settings, installDir string) (*env.Settings, error) {
	s.Ninja = filepath.Join(installDir, "bin", toolchain.BinName("ninja"))
	return s, nil
}

func ninjaRequiredEnvRemove(s *env.Settings) *env.Settings {
	s.Ninja = ""
	return s
}

func openocdRequiredEnv(s *env.Settings, installDir string) (*env.Settings, error) {
	s.OpenOCD = filepath.Join(installDir, "bin", toolchain.BinName("openocd"))
	return s, nil
}

func openocdRequiredEnvRemove(s *env.Settings) *env.Settings {
	s.OpenOCD = ""
	return s
}

func compilerRequiredEnv(s *env.Settings, installDir string) (*env.Settings, error) {
	dir := filepath.Join(installDir, "bin")
	s.GCCToolchainBinDir = dir
	s.ClangToolchainBinDir = dir
	s.ToolsBinDir = dir
	return s, nil
}

func compilerRequiredEnvRemove(s *env.Settings) *env.Settings {
	s.GCCToolchainBinDir = ""
	s.ClangToolchainBinDir = ""
	s.ToolsBinDir = ""
	return s
}

func (p Packet) SaveRequiredPaths(resolved_settings *env.Settings, installDir string) error {
	setter, ok := RequiredEnvTable[p.Type]
	if !ok {
		return nil
	}

	toolchain_settings, err := env.ReadToolchainEnv()
	if err != nil {
		toolchain_settings = resolved_settings
	}

	toolchain_settings, err = setter(toolchain_settings, installDir)
	if err != nil {
		return fmt.Errorf("resolve required paths for %s: %w", p.Type, err)
	}

	return env.Save(toolchain_settings)
}

const requestTimeout = 5 * time.Minute

func httpClient(proxy string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("parse proxy %q: %w", proxy, err)
		}
		if parsed.Host == "" {
			return nil, fmt.Errorf("proxy %q has no host: expected scheme://host:port", proxy)
		}
		transport.Proxy = http.ProxyURL(parsed)
	}

	return &http.Client{Timeout: requestTimeout, Transport: transport}, nil
}

func QueryPackets(repoUrl string) ([]Packet, error) {
	client, err := httpClient("")
	if err != nil {
		return nil, err
	}

	resp, err := client.Get(repoUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	packets, err := ParsePacketList(data)
	if err != nil {
		return nil, err
	}

	return FilterPackets(packets), nil
}

type ToolStatus struct {
	Name     string `json:"name"`
	Packet   string `json:"packet"`
	Command  string `json:"command"`
	Required bool   `json:"required"`
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
}

func CheckTools(s *env.Settings) []ToolStatus {
	results := make([]ToolStatus, 0, len(RequiredToolTable))
	for idx := range RequiredToolTable {
		tool := &RequiredToolTable[idx]
		status := ToolStatus{
			Name:     tool.LogName,
			Packet:   tool.PacketName,
			Command:  tool.CommandName(s),
			Required: tool.Demand == Required,
			OK:       true,
		}
		if err := testTool(tool, s); err != nil {
			status.OK = false
			status.Message = err.Error()
		}
		results = append(results, status)
	}
	return results
}

// MissingRequired counts the required tools that failed their check.
func MissingRequired(results []ToolStatus) int {
	n := 0
	for _, r := range results {
		if r.Required && !r.OK {
			n++
		}
	}
	return n
}

func PrintDoctor(w io.Writer, results []ToolStatus) {
	failed := false
	for _, r := range results {
		switch {
		case r.OK:
			fmt.Fprintf(w, " %s: [OK]\n", r.Name)
		case r.Required:
			fmt.Fprintf(w, " %s: %s\n", r.Name, r.Message)
		default:
			fmt.Fprintf(w, " %s: optional: %s\n", r.Name, r.Message)
		}
		failed = failed || !r.OK
	}
	if failed {
		fmt.Fprintln(w, "To install tools manually edit toolchain.env or local.env, or add tools to PATH, or use `anodik install (help)` command.")
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func testTool(tool *Tool, s *env.Settings) error {
	command := tool.CommandName(s)

	if !commandExists(command) {
		return fmt.Errorf("[FATAL] not found %s: specify in toolchain.env or local.env or add to PATH", command)
	}

	version := tool.GetVersion(command)

	if !version.GTE(tool.MinVersion) {
		return fmt.Errorf("[FATAL] found version %s, requireed %s+,", version, tool.MinVersion.String())
	}

	return tool.Check(command)
}
