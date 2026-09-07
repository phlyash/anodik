package packetmanager

import (
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"fmt"
	"io"
	"net/http"
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
	PacketName  PacketType                 // name of packet on server
	CommandName func(*env.Settings) string // name of command in PATH to lookup
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

func QueryPackets(url string) ([]Packet, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
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

	return ParsePacketList(data)
}

func Doctor(s *env.Settings) ([]int, error) {
	var numFailed int = 0
	var listFailed []int = make([]int, len(RequiredToolTable))

	for idx := range RequiredToolTable {
		tool := &RequiredToolTable[idx]
		err := testTool(tool, s)
		if err != nil {
			if tool.Demand == Required {
				fmt.Printf(" %s: %s\n", tool.LogName, err)
			} else {
				fmt.Printf(" %s: optional: %s\n", tool.LogName, err)
			}
			listFailed = append(listFailed, idx)
			numFailed++
		} else {
			fmt.Printf(" %s: [OK]\n", tool.LogName)
		}
	}

	if numFailed > 0 {
		fmt.Printf("To install tools manually edit toolchain.env or local.env, or add tools to PATH, or use `anodik install (help)` command.")
	}

	return listFailed, nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func findTool(dir, name string) bool {
	if dir == "" {
		return commandExists(name)
	}
	_, err := exec.LookPath(dir + "/" + name)
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
