package jobs

import (
	"anodik/internal/env"
	"anodik/internal/packetmanager"
	"anodik/internal/process"
)

type DoctorJob struct {
	NoSDKRootRequirement
}

func (j *DoctorJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *DoctorJob) Run(s *env.Settings, _ *process.Manager) error {
	_, err := packetmanager.Doctor(s)
	return err
}

func (j *DoctorJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *DoctorJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "check the host toolchain (cmake, ninja, gcc, openocd, ...)",
		Usage:   "",
	}
}
