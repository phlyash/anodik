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
	JSONOutput
	Interfaces bool
	SoCs       bool
	Boards     bool
}

func (j *ListJob) Prepare(*env.Settings, *process.Manager) error {
	return nil
}

func (j *ListJob) Run(s *env.Settings, _ *process.Manager) error {
	switch {
	case j.Interfaces:
		return j.listCfgStems(ocd.InterfaceCfgDir(s.SDKRoot))
	case j.SoCs:
		return j.listCfgStems(ocd.TargetCfgDir(s.SDKRoot))
	case j.Boards:
		return j.listBoards(s)
	default:
		return j.listExamples(s)
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
			{Name: "boards", Desc: "list available BSP configs with standard names"},
		},
	}.Merge(j.JSONOutput.Help())
}

func (j *ListJob) listCfgStems(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return err
	}

	names := make([]string, 0, len(matches))
	for _, path := range matches {
		name := filepath.Base(path)
		names = append(names, name[:len(name)-len(filepath.Ext(name))])
	}

	if j.JSON {
		return printJSON(names)
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

func (j *ListJob) listExamples(s *env.Settings) error {
	localExamples := discover.DiscoverExamples(s.WorkingDir)
	sdkExamples := discover.DiscoverExamples(discover.ExamplesDir(s.SDKRoot))

	if j.JSON {
		return printJSON(struct {
			Local []string `json:"local"`
			SDK   []string `json:"sdk"`
		}{nonNil(localExamples), nonNil(sdkExamples)})
	}

	if len(localExamples) > 0 {
		fmt.Println("Local examples:")
		for _, name := range localExamples {
			fmt.Printf("  %s\n", name)
		}
	}

	if len(sdkExamples) > 0 {
		fmt.Println("\nSDK examples:")
		for _, name := range sdkExamples {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}

func (j *ListJob) listBoards(s *env.Settings) error {
	boards, err := discover.DiscoverBoards(s)
	if err != nil {
		return err
	}
	if j.JSON {
		return printJSON(nonNil(boards))
	}
	for _, board := range boards {
		fmt.Println(board)
	}

	return nil
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
