package cli

import (
	"anodik/internal/discover"
	"anodik/internal/env"
	"anodik/internal/jobs"
	"anodik/internal/toolchain"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

// parseFunc parses a command's arguments into a Job. sdkRoot/workingDir are
// threaded through because example/project resolution happens as part of
// parsing (see resolvePositionalExample and ExampleOptions.Resolve).
type parseFunc func(args []string, sdkRoot, workingDir string) (jobs.Job, error)

// command pairs a command's arg parser with a zero-value constructor used
// only to read its Describe() for help output, so building the help text
// never has to run real argument/example resolution (which can fail, e.g.
// ErrExampleRequired, for perfectly valid commands with no args yet).
type command struct {
	parse parseFunc
	job   func() jobs.Describable
}

// commandTable is the single source of truth for dispatch (ParseArgs) and
// for "anodik help" (PrintCommandList): every entry here is both runnable
// and listed, so the two can never drift apart. Adding a command means
// adding one entry here — nothing else needs to change to pick it up.
var commandTable = map[string]command{
	"build":       {wrapNoCtx(parseBuild), func() jobs.Describable { return &jobs.BuildJob{} }},
	"configure":   {wrapNoCtx(parseConfigure), func() jobs.Describable { return &jobs.ConfigureJob{} }},
	"reconfigure": {wrapNoCtx(parseReconfigure), func() jobs.Describable { return &jobs.ReconfigureJob{} }},
	"clean":       {wrapNoCtx(parseClean), func() jobs.Describable { return &jobs.CleanJob{} }},
	"flash":       {wrapNoCtx(parseFlash), func() jobs.Describable { return &jobs.FlashJob{} }},
	"run":         {wrapNoCtx(parseRun), func() jobs.Describable { return &jobs.RunJob{} }},
	"ocd":         {wrapNoCtx(parseOCD), func() jobs.Describable { return &jobs.OCDJob{} }},
	"gdb":         {wrapNoCtx(parseGDB), func() jobs.Describable { return &jobs.GDBJob{} }},
	"pll":         {wrapNoCtx(parsePll), func() jobs.Describable { return &jobs.PllJob{} }},
	"size":        {wrapNoCtx(parseSize), func() jobs.Describable { return &jobs.SizeJob{} }},
	"doctor":      {ignoreCtx(parseDoctor), func() jobs.Describable { return &jobs.DoctorJob{} }},
	"new":         {ignoreCtx(parseNew), func() jobs.Describable { return &jobs.NewJob{} }},
	"list":        {ignoreCtx(parseList), func() jobs.Describable { return &jobs.ListJob{} }},
	"info":        {wrapNoCtx(parseInfo), func() jobs.Describable { return &jobs.InfoJob{} }},
	"register":    {ignoreCtx(parseRegister), func() jobs.Describable { return &jobs.RegisterJob{} }},
	"srv_erase":   {wrapNoCtx(parseSrvErase), func() jobs.Describable { return &jobs.SrvEraseJob{} }},
	"install":     {ignoreCtx(parseInstall), func() jobs.Describable { return &jobs.InstallJob{} }},
	"uninstall":   {ignoreCtx(parseUninstall), func() jobs.Describable { return &jobs.UninstallJob{} }},
}

// wrapNoCtx is an identity adapter kept so every commandTable entry has the
// same parseFunc shape regardless of whether the underlying parse<X>
// happens to need sdkRoot/workingDir.
func wrapNoCtx(f parseFunc) parseFunc { return f }

// ignoreCtx adapts a parse<X> that doesn't need sdkRoot/workingDir to the
// common parseFunc shape used by commandTable.
func ignoreCtx(f func([]string) (jobs.Job, error)) parseFunc {
	return func(args []string, _, _ string) (jobs.Job, error) {
		return f(args)
	}
}

// ParseArgs parses argv into a Job. sdkRoot and workingDir must already be
// resolved (see platform.Resolve) since example/project directory
// resolution happens as part of parsing, not later during Prepare/Run.
func ParseArgs(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("command required (run: anodik help)")
	}

	if jobs.IsHelpToken(args[0]) {
		printGlobalHelp()
		return nil, flag.ErrHelp
	}

	if cmd, ok := commandTable[args[0]]; ok {
		if hasHelpToken(args[1:]) {
			printCommandHelp(args[0], cmd.job())
			return nil, flag.ErrHelp
		}
		return cmd.parse(args[1:], sdkRoot, workingDir)
	}

	// Unknown command shortcut: if the word matches an SDK example, treat
	// it as "anodik build <word>" so "anodik rndis" just builds rndis.
	if discover.ExampleExists(sdkRoot, args[0]) {
		return parseBuild(args, sdkRoot, workingDir)
	}

	return nil, fmt.Errorf("unknown command %q (run: anodik help)", args[0])
}

