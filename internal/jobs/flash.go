package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
	"os"
)

type FlashJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	DebugOptions
	ProcessOptions
}

func (j *FlashJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *FlashJob) Run(s *env.Settings, manager *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, j.GDBPort)
	if err != nil {
		return err
	}

	if err := checkOCDConfigs(cfg); err != nil {
		return err
	}
	if err := checkELF(cfg, j.Example); err != nil {
		return err
	}

	ports := ocd.AllocatePorts(cfg.GDBPort)

	fmt.Printf("Flashing %s...\n", cfg.ELFPath)
	ocd.PrintInfo(cfg, &ports)

	tail := []string{"-c", "program " + cfg.ELFPath + " verify reset exit"}
	args := ocd.Command(s, s.SDKRoot, cfg, ports, tail)

	return manager.RunForeground(args[0], args[1:])
}

func (j *FlashJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *FlashJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "flash a built example onto the target via OpenOCD",
		Usage:   "<example> [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help(), j.DebugOptions.Help())
}

func checkOCDConfigs(cfg ocd.Config) error {
	for _, path := range []string{cfg.InterfaceCfg, cfg.TargetCfg} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("OpenOCD config not found: %s", path)
		}
	}
	return nil
}

func checkELF(cfg ocd.Config, example string) error {
	if !cfg.HasELFPath {
		return fmt.Errorf("no ELF resolved for this command")
	}
	if _, err := os.Stat(cfg.ELFPath); err != nil {
		return fmt.Errorf("ELF not found: %s\nBuild first: anodik build %s", cfg.ELFPath, example)
	}
	return nil
}
