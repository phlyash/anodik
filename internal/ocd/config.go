package ocd

import (
	"anodik/internal/discover"
	"anodik/internal/env"
	"path/filepath"
)

const (
	DefaultSoC       = "k1921vg7t"
	DefaultInterface = "jlink"
	DefaultGDBPort   = "3333"
)

// Config is the fully resolved set of values needed to talk to a board:
// which SoC/interface config files to feed OpenOCD, and which ports to use.
// Priority for every field is CLI flag > debug.mk (the example's last
// build) > local.env/user env > built-in default.
type Config struct {
	SoC             string
	Interface       string
	GDBPort         string
	TargetCfg       string
	InterfaceCfg    string
	ExtraArgs       string
	ELFPath         string
	HasELFPath      bool
}

func ocdDir(sdkRoot string) string {
	return filepath.Join(sdkRoot, "tools", "ocd")
}

func HelpersDir(sdkRoot string) string {
	return filepath.Join(ocdDir(sdkRoot), "helpers")
}

func TargetCfgDir(sdkRoot string) string {
	return filepath.Join(ocdDir(sdkRoot), "target")
}

func InterfaceCfgDir(sdkRoot string) string {
	return filepath.Join(ocdDir(sdkRoot), "interface")
}

// Resolve computes a Config for the given example (may be "" for
// example-independent commands like srv_erase). cliSoC/cliInterface/
// cliGDBPort are the values passed on the command line, if any.
func Resolve(s *env.Settings, exampleName, cliSoC, cliInterface, cliGDBPort string) (Config, error) {
	var mk *env.BuildConfig
	if exampleName != "" {
		buildDir := discover.BuildDir(s.SDKRoot, s.WorkingDir, exampleName)
		cfg, err := env.ReadDebugMK(buildDir)
		if err == nil {
			mk = cfg
		} else if err != env.ErrDebugMkNotFound {
			return Config{}, err
		}
	}

	mkValue := func(get func(*env.BuildConfig) string) string {
		if mk == nil {
			return ""
		}
		return get(mk)
	}

	soc := pick(cliSoC, mkValue(func(c *env.BuildConfig) string { return c.SoC }), s.SoC, DefaultSoC)
	iface := pick(cliInterface, mkValue(func(c *env.BuildConfig) string { return c.Interface }), s.Interface, DefaultInterface)
	gdbPort := pick(cliGDBPort, mkValue(func(c *env.BuildConfig) string { return c.GDBPort }), s.GDBPort, DefaultGDBPort)

	targetCfg := pick("", mkValue(func(c *env.BuildConfig) string { return c.OCDTargetCfg }), s.OCDTargetCfg,
		filepath.Join(TargetCfgDir(s.SDKRoot), soc+".cfg"))
	interfaceCfg := pick("", mkValue(func(c *env.BuildConfig) string { return c.OCDInterfaceCfg }), s.OCDInterfaceCfg,
		filepath.Join(InterfaceCfgDir(s.SDKRoot), iface+".cfg"))
	extraArgs := pick("", mkValue(func(c *env.BuildConfig) string { return c.OCDExtraArgs }), s.OCDExtraArgs, "")

	cfg := Config{
		SoC:          soc,
		Interface:    iface,
		GDBPort:      gdbPort,
		TargetCfg:    targetCfg,
		InterfaceCfg: interfaceCfg,
		ExtraArgs:    extraArgs,
	}

	if exampleName != "" {
		exampleDir := discover.ExampleDir(s.SDKRoot, s.WorkingDir, exampleName)
		cfg.ELFPath = filepath.Join(exampleDir, "build", filepath.Base(exampleName))
		cfg.HasELFPath = true
	}

	return cfg, nil
}

func pick(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
