package jobs

import (
	"anodik/internal/cmakebuild"
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/process"
	"path/filepath"
)

type BuildJob struct {
	SDKRootRequirement
	ExampleOptions
	BuildOptions
}

func (j *BuildJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *BuildJob) Run(s *env.Settings, manager *process.Manager) error {
	buildType := j.BuildType.Resolved(s.BuildType)

	if j.All {
		return buildAll(s, manager, buildType, j.Jobs)
	}

	return buildOne(s, manager, j.ProjectDir, j.Example, buildType, j.Jobs)
}

func (j *BuildJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *BuildJob) Describe() HelpInfo {
	info := HelpInfo{
		Summary: "configure and build an example",
		Usage:   "<example> [flags].\nIf you're building an example from SDK, you can use shortcut `anodik example_name`.\nTry `anodik bsp_smoke`",
	}.Merge(j.ExampleOptions.Help(), j.BuildOptions.Help())
	info.Flags = append(info.Flags, j.ExampleOptions.HelpAll())
	return info
}

func buildOne(s *env.Settings, manager *process.Manager, srcDir, exampleName string, buildType BuildType, jobCount int) error {
	buildDir := filepath.Join(srcDir, "build")
	projectDir := discover.ProjectDir(s.SDKRoot, s.WorkingDir, exampleName)
	return cmakebuild.Build(s, manager, srcDir, buildDir, projectDir, buildType, jobCount)
}

func buildAll(s *env.Settings, manager *process.Manager, buildType BuildType, jobCount int) error {
	for _, name := range discover.DiscoverExamples(s.WorkingDir) {
		if err := buildOne(s, manager, s.WorkingDir, name, buildType, jobCount); err != nil {
			return err
		}
	}

	for _, name := range discover.DiscoverExamples(discover.ExamplesDir(s.SDKRoot)) {
		srcDir := filepath.Join(discover.ExamplesDir(s.SDKRoot), name)
		if err := buildOne(s, manager, srcDir, name, buildType, jobCount); err != nil {
			return err
		}
	}

	return nil
}
