package jobs

import (
	"anodik/internal/discover"
	"errors"
)

var ErrExampleRequired = errors.New("example name required (run: anodik list)")

// ExampleOptions carries the --example/--all flags plus the resolved
// project directory. Resolve must be called once, right after flag
// parsing, with the SDK root and working directory determined at startup.
type ExampleOptions struct {
	Example string
	All     bool

	ProjectDir    string
	InSDKExamples bool
}

func (o *ExampleOptions) SetExample(example string) {
	o.Example = example
}

func (o *ExampleOptions) SetAll(all bool) {
	o.All = all
}

func (ExampleOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "example", Arg: "<name>", Desc: "example to target (or pass it positionally, e.g. \"anodik build rndis\")"},
		},
	}
}

func (ExampleOptions) HelpAll() FlagHelp {
	return FlagHelp{Name: "all", Desc: "apply to every example instead of just one"}
}

func (o *ExampleOptions) Resolve(sdkRoot, workingDir string) error {
	if err := o.ResolveOptional(sdkRoot, workingDir); err != nil {
		return err
	}
	if o.Example == "" {
		return ErrExampleRequired
	}
	return nil
}

func (o *ExampleOptions) ResolveOptional(sdkRoot, workingDir string) error {
	name := o.Example
	if name == "" {
		name = discover.DetectCurrentExample(sdkRoot, workingDir)
		if name == "" {
			return nil
		}
		o.Example = name
	}

	o.ProjectDir = discover.ExampleDir(sdkRoot, workingDir, name)
	o.InSDKExamples = discover.ProjectDir(sdkRoot, workingDir, name) == sdkRoot && sdkRoot != ""

	return nil
}
