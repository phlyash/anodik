package ocd

import (
	"path/filepath"
	"testing"
)

func TestSessionNameIsPerInterface(t *testing.T) {
	if SessionName("jlink") == SessionName("ftdi") {
		t.Fatal("different interfaces must not share a session name")
	}

	cases := map[string]string{
		"jlink":     "openocd_jlink",
		"JLink":     "openocd_jlink",
		" jlink ":   "openocd_jlink",
		"jlink.cfg": "openocd_jlink",
		"cmsis-dap": "openocd_cmsis-dap",
		"a/b/ftdi":  "openocd_ftdi",
		"we ird!":   "openocd_we_ird_",
		"":          "openocd_unknown",
	}
	for in, want := range cases {
		if got := SessionName(in); got != want {
			t.Errorf("SessionName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPathsArePerInterface(t *testing.T) {
	if want := filepath.Join(RuntimeDir(), "openocd_jlink.log"); LogPath("jlink") != want {
		t.Errorf("LogPath = %q, want %q", LogPath("jlink"), want)
	}
	if want := filepath.Join(RuntimeDir(), "openocd_jlink.session"); SessionPath("jlink") != want {
		t.Errorf("SessionPath = %q, want %q", SessionPath("jlink"), want)
	}
	if LogPath("jlink") == LogPath("ftdi") {
		t.Error("different interfaces must not share a log file")
	}
	if SessionPath("jlink") == SessionPath("ftdi") {
		t.Error("different interfaces must not share a session file")
	}
	if LogPath("jlink") == SessionPath("jlink") {
		t.Error("log and session must not be the same file")
	}
}

func TestRuntimeDirUnderNiiet(t *testing.T) {
	dir := RuntimeDir()
	if filepath.Base(dir) != "ocd" {
		t.Errorf("RuntimeDir = %q, want it to end in ocd", dir)
	}
	if filepath.Base(filepath.Dir(dir)) != "niiet" {
		t.Errorf("RuntimeDir = %q, want it under a niiet directory", dir)
	}
}
