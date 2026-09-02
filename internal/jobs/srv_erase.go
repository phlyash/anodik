package jobs

import (
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
)

type SrvEraseJob struct {
	SDKRootRequirement
	ExampleOptions
	SoCOptions
	ProcessOptions
}

func (j *SrvEraseJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Run(s *env.Settings, manager *process.Manager) error {
	cfg, err := ocd.Resolve(s, j.Example, j.SoC, j.Interface, "")
	if err != nil {
		return err
	}
	if err := checkOCDConfigs(cfg); err != nil {
		return err
	}

	ports := ocd.AllocatePorts(cfg.GDBPort)

	fmt.Println("Erasing flash...")
	ocd.PrintInfo(cfg, &ports)

	tail := []string{"-c", "init", "-c", "reset halt", "-c", "flash erase_sector 0 0 last", "-c", "reset", "-c", "exit"}
	args := ocd.Command(s, s.SDKRoot, cfg, ports, tail)

	return manager.RunForeground(args[0], args[1:])
}

func (j *SrvEraseJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Stop(*env.Settings, *process.Manager) error {
	return nil
}

func (j *SrvEraseJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "mass-erase flash via OpenOCD",
		Usage:   "[example] [flags]",
	}.Merge(j.ExampleOptions.Help(), j.SoCOptions.Help())
}
