package cmakebuild

import (
	"anodik/internal/buildtype"
	"anodik/internal/env"
	"anodik/internal/process"
	"anodik/internal/toolchain"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

// EnsureConfigured runs `cmake -S srcDir -B buildDir ...` if the stamp from
// a previous configure doesn't match what would be generated now.
func EnsureConfigured(s *env.Settings, manager *process.Manager, srcDir, buildDir string, buildType buildtype.BuildType) error {
	args := ConfigureArgs(s, srcDir, buildDir, buildType)

	if !NeedsConfigure(buildDir, args) {
		return nil
	}

	if err := manager.Run(args[0], args[1:], false); err != nil {
		return fmt.Errorf("configure: %w", err)
	}

	return WriteStamp(buildDir, args)
}

// Build configures (if needed) and builds srcDir/buildDir, then links
// compile_commands.json into projectDir/build for editor tooling.
func Build(s *env.Settings, manager *process.Manager, srcDir, buildDir, projectDir string, buildType buildtype.BuildType, jobCount int) error {
	if err := EnsureConfigured(s, manager, srcDir, buildDir, buildType); err != nil {
		return err
	}

	if jobCount <= 0 {
		jobCount = runtime.NumCPU()
		if jobCount <= 0 {
			jobCount = 4
		}
	}

	buildArgs := []string{"--build", buildDir, "-j" + strconv.Itoa(jobCount)}
	if err := manager.Run(toolchain.CMake(s), buildArgs, false); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	return LinkCompileCommands(s, buildDir, projectDir)
}

// Clean removes the build directory and any linked compile_commands.json
// left in projectDir.
func Clean(buildDir, projectDir string) error {
	if err := os.RemoveAll(buildDir); err != nil {
		return fmt.Errorf("remove build dir: %w", err)
	}

	linked := compileCommandsLinkPath(projectDir)
	if err := os.Remove(linked); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove linked compile_commands.json: %w", err)
	}

	return nil
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
