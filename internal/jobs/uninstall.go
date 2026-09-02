package jobs

import (
	"anodik/internal/env"
	"anodik/internal/packetmanager"
	"anodik/internal/platform"
	"anodik/internal/process"
	"errors"
	"fmt"
	"os"
)

type UninstallJob struct {
	NoSDKRootRequirement

	InstallDir       string
	PacketName       string
	RemoveEverything bool
}

func (j *UninstallJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "cleans installed packages or SDK directory",
		Usage:   "[--everything | <packettype>]",
		Flags: []FlagHelp{
			{Name: "everything", Desc: "clears everything: all packets, cmake hooks, all packets and user-wide configs"},
		},
	}
}

func (j *UninstallJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *UninstallJob) Run(s *env.Settings, _ *process.Manager) error {
	switch {
	case j.RemoveEverything:
		return j.runUninstallEverything(s)
	case j.PacketName != "":
		return j.runUninstallPacket(s)
	default:
		return fmt.Errorf("nothing to do: specify --everything or a packet name (run: anodik uninstall help)")
	}
}

func (j *UninstallJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *UninstallJob) runUninstallEverything(s *env.Settings) error {
	if s.SDKRoot == "" {
		root, _ := platform.ResolveSdkRoot()
		if root != "" {
			s.SDKRoot = root
		}
	}
	if s.SDKRoot == "" {
		fmt.Println("Cannot find SDK root, skipping")
	} else {
		fmt.Printf("SDK root directory: %s\n", s.SDKRoot)
	}

	if j.InstallDir == "" {
		fmt.Println("Cannot find packets installation directory, skipping")
	} else {
		fmt.Printf("Packets installation directory: %s\n", j.InstallDir)
	}

	configDir, _ := env.ConfigDir()
	if configDir == "" {
		fmt.Println("Cannot find config directory, skipping")
	} else {
		fmt.Printf("Config directoty: %s\n", configDir)
	}

	if j.InstallDir == "" && s.SDKRoot == "" && configDir == "" {
		fmt.Println("Nothing to clear")
		return nil
	}

	if packetmanager.Ask(
		"Do you want to remove directories listed above?",
		packetmanager.Yes|packetmanager.No,
	) == packetmanager.No {
		return nil
	}

	var packetErr error = nil
	var sdkErr error = nil
	var configErr error = nil

	if j.InstallDir != "" {
		packetErr = os.RemoveAll(j.InstallDir)
	}

	if s.SDKRoot != "" {
		sdkErr = errors.Join(os.RemoveAll(s.SDKRoot), platform.UnregisterSdk())
	}

	if configDir != "" {
		configErr = os.RemoveAll(configDir)
	}

	return errors.Join(packetErr, sdkErr, configErr)
}

func (j *UninstallJob) runUninstallPacket(*env.Settings) error {
	if !packetmanager.IsValidName(j.PacketName) {
		return fmt.Errorf("Packet name %s is not valid, check packet names using `anodik install --list` command", j.PacketName)
	}
	packet := packetmanager.Packet{Type: j.PacketName}
	return packetmanager.Uninstall(packet, j.InstallDir)
}
