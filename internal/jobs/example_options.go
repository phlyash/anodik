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

	// ProjectDir is the resolved source directory: the matching SDK
	// example directory if Example names one, otherwise the working
	// directory (which must itself be a valid CMake project).
	ProjectDir string

	// InSDKExamples is true when ProjectDir lives under
	// <sdkRoot>/examples, meaning shared artifacts belong at the SDK
	// root rather than inside the project itself.
	InSDKExamples bool
}

func (o *ExampleOptions) SetExample(example string) {
	o.Example = example
}

func (o *ExampleOptions) SetAll(all bool) {
	o.All = all
}

// Help describes --example / the positional example-name shortcut. Every
// command taking an example picks up this text via HelpInfo.Merge.
func (ExampleOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "example", Arg: "<name>", Desc: "example to target (or pass it positionally, e.g. \"anodik build rndis\")"},
		},
	}
}

// HelpAll describes --all, for the subset of commands (build, clean) that
// support acting on every example at once.
func (ExampleOptions) HelpAll() FlagHelp {
	return FlagHelp{Name: "all", Desc: "apply to every example instead of just one"}
}

// Resolve fills in ProjectDir. If Example is empty, it auto-detects the
// current example by walking up from workingDir looking for a
// CMakeLists.txt with a project() and add_executable() for it — mirroring
// the python tool's cwd-based auto-detection. Returns ErrExampleRequired if
// nothing could be resolved.
func (o *ExampleOptions) Resolve(sdkRoot, workingDir string) error {
	if err := o.ResolveOptional(sdkRoot, workingDir); err != nil {
		return err
	}
	if o.Example == "" {
		return ErrExampleRequired
	}
	return nil
}

// ResolveOptional behaves like Resolve, except that when no example name
// is given and none can be auto-detected from workingDir, it leaves
// Example/ProjectDir empty instead of failing. Used by commands where the
// example is a convenience for picking up its --soc/--interface/--gdb-port
// from debug.mk (ocd, srv_erase) rather than a hard requirement — those
// commands can still be pointed at a target directly via --soc.
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
