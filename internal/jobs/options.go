package jobs

import "anodik/internal/buildtype"

type SoCOptions struct {
	SoC       string
	Interface string
}

func (o *SoCOptions) SetSoC(soc string) {
	o.SoC = soc
}

func (o *SoCOptions) SetInterface(iface string) {
	o.Interface = iface
}

func (SoCOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "soc", Arg: "<name>", Desc: "target SoC (defaults to the example's debug.mk, then to a built-in default)"},
			{Name: "interface", Arg: "<name>", Desc: "OpenOCD debug interface (jlink, cmsis-dap, ftdi)"},
		},
	}
}

type BuildType = buildtype.BuildType

const (
	DEBUG   = buildtype.DEBUG
	RELEASE = buildtype.RELEASE
)

type BuildOptions struct {
	BuildType BuildType
	Jobs      int
}

func (o *BuildOptions) SetBuildType(buildType BuildType) {
	o.BuildType = buildType
}

func (o *BuildOptions) SetJobs(jobs int) {
	o.Jobs = jobs
}

func (BuildOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "build-type", Arg: "<debug|release>", Desc: "build configuration (default: debug, or BUILD_TYPE from env)"},
			{Name: "release", Desc: "shorthand for --build-type release"},
			{Name: "jobs", Arg: "<n>", Desc: "parallel build jobs (also -j)"},
		},
	}
}

type DebugOptions struct {
	GDBPort    string
	TelnetPort string
}

func (o *DebugOptions) SetGDBPort(port string) {
	o.GDBPort = port
}

func (o *DebugOptions) SetTelnetPort(port string) {
	o.TelnetPort = port
}

func (DebugOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "gdb-port", Arg: "<port>", Desc: "GDB server port (defaults to the example's debug.mk, then to a built-in default)"},
			{Name: "telnet-port", Arg: "<port>", Desc: "OpenOCD telnet port"},
		},
	}
}

type ProcessOptions struct {
	RunInBackground bool
	StopProcess     bool
	Status          bool
}

func (o *ProcessOptions) SetBackground(background bool) {
	o.RunInBackground = background
}

func (o *ProcessOptions) Background() bool {
	return o.RunInBackground
}

func (o *ProcessOptions) SetStop(stop bool) {
	o.StopProcess = stop
}

func (o *ProcessOptions) SetStatus(status bool) {
	o.Status = status
}

func (ProcessOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "background", Desc: "run in the background, detached (also --bg). Stores `.session` file with PID in build/ in working directory"},
			{Name: "stop", Desc: "stop a background instance started earlier. Gets PID from `.session` file from build/ in working directory"},
			{Name: "status", Desc: "report whether a background instance is running"},
		},
	}
}

type PllOptions struct {
	Input        float64
	Output       float64
	Output2      float64
	HasOutput2   bool
	NumSolutions int
}

func (o *PllOptions) SetInput(khz float64) {
	o.Input = khz
}

func (o *PllOptions) SetOutput(khz float64) {
	o.Output = khz
}

func (o *PllOptions) SetOutput2(khz float64) {
	o.Output2 = khz
	o.HasOutput2 = true
}

func (o *PllOptions) SetNumSolutions(n int) {
	o.NumSolutions = n
}

func (PllOptions) Help() HelpInfo {
	return HelpInfo{
		Flags: []FlagHelp{
			{Name: "input", Arg: "<kHz>", Desc: "reference (input) frequency in kHz"},
			{Name: "output", Arg: "<kHz>", Desc: "desired output frequency in kHz"},
			{Name: "output2", Arg: "<kHz>", Desc: "desired second output frequency in kHz (K1921VG015 only)"},
			{Name: "n", Arg: "<count>", Desc: "number of candidate solutions to print (default: 1)"},
		},
	}
}
