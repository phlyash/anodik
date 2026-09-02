package jobs

import (
	"anodik/internal/cmakebuild"
	"anodik/internal/env"
	"anodik/internal/process"
	"path/filepath"
)

type ConfigureJob struct {
	SDKRootRequirement
	ExampleOptions
	BuildOptions
}

func (j *ConfigureJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ConfigureJob) Run(s *env.Settings, manager *process.Manager) error {
	buildType := j.BuildType.Resolved(s.BuildType)
	buildDir := filepath.Join(j.ProjectDir, "build")
	return cmakebuild.EnsureConfigured(s, manager, j.ProjectDir, buildDir, buildType)
}

func (j *ConfigureJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ConfigureJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "run cmake configure for an example without building it",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.BuildOptions.Help())
}
