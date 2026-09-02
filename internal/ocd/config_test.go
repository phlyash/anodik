package ocd

import (
	"anodik/internal/env"
	"os"
	"path/filepath"
	"testing"
)

func writeDebugMk(t *testing.T, buildDir string, kv map[string]string) {
	t.Helper()
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := ""
	for k, v := range kv {
		content += k + " := " + v + "\n"
	}
	if err := os.WriteFile(filepath.Join(buildDir, "debug.mk"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePriorityCLIWins(t *testing.T) {
	sdkRoot := t.TempDir()
	workingDir := t.TempDir()
	example := "myexample"

	writeDebugMk(t, filepath.Join(workingDir, "build"), map[string]string{
		"TARGET_SOC": "k1921vg1t",
	})

	s := &env.Settings{WorkingDir: workingDir, SDKRoot: sdkRoot, SoC: "k1921vg015"}

	cfg, err := Resolve(s, example, "k1921vg3t", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SoC != "k1921vg3t" {
		t.Errorf("SoC = %q, want k1921vg3t (CLI should win)", cfg.SoC)
	}
}

func TestResolvePriorityDebugMkOverSettings(t *testing.T) {
	sdkRoot := t.TempDir()
	workingDir := t.TempDir()
	example := "myexample"

	writeDebugMk(t, filepath.Join(workingDir, "build"), map[string]string{
		"TARGET_SOC": "k1921vg1t",
	})

	s := &env.Settings{WorkingDir: workingDir, SDKRoot: sdkRoot, SoC: "k1921vg015"}

	cfg, err := Resolve(s, example, "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SoC != "k1921vg1t" {
		t.Errorf("SoC = %q, want k1921vg1t (debug.mk should win over settings)", cfg.SoC)
	}
}

func TestResolvePrioritySettingsOverDefault(t *testing.T) {
	sdkRoot := t.TempDir()
	workingDir := t.TempDir()

	s := &env.Settings{WorkingDir: workingDir, SDKRoot: sdkRoot, SoC: "k1921vg015"}

	cfg, err := Resolve(s, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SoC != "k1921vg015" {
		t.Errorf("SoC = %q, want k1921vg015 (settings should win over default)", cfg.SoC)
	}
}

func TestResolveDefault(t *testing.T) {
	sdkRoot := t.TempDir()
	workingDir := t.TempDir()

	s := &env.Settings{WorkingDir: workingDir, SDKRoot: sdkRoot}

	cfg, err := Resolve(s, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SoC != DefaultSoC {
		t.Errorf("SoC = %q, want default %q", cfg.SoC, DefaultSoC)
	}
	if cfg.Interface != DefaultInterface {
		t.Errorf("Interface = %q, want default %q", cfg.Interface, DefaultInterface)
	}
	if cfg.GDBPort != DefaultGDBPort {
		t.Errorf("GDBPort = %q, want default %q", cfg.GDBPort, DefaultGDBPort)
	}
	if cfg.HasELFPath {
		t.Error("HasELFPath should be false when no example is given")
	}
}

func TestResolveCfgPathsDerivedFromSoC(t *testing.T) {
	sdkRoot := t.TempDir()
	workingDir := t.TempDir()

	s := &env.Settings{WorkingDir: workingDir, SDKRoot: sdkRoot}

	cfg, err := Resolve(s, "", "k1921vg7t", "stlink", "")
	if err != nil {
		t.Fatal(err)
	}

	wantTarget := filepath.Join(sdkRoot, "tools", "ocd", "target", "k1921vg7t.cfg")
	if cfg.TargetCfg != wantTarget {
		t.Errorf("TargetCfg = %q, want %q", cfg.TargetCfg, wantTarget)
	}

	wantInterface := filepath.Join(sdkRoot, "tools", "ocd", "interface", "stlink.cfg")
	if cfg.InterfaceCfg != wantInterface {
		t.Errorf("InterfaceCfg = %q, want %q", cfg.InterfaceCfg, wantInterface)
	}
}
