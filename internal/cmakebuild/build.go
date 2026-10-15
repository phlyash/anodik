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

func EnsureConfigured(s *env.Settings, manager *process.Manager, srcDir, buildDir string, buildType buildtype.BuildType) error {
	args := ConfigureArgs(s, srcDir, buildDir, buildType)

	if !NeedsConfigure(buildDir, args) {
		return nil
	}

	if err := manager.Run(args[0], args[1:]); err != nil {
		return fmt.Errorf("configure: %w", err)
	}

	return WriteStamp(buildDir, args)
}

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
	if err := manager.Run(toolchain.CMake(s), buildArgs); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	return LinkCompileCommands(s, buildDir, projectDir)
}

func Clean(buildDir, projectDir string) (removed bool, err error) {
	_, statErr := os.Lstat(buildDir)
	removed = statErr == nil
	if err := os.RemoveAll(buildDir); err != nil {
		return false, fmt.Errorf("remove build dir: %w", err)
	}

	linked := compileCommandsLinkPath(projectDir)
	if err := os.Remove(linked); err != nil && !os.IsNotExist(err) {
		return removed, fmt.Errorf("remove linked compile_commands.json: %w", err)
	}

	return removed, nil
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
