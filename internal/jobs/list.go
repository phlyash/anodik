package jobs

import (
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/ocd"
	"anodik/internal/process"
	"fmt"
	"path/filepath"
)

type ListJob struct {
	SDKRootRequirement
	Interfaces bool
	SoCs       bool
}

func (j *ListJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ListJob) Run(s *env.Settings, _ *process.Manager) error {
	switch {
	case j.Interfaces:
		return listCfgStems(ocd.InterfaceCfgDir(s.SDKRoot))
	case j.SoCs:
		return listCfgStems(ocd.TargetCfgDir(s.SDKRoot))
	default:
		return listExamples(s)
	}
}

func (j *ListJob) Finish(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ListJob) Describe() HelpInfo {
	return HelpInfo{
		Summary: "list available examples, debug interfaces, or SoC configs",
		Flags: []FlagHelp{
			{Name: "interfaces", Desc: "list OpenOCD interface configs instead of examples"},
			{Name: "socs", Desc: "list OpenOCD target/SoC configs instead of examples"},
		},
	}
}

func listCfgStems(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return err
	}

	for _, path := range matches {
		name := filepath.Base(path)
		fmt.Println(name[:len(name)-len(filepath.Ext(name))])
	}

	return nil
}

func listExamples(s *env.Settings) error {
	localExamples := discover.DiscoverExamples(s.WorkingDir)
	if len(localExamples) > 0 {
		fmt.Println("Local examples:")
		for _, name := range localExamples {
			fmt.Printf("  %s\n", name)
		}
	}

	sdkExamples := discover.DiscoverExamples(discover.ExamplesDir(s.SDKRoot))
	if len(sdkExamples) > 0 {
		fmt.Println("\nSDK examples:")
		for _, name := range sdkExamples {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}
