package jobs

import (
	"anodik/internal/env"
	"anodik/internal/integration"
	"anodik/internal/process"
	"anodik/internal/toolchain"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var sdkTemplateFiles = []string{
	".clang-tidy",
	".clang-format",
	".gitattributes",
	".editorconfig",
	".gdbinit",
	".clangd",
}

const CMAKE_LISTS_TEMPLATE = `cmake_minimum_required(VERSION 3.20)

# Inline SDK configuration — these CONFIG_* variables, set before project(),
# shadow the CACHE defaults tools/cmake/config/ would otherwise declare.
# Edit to match your target.
set(CONFIG_SOC         "k1921vg1t")
set(CONFIG_RISCV_ARCH  "rv32imafc_zicsr_zifencei")
set(CONFIG_RISCV_ABI   "ilp32f")
set(CONFIG_LINKER_MODE "flash")
set(CONFIG_LIBC        "default")

find_package(RiscvSdk REQUIRED)

project(%s LANGUAGES C ASM)

sdk_add_executable(${PROJECT_NAME}
    src/main.c
)
`

const MAIN_C_TEMPLATE = `int main(void)
{
    for(;;){}
}
`

const GITIGNORE_TEMPLATE = `build/
lib/
.zed/
.vscode/
.cache/
__pycache__/
.venv/
.asm-lsp.toml
compile_commands.json
pyrightconfig.json
local.env
.config*
.DS_Store
CMakePresets.json
`

const CMAKE_PRESETS_TEMPLATE = `{
  "version": 6,
  "cmakeMinimumRequired": { "major": 3, "minor": 23, "patch": 0 },
  "include": [
    "%s"
  ]
}
`

type NewJob struct {
	SDKRootRequirement
	Path   string
	Name   string
	Zed    bool
	VScode bool
}

func (j *NewJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *NewJob) Run(s *env.Settings, _ *process.Manager) error {
	targetDir, err := resolveNewProjectDir(s.WorkingDir, j.Path)
	if err != nil {
		return err
	}

	name := j.Name
	if name == "" {
		name = filepath.Base(targetDir)
	}

	if err := os.MkdirAll(filepath.Join(targetDir, "inc"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "src"), 0755); err != nil {
		return err
	}

	files := map[string]string{
		"CMakeLists.txt":    fmt.Sprintf(CMAKE_LISTS_TEMPLATE, name),
		"CMakePresets.json": fmt.Sprintf(CMAKE_PRESETS_TEMPLATE, filepath.Join(s.SDKRoot, "CMakePresets.json")),
		"src/main.c":        MAIN_C_TEMPLATE,
		".gitignore":        GITIGNORE_TEMPLATE,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(targetDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}

	for _, name := range sdkTemplateFiles {
		copySDKTemplate(s.SDKRoot, targetDir, name)
	}

	fmt.Printf("Created project '%s' in %s\n", name, targetDir)
	var zedErr error = nil
	var vscodeErr error = nil

	if j.Zed {
		zedErr = generateZedDirectory(s, targetDir)
	}
	if j.VScode {
		vscodeErr = generateVscodeDirecotry(s, targetDir, name)
	}
	return errors.Join(zedErr, vscodeErr)
}

func (j *NewJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *NewJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "scaffold a new project",
		Flags: []FlagHelp{
			{Name: "path", Arg: "<dir>", Desc: "target directory (default: current directory, must be empty)"},
			{Name: "name", Arg: "<name>", Desc: "project name (default: target directory's name)"},
			{Name: "zed", Desc: "generate tasks.json for zed editor"},
			{Name: "vscode", Desc: "generate tasks.json for vscode"},
		},
	}
}

func resolveNewProjectDir(workingDir, path string) (string, error) {
	if path == "" {
		entries, err := os.ReadDir(workingDir)
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}
		if len(entries) > 0 {
			return "", fmt.Errorf("current directory is not empty; specify --path or use an empty directory")
		}
		return workingDir, nil
	}

	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(workingDir, path), nil
}

// copySDKTemplate copies name from the SDK root into targetDir if it
// exists there. Missing template files are skipped rather than failing the
// whole scaffold — they're editor/tooling niceties, not required for a
// buildable project.
func copySDKTemplate(sdkRoot, targetDir, name string) {
	data, err := os.ReadFile(filepath.Join(sdkRoot, name))
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(targetDir, name), data, 0644)
}

func anodikPath() string {
	return filepath.ToSlash(toolchain.Anodik())
}

func generateZedDirectory(_ *env.Settings, targetDir string) error {
	if err := os.MkdirAll(filepath.Join(targetDir, ".zed"), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(targetDir, ".zed", "tasks.json"), []byte(integration.ZedTasks(anodikPath())), 0644)
}

func generateVscodeDirecotry(s *env.Settings, targetDir string, projectName string) error {
	dir := filepath.Join(targetDir, ".vscode")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	taskErr := os.WriteFile(filepath.Join(dir, "tasks.json"), []byte(integration.VscodeTasks(anodikPath())), 0644)
	var launchErr error = nil
	/// todo: not stable yet
	// launchErr := os.WriteFile(filepath.Join(dir, "launch.json"), []byte(integration.VscodeLaunch(projectName, filepath.ToSlash(toolchain.GDB(s)))), 0644)
	return errors.Join(taskErr, launchErr)
}
