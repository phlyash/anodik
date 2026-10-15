package jobs

import (
	"anodik/internal/env"
	"anodik/internal/packetmanager"
	"anodik/internal/process"
	"os"
)

type DoctorJob struct {
	NoSDKRootRequirement
	JSONOutput
}

func (j *DoctorJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *DoctorJob) Run(s *env.Settings, _ *process.Manager) error {
	results := packetmanager.CheckTools(s)

	if j.JSON {
		return printJSON(struct {
			OK    bool                       `json:"ok"`
			Tools []packetmanager.ToolStatus `json:"tools"`
		}{packetmanager.MissingRequired(results) == 0, results})
	}
	packetmanager.PrintDoctor(os.Stdout, results)
	return nil
}

func (j *DoctorJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *DoctorJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "check the host toolchain (cmake, ninja, gcc, openocd, ...)",
		Usage:   "",
	}.Merge(j.JSONOutput.Help())
}
