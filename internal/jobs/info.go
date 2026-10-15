package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
)

type InfoJob struct {
	SDKRootRequirement
	ExampleOptions
	JSONOutput
}

func (j *InfoJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *InfoJob) Run(s *env.Settings, _ *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, "", "", "")
	if err != nil {
		return err
	}

	if j.JSON {
		return printJSON(ocd.NewInfo(j.Example, cfg, nil))
	}

	fmt.Printf("Configuration for '%s':\n", j.Example)
	ocd.PrintInfo(cfg, nil)
	return nil
}

func (j *InfoJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *InfoJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "print the resolved SoC/interface/port config for an example",
		Usage:   "<example>",
	}.Merge(j.ExampleOptions.Help(), j.JSONOutput.Help())
}
