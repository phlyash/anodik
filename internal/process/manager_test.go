package process

import (
	"path/filepath"
	"testing"
)

func TestSessionRoundTrip(t *testing.T) {
	m := &Manager{}
	path := filepath.Join(t.TempDir(), "nested", "openocd_jlink.session")

	if err := m.writeSession(path, 12345, "3334"); err != nil {
		t.Fatal(err)
	}

	pid, metadata, ok := m.Session(path)
	if !ok || pid != 12345 || metadata != "3334" {
		t.Fatalf("Session() = (%d, %q, %v), want (12345, \"3334\", true)", pid, metadata, ok)
	}
}

func TestSessionWithoutMetadata(t *testing.T) {
	m := &Manager{}
	path := filepath.Join(t.TempDir(), "openocd_jlink.session")

	if err := m.writeSession(path, 7, ""); err != nil {
		t.Fatal(err)
	}

	pid, metadata, ok := m.Session(path)
	if !ok || pid != 7 || metadata != "" {
		t.Fatalf("Session() = (%d, %q, %v), want (7, \"\", true)", pid, metadata, ok)
	}
}

func TestSessionAbsent(t *testing.T) {
	m := &Manager{}
	path := filepath.Join(t.TempDir(), "openocd_jlink.session")

	if _, _, ok := m.Session(path); ok {
		t.Error("Session() reported ok for a missing session file")
	}
	if m.IsAlive(path) {
		t.Error("IsAlive() reported alive for a missing session file")
	}
}
