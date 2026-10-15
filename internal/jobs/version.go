package jobs

import (
	"anodik/internal/env"
	"anodik/internal/process"
	"fmt"
)

type VersionJob struct {
	NoSDKRootRequirement
	JSONOutput

	Version string
}

func (j *VersionJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *VersionJob) Run(*env.Settings, *process.Manager) error {
	if j.JSON {
		return printJSON(struct {
			Version string `json:"version"`
		}{j.Version})
	}
	fmt.Println(j.Version)
	return nil
}

func (j *VersionJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *VersionJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "print the anodik version",
	}.Merge(j.JSONOutput.Help())
}
