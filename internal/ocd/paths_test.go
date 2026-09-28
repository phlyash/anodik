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

func TestLogPathIsPerInterface(t *testing.T) {
	want := filepath.Join("/wd", "build", "openocd_jlink.log")
	if got := LogPath("/wd", "jlink"); got != want {
		t.Errorf("LogPath = %q, want %q", got, want)
	}
	if LogPath("/wd", "jlink") == LogPath("/wd", "ftdi") {
		t.Error("different interfaces must not share a log file")
	}
}
