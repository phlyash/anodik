package jobs

import (
	"anodik/internal/env"
	"anodik/internal/process"
	"fmt"
)

type UpdateJob struct {
	NoSDKRootRequirement

	InstallDir string
	PacketName string
	AssumeYes  bool
}

func (j *UpdateJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "reinstall a packet: uninstall it, then install it again",
		Usage:   "[--yes] <packettype>",
		Flags: []FlagHelp{
			{Name: "yes", Desc: "do not ask before deleting the installed packet (also -y)"},
		},
	}
}

func (j *UpdateJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *UpdateJob) Run(s *env.Settings, _ *process.Manager) error {
	if j.PacketName == "" {
		return fmt.Errorf("nothing to do: specify a packet name (run: anodik update help)")
	}

	uninstall := &UninstallJob{InstallDir: j.InstallDir, PacketName: j.PacketName, AssumeYes: j.AssumeYes}
	removed, err := uninstall.uninstallPacket()
	if err != nil {
		return err
	}
	if !removed {
		fmt.Printf("update of %s cancelled\n", j.PacketName)
		return nil
	}

	return runInstallPacket(j.PacketName, s, j.InstallDir)
}

func (j *UpdateJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}
