package jobs

import (
	"anodik/internal/cmakebuild"
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/process"
	"fmt"
	"path/filepath"
)

type SizeJob struct {
	SDKRootRequirement
	ExampleOptions
}

func (j *SizeJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SizeJob) Run(s *env.Settings, _ *process.Manager) error {
	var names []string
	if j.Example != "" {
		names = []string{j.Example}
	} else {
		names = discover.DiscoverExamples(s.WorkingDir)
	}

	found := false
	for _, name := range names {
		srcDir := discover.ExampleDir(s.SDKRoot, s.WorkingDir, name)
		buildDir := filepath.Join(srcDir, "build")

		ok, err := cmakebuild.Size(s, buildDir, filepath.Base(name))
		if err != nil {
			return err
		}
		found = found || ok
	}

	if !found {
		fmt.Println("No ELF files found. Build first: anodik build <example>")
	}

	return nil
}

func (j *SizeJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SizeJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "print a berkeley-format size report for built ELFs",
		Usage:   "[example] [flags]",
	}.Merge(j.ExampleOptions.Help())
}
