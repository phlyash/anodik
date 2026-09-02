package jobs

import (
	"anodik/internal/cmakebuild"
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/process"
	"path/filepath"
)

type CleanJob struct {
	SDKRootRequirement
	ExampleOptions
}

func (j *CleanJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *CleanJob) Run(s *env.Settings, _ *process.Manager) error {
	if j.All {
		return cleanAll(s)
	}

	buildDir := filepath.Join(j.ProjectDir, "build")
	projectDir := discover.ProjectDir(s.SDKRoot, s.WorkingDir, j.Example)
	return cmakebuild.Clean(buildDir, projectDir)
}

func (j *CleanJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *CleanJob) Describe() HelpInfo {
	info := HelpInfo{
		Summary: "remove an example's build directory",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help())
	info.Flags = append(info.Flags, j.ExampleOptions.HelpAll())
	return info
}

func cleanAll(s *env.Settings) error {
	for _, name := range discover.DiscoverExamples(s.WorkingDir) {
		srcDir := filepath.Join(s.WorkingDir, name)
		if err := cmakebuild.Clean(filepath.Join(srcDir, "build"), srcDir); err != nil {
			return err
		}
	}

	examplesDir := discover.ExamplesDir(s.SDKRoot)
	for _, name := range discover.DiscoverExamples(examplesDir) {
		srcDir := filepath.Join(examplesDir, name)
		if err := cmakebuild.Clean(filepath.Join(srcDir, "build"), s.SDKRoot); err != nil {
			return err
		}
	}

	return cmakebuild.Clean(filepath.Join(s.SDKRoot, "build"), s.SDKRoot)
}