// printGlobalHelp prints the full command list, built from commandTable so
// it always matches what's actually dispatchable. Each command's Describe()
// is read off a zero-value job, so this never runs real flag/example
// resolution and can't fail even for commands that require an example.
func printGlobalHelp() {
	entries := make([]jobs.CommandHelp, 0, len(commandTable))
	for name, cmd := range commandTable {
		entries = append(entries, jobs.CommandHelp{Name: name, Info: cmd.job().Describe()})
	}
	jobs.PrintCommandList(os.Stdout, entries)
}

// printCommandHelp prints a single command's detailed help.
func printCommandHelp(name string, job jobs.Describable) {
	job.Describe().Print(os.Stdout, name)
}

// hasHelpToken reports whether "help" appears anywhere among args. Checked
// before the command's own flag parsing so "anodik build help",
// "anodik flash --soc x help", etc. all short-circuit to help output
// without resolving examples or validating other flags — per the CLI
// contract, a positional "help" makes everything after it (and around it)
// irrelevant.
func hasHelpToken(args []string) bool {
	for _, a := range args {
		if jobs.IsHelpToken(a) {
			return true
		}
	}
	return false
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// parseExampleOptions registers --example/--all and, via
// resolvePositionalExample, allows the example name to be given as a bare
// positional argument instead (e.g. "anodik build rndis").
func parseExampleOptions(fs *flag.FlagSet, options *jobs.ExampleOptions) {
	fs.StringVar(&options.Example, "example", "", "")
	fs.BoolVar(&options.All, "all", false, "")
}

// resolvePositionalExample fills options.Example from the first leftover
// positional argument after fs.Parse, if --example wasn't given explicitly.
// flag.FlagSet stops parsing at the first non-flag token, so a positional
// example name must come before any flags: "anodik build rndis --release"
// works, "anodik build --release rndis" does not (use --example instead).
// Returns an error if more than one positional argument is left over, since
// that's more likely a typo than an intentional second example name.
func resolvePositionalExample(fs *flag.FlagSet, options *jobs.ExampleOptions) error {
	remaining := fs.Args()
	if len(remaining) == 0 {
		return nil
	}
	if len(remaining) > 1 {
		return fmt.Errorf("unexpected extra arguments: %v", remaining[1:])
	}

	if options.Example == "" {
		options.Example = remaining[0]
	}
	return nil
}

func parseSoCOptions(fs *flag.FlagSet, options *jobs.SoCOptions) {
	fs.StringVar(&options.SoC, "soc", "", "")
	fs.StringVar(&options.Interface, "interface", "", "")
}

func parseBuildOptions(fs *flag.FlagSet, options *jobs.BuildOptions) {
	fs.Var(buildTypeValue{value: &options.BuildType}, "build-type", "")
	fs.BoolFunc("release", "", func(string) error {
		options.BuildType = jobs.RELEASE
		return nil
	})
	fs.IntVar(&options.Jobs, "jobs", 0, "")
	fs.IntVar(&options.Jobs, "j", 0, "")
}

func parseDebugOptions(fs *flag.FlagSet, options *jobs.DebugOptions) {
	fs.StringVar(&options.GDBPort, "gdb-port", "", "")
	fs.StringVar(&options.TelnetPort, "telnet-port", "", "")
}

func parseProcessOptions(fs *flag.FlagSet, options *jobs.ProcessOptions) {
	fs.BoolVar(&options.RunInBackground, "background", false, "")
	fs.BoolVar(&options.RunInBackground, "bg", false, "")
	fs.BoolVar(&options.StopProcess, "stop", false, "")
	fs.BoolVar(&options.Status, "status", false, "")
}

func parsePllOptions(fs *flag.FlagSet, options *jobs.PllOptions) {
	fs.Float64Var(&options.Input, "input", 0, "")
	fs.Float64Var(&options.Output, "output", 0, "")
	fs.Func("output2", "", func(value string) error {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid --output2 %q: %w", value, err)
		}
		options.SetOutput2(f)
		return nil
	})
	fs.IntVar(&options.NumSolutions, "num-solutions", 1, "")
	fs.IntVar(&options.NumSolutions, "n", 1, "")
}

