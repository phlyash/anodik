package toolchain

import (
	"anodik/internal/env"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const DefaultPrefix = "riscv32-unknown-elf"

func Prefix(s *env.Settings) string {
	if s.RiscvToolchainPrefix != "" {
		return s.RiscvToolchainPrefix
	}
	return DefaultPrefix
}

// binary resolves a tool named "<prefix>-<suffix>" inside binDir if set,
// otherwise returns the bare name to be looked up on PATH. On Windows the
// binDir form gets an .exe suffix; the bare-name form is left to the OS
// (exec.LookPath already checks PATHEXT).
func binary(binDir, prefix, suffix string) string {
	name := prefix + "-" + suffix

	if binDir == "" {
		return name
	}

	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, name)
}

func BinName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func GCC(s *env.Settings) string {
	return binary(s.GCCToolchainBinDir, Prefix(s), "gcc")
}

func Size(s *env.Settings) string {
	return binary(s.GCCToolchainBinDir, Prefix(s), "size")
}

func notEmptyOrElse(notEmpty string, def string) string {
	if notEmpty != "" {
		return notEmpty
	}

	return def
}

func GDB(s *env.Settings) string {
	return notEmptyOrElse(s.GDB, binary(s.GCCToolchainBinDir, Prefix(s), "gdb"))
}

func Clang(s *env.Settings) string {
	return binary(s.GCCToolchainBinDir, "", "clang")
}

func OpenOCD(s *env.Settings) string {
	return notEmptyOrElse(s.OpenOCD, "openocd")
}

func CMake(s *env.Settings) string {
	return notEmptyOrElse(s.CMake, "cmake")
}

func Ninja(s *env.Settings) string {
	return notEmptyOrElse(s.Ninja, "ninja")
}

func Clangd(s *env.Settings) string {

	return notEmptyOrElse(filepath.Join(s.ToolsBinDir, "clangd"), "clangd")
}

func ClangFormat(s *env.Settings) string {
	return notEmptyOrElse(filepath.Join(s.ToolsBinDir, "clang-format"), "clang-format")
}

func ClangTidy(s *env.Settings) string {
	return notEmptyOrElse(filepath.Join(s.ToolsBinDir, "clang-tidy"), "clang-tidy")
}

func Anodik() string {
	executable, err := os.Executable()
	if err != nil {
		executable = BinName("anodik")
	}
	return executable
}

func userInstallDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "niiet"), nil

	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "niiet"), nil

	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, "niiet"), nil
	}
}

func DefaultInstallDirectory() (string, error) {
	return userInstallDir()
}

func InstallDirectory(s *env.Settings) string {
	defaultDir, err := userInstallDir()

	if s.PacketsInstallPath == "" && err != nil {
		panic(fmt.Errorf("cannot lookup default installation path. define %s in toolchain.env or local.env: %w", env.EnvKey("PacketsInstallPath"), err))
	}

	return notEmptyOrElse(s.PacketsInstallPath, defaultDir)
}
