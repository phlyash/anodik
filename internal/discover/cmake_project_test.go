package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCMakeLists(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestIsExecutableProjectDirectName(t *testing.T) {
	dir := t.TempDir()
	writeCMakeLists(t, dir, `
project(run_leds LANGUAGES C ASM)
sdk_add_executable(run_leds
    src/main.c
)
`)

	ok, name := IsExecutableProject(dir)
	if !ok || name != "run_leds" {
		t.Fatalf("got ok=%v name=%q, want ok=true name=run_leds", ok, name)
	}
}

func TestIsExecutableProjectViaProjectNameVar(t *testing.T) {
	dir := t.TempDir()
	writeCMakeLists(t, dir, `
project(blinky LANGUAGES C ASM)
add_executable(${PROJECT_NAME} src/main.c)
`)

	ok, name := IsExecutableProject(dir)
	if !ok || name != "blinky" {
		t.Fatalf("got ok=%v name=%q, want ok=true name=blinky", ok, name)
	}
}

func TestIsExecutableProjectQuotedName(t *testing.T) {
	dir := t.TempDir()
	writeCMakeLists(t, dir, `
project("my project" LANGUAGES C)
add_executable("my project" src/main.c)
`)

	ok, name := IsExecutableProject(dir)
	if !ok || name != "my project" {
		t.Fatalf("got ok=%v name=%q, want ok=true name=%q", ok, name, "my project")
	}
}

func TestIsExecutableProjectNoProject(t *testing.T) {
	dir := t.TempDir()
	writeCMakeLists(t, dir, `add_library(foo src/foo.c)`)

	if ok, _ := IsExecutableProject(dir); ok {
		t.Error("expected not an executable project")
	}
}

func TestIsExecutableProjectMissingFile(t *testing.T) {
	if ok, _ := IsExecutableProject(t.TempDir()); ok {
		t.Error("expected false when CMakeLists.txt is absent")
	}
}
