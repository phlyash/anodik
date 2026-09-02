package jobs

import (
	"anodik/internal/env"
	"anodik/internal/packetmanager"
	"anodik/internal/process"
	"anodik/internal/toolchain"
	"fmt"
)

const Url = packetmanager.RepoUrl

type InstallJob struct {
	NoSDKRootRequirement

	InstallDir string
	SetPath    bool
	List       bool
	PacketName string
}

func (j *InstallJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *InstallJob) Run(s *env.Settings, _ *process.Manager) error {
	switch {
	case j.SetPath:
		return runInstallSetPath(s, j.InstallDir)
	case j.List:
		return runInstallList(s)
	case j.PacketName != "":
		return runInstallPacket(j.PacketName, s, j.InstallDir)
	default:
		return fmt.Errorf("nothing to do: specify --path, --list, or a packet name (run: anodik install help)")
	}
}

func (j *InstallJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *InstallJob) Describe() HelpInfo {
	desc := "root directory for all installed packets"

	toolchain_settings, err := env.ReadToolchainEnv()
	if err == nil && j.InstallDir == "" {
		j.InstallDir = toolchain_settings.PacketsInstallPath
	}

	if j.InstallDir != "" {
		desc = fmt.Sprintf("%s (current: %s)", desc, j.InstallDir)
	} else if defaultDir, err := toolchain.DefaultInstallDirectory(); err == nil {
		desc = fmt.Sprintf("%s (default: %s)", desc, defaultDir)
	}
	return HelpInfo{
		Summary: "install useful tools: toolchain, openocd, debugger and etc",
		Usage:   "[--path <dir> | --list | <packettype>]",
		Flags: []FlagHelp{
			{Name: "path", Arg: "/path/to/packet/storage", Desc: desc},
			{Name: "list", Desc: "query the repository and print available packets for this OS/arch"},
		},
	}
}

func runInstallSetPath(resolved_settings *env.Settings, path string) error {

	toolchain_settings, err := env.ReadToolchainEnv()
	var settings *env.Settings = toolchain_settings
	if err != nil {
		settings = resolved_settings
	}

	settings.PacketsInstallPath = path

	if err := env.Save(settings); err != nil {
		return fmt.Errorf("save install path: %w", err)
	}
	fmt.Printf("install root set to %s\n", path)
	return nil
}

func runInstallList(*env.Settings) error {
	packets, err := packetmanager.QueryPackets(Url)
	if err != nil {
		return fmt.Errorf("query packets: %w", err)
	}

	if len(packets) == 0 {
		fmt.Println("no packets available for this OS/architecture")
		return nil
	}

	fmt.Println("available packets:")
	for _, p := range packets {
		fmt.Printf("  %s\n", p.String())
	}
	return nil
}

func runInstallPacket(name string, s *env.Settings, installDir string) error {
	packets, err := packetmanager.QueryPackets(Url)
	if err != nil {
		return fmt.Errorf("query packets: %w", err)
	}

	packet, ok := findPacket(packets, name)
	if !ok {
		return fmt.Errorf("packet %q not found (run: anodik install --list)", name)
	}

	packetInstallDir := packet.Directory(installDir)

	if err := packetmanager.Install(packet, packetInstallDir); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}

	if err := packet.SaveRequiredPaths(s, packetInstallDir); err != nil {
		return fmt.Errorf("save required paths for %s: %w", name, err)
	}

	fmt.Printf("installed %s\n", packet.String())
	return nil
}

func findPacket(packets []packetmanager.Packet, name string) (packetmanager.Packet, bool) {
	for _, p := range packets {
		if p.Type == name {
			return p, true
		}
	}
	return packetmanager.Packet{}, false
}
