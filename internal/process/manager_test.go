package process

import (
	"path/filepath"
	"testing"
)

func TestSessionRoundTripUsesLogicalKey(t *testing.T) {
	m := &Manager{WorkingDir: t.TempDir()}

	if err := m.writeSession("openocd", 12345, "3334"); err != nil {
		t.Fatal(err)
	}

	pid, metadata, ok := m.Session("openocd")
	if !ok || pid != 12345 || metadata != "3334" {
		t.Fatalf("Session() = (%d, %q, %v), want (12345, \"3334\", true)", pid, metadata, ok)
	}

	want := filepath.Join(m.WorkingDir, "build", "openocd.session")
	if got := m.sessionPath("openocd"); got != want {
		t.Errorf("sessionPath() = %q, want %q", got, want)
	}
}

func TestSessionWithoutMetadata(t *testing.T) {
	m := &Manager{WorkingDir: t.TempDir()}

	if err := m.writeSession("openocd", 7, ""); err != nil {
		t.Fatal(err)
	}

	pid, metadata, ok := m.Session("openocd")
	if !ok || pid != 7 || metadata != "" {
		t.Fatalf("Session() = (%d, %q, %v), want (7, \"\", true)", pid, metadata, ok)
	}
}

func TestSessionAbsent(t *testing.T) {
	m := &Manager{WorkingDir: t.TempDir()}

	if _, _, ok := m.Session("openocd"); ok {
		t.Error("Session() reported ok for a missing session file")
	}
	if m.IsAlive("openocd") {
		t.Error("IsAlive() reported alive for a missing session file")
	}
}
