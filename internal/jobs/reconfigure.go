package jobs

import (
	"anodik/internal/cmakebuild"
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/process"
	"path/filepath"
)

type ReconfigureJob struct {
	SDKRootRequirement
	ExampleOptions
	BuildOptions
}

func (j *ReconfigureJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ReconfigureJob) Run(s *env.Settings, manager *process.Manager) error {
	buildType := j.BuildType.Resolved(s.BuildType)
	buildDir := filepath.Join(j.ProjectDir, "build")
	projectDir := discover.ProjectDir(s.SDKRoot, s.WorkingDir, j.Example)

	if err := cmakebuild.Clean(buildDir, projectDir); err != nil {
		return err
	}

	return cmakebuild.Build(s, manager, j.ProjectDir, buildDir, projectDir, buildType, j.Jobs)
}

func (j *ReconfigureJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ReconfigureJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "wipe an example's build directory and build it from scratch",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.BuildOptions.Help())
}