type buildTypeValue struct {
	value *jobs.BuildType
}

func (v buildTypeValue) String() string {
	if v.value == nil {
		return ""
	}
	return string(*v.value)
}

func (v buildTypeValue) Set(value string) error {
	switch jobs.BuildType(value) {
	case jobs.DEBUG, jobs.RELEASE:
		*v.value = jobs.BuildType(value)
		return nil
	default:
		return fmt.Errorf("invalid build type %q", value)
	}
}

// resolveExample runs ExampleOptions.Resolve after flag parsing so the
// project directory is known before Prepare/Run — required for commands
// that need a project (i.e. anything requiring the SDK root).
func resolveExample(fs *flag.FlagSet, options *jobs.ExampleOptions, sdkRoot, workingDir string) error {
	if err := resolvePositionalExample(fs, options); err != nil {
		return err
	}
	if options.All {
		return nil
	}
	return options.Resolve(sdkRoot, workingDir)
}

func parseBuild(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.BuildJob{}
	fs := newFlagSet("build")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseBuildOptions(fs, &job.BuildOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolveExample(fs, &job.ExampleOptions, sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseConfigure(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.ConfigureJob{}
	fs := newFlagSet("configure")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseBuildOptions(fs, &job.BuildOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseReconfigure(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.ReconfigureJob{}
	fs := newFlagSet("reconfigure")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseBuildOptions(fs, &job.BuildOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseClean(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.CleanJob{}
	fs := newFlagSet("clean")

	parseExampleOptions(fs, &job.ExampleOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolveExample(fs, &job.ExampleOptions, sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseFlash(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.FlashJob{}
	fs := newFlagSet("flash")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parseDebugOptions(fs, &job.DebugOptions)
	parseProcessOptions(fs, &job.ProcessOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseRun(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.RunJob{}
	fs := newFlagSet("run")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parseBuildOptions(fs, &job.BuildOptions)
	parseDebugOptions(fs, &job.DebugOptions)
	parseProcessOptions(fs, &job.ProcessOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseOCD(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.OCDJob{}
	fs := newFlagSet("ocd")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parseDebugOptions(fs, &job.DebugOptions)
	parseProcessOptions(fs, &job.ProcessOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.ResolveOptional(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseGDB(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.GDBJob{}
	fs := newFlagSet("gdb")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parseDebugOptions(fs, &job.DebugOptions)
	parseProcessOptions(fs, &job.ProcessOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parsePll(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.PllJob{}
	fs := newFlagSet("pll")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parsePllOptions(fs, &job.PllOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}

	return job, nil
}

func parseSize(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.SizeJob{}
	fs := newFlagSet("size")

	parseExampleOptions(fs, &job.ExampleOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}

	// size with no --example scans the working directory for every built
	// example, so unlike other commands it must not fail when nothing is
	// auto-detectable; only resolve when an example name was given.
	if job.Example != "" {
		if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
			return nil, err
		}
	}

	return job, nil
}

func parseDoctor(args []string) (jobs.Job, error) {
	job := &jobs.DoctorJob{}
	fs := newFlagSet("doctor")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return job, nil
}

func parseNew(args []string) (jobs.Job, error) {
	job := &jobs.NewJob{}
	fs := newFlagSet("new")

	fs.StringVar(&job.Path, "path", "", "")
	fs.StringVar(&job.Name, "name", "", "")
	fs.BoolVar(&job.Zed, "zed", false, "")
	fs.BoolVar(&job.VScode, "vscode", false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return job, nil
}

func parseList(args []string) (jobs.Job, error) {
	job := &jobs.ListJob{}
	fs := newFlagSet("list")

	fs.BoolVar(&job.Interfaces, "interfaces", false, "")
	fs.BoolVar(&job.SoCs, "socs", false, "")
	fs.BoolVar(&job.Boards, "boards", false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return job, nil
}

func parseInstall(args []string) (jobs.Job, error) {
	job := &jobs.InstallJob{}
	fs := newFlagSet("install")

	var path string
	fs.StringVar(&path, "path", "", "")
	fs.BoolVar(&job.List, "list", false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	remaining := fs.Args()
	if len(remaining) > 1 {
		return nil, fmt.Errorf("unexpected extra arguments: %v", remaining[1:])
	}
	if len(remaining) == 1 {
		job.PacketName = remaining[0]
	}

	set := 0
	if path != "" {
		set++
	}
	if job.List {
		set++
	}
	if job.PacketName != "" {
		set++
	}
	if set > 1 {
		return nil, fmt.Errorf("--path, --list, and a packet name are mutually exclusive")
	}

	if path != "" {
		job.SetPath = true
		job.InstallDir = path
	} else {
		defaultDir, err := toolchain.DefaultInstallDirectory()
		if err != nil {
			return nil, fmt.Errorf("resolve default install directory: %w", err)
		}
		job.InstallDir = defaultDir
	}

	return job, nil
}

func parseUninstall(args []string) (jobs.Job, error) {
	job := &jobs.UninstallJob{}
	fs := newFlagSet("uninstall")

	fs.BoolVar(&job.RemoveEverything, "everything", false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	remaining := fs.Args()
	if len(remaining) > 1 {
		return nil, fmt.Errorf("unexpected extra arguments: %v", remaining[1:])
	}
	if len(remaining) == 1 {
		job.PacketName = remaining[0]
	}

	set := 0
	if job.RemoveEverything {
		set++
	}
	if job.PacketName != "" {
		set++
	}
	if set > 1 {
		return nil, fmt.Errorf("--everything, and a packet name are mutually exclusive")
	}

	var defaultDir string = ""
	settings, err := env.ReadToolchainEnv()
	if err != nil || settings.PacketsInstallPath == "" {
		defaultDir, err = toolchain.DefaultInstallDirectory()
		if err != nil {
			return nil, fmt.Errorf("resolve default install directory: %w", err)
		}
	} else {
		defaultDir = settings.PacketsInstallPath
	}

	job.InstallDir = defaultDir

	return job, nil
}

func parseInfo(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.InfoJob{}
	fs := newFlagSet("info")

	parseExampleOptions(fs, &job.ExampleOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.Resolve(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}

func parseRegister(args []string) (jobs.Job, error) {
	job := &jobs.RegisterJob{}
	fs := newFlagSet("register")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return job, nil
}

func parseSrvErase(args []string, sdkRoot, workingDir string) (jobs.Job, error) {
	job := &jobs.SrvEraseJob{}
	fs := newFlagSet("srv_erase")

	parseExampleOptions(fs, &job.ExampleOptions)
	parseSoCOptions(fs, &job.SoCOptions)
	parseProcessOptions(fs, &job.ProcessOptions)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if err := resolvePositionalExample(fs, &job.ExampleOptions); err != nil {
		return nil, err
	}
	if err := job.ExampleOptions.ResolveOptional(sdkRoot, workingDir); err != nil {
		return nil, err
	}

	return job, nil
}
