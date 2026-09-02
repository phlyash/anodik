package cmakebuild

import (
	"anodik/internal/buildtype"
	"anodik/internal/env"
	"anodik/internal/toolchain"
	"path/filepath"
	"strings"
)

// ConfigureArgs builds the `cmake -S ... -B ... -G Ninja ...` argv for
// configuring srcDir into buildDir at the given build type.
func ConfigureArgs(s *env.Settings, srcDir, buildDir string, buildType buildtype.BuildType) []string {
	args := []string{
		toolchain.CMake(s),
		"-S", srcDir,
		"-B", buildDir,
		"-G", "Ninja",
		"-DCMAKE_MAKE_PROGRAM=" + toolchain.Ninja(s),
		"-DCMAKE_BUILD_TYPE=" + capitalize(string(buildType)),
		"-DSDK_ROOT=" + s.SDKRoot,
		"-DCMAKE_TOOLCHAIN_FILE=" + filepath.Join(s.SDKRoot, "tools", "cmake", "riscv_toolchain.cmake"),
	}

	var toolchainOverrideKeys = []string{
		env.EnvKey("GCCToolchainBinDir"),
		env.EnvKey("ClangToolchainBinDir"),
		env.EnvKey("ConfigForceCompiler"),
		env.EnvKey("RiscvToolchainPrefix"),
	}

	for _, key := range toolchainOverrideKeys {
		if value := env.EnvValue(s, key); value != "" {
			args = append(args, "-D"+key+"="+value)
		}
	}

	return args
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
