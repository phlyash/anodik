package packetmanager

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type zipEntry struct {
	name    string
	content string
	isDir   bool
}

func writeZip(t *testing.T, path string, entries []zipEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		name := e.name
		if e.isDir && name[len(name)-1] != '/' {
			name += "/"
		}
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create entry %q: %v", e.name, err)
		}
		if !e.isDir {
			if _, err := w.Write([]byte(e.content)); err != nil {
				t.Fatalf("zip write entry %q: %v", e.name, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
}

type tarEntry struct {
	name     string
	content  string
	typeflag byte
	linkname string
	mode     int64
}

func writeTarGz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     mode,
			Linkname: e.linkname,
		}
		if e.typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.content))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header %q: %v", e.name, err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("write tar content %q: %v", e.name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	return string(b)
}

func TestExtractZip_Basic(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.zip")
	destDir := filepath.Join(dir, "out")

	writeZip(t, archivePath, []zipEntry{
		{name: "hello.txt", content: "hello world"},
		{name: "sub/", isDir: true},
		{name: "sub/nested.txt", content: "nested content"},
	})

	if err := Extract(archivePath, destDir, "zip"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := readFile(t, filepath.Join(destDir, "hello.txt")); got != "hello world" {
		t.Errorf("hello.txt = %q, want %q", got, "hello world")
	}
	if got := readFile(t, filepath.Join(destDir, "sub", "nested.txt")); got != "nested content" {
		t.Errorf("sub/nested.txt = %q, want %q", got, "nested content")
	}

	info, err := os.Stat(filepath.Join(destDir, "sub"))
	if err != nil || !info.IsDir() {
		t.Errorf("expected sub to be a directory")
	}
}

func TestExtractZip_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil.zip")
	destDir := filepath.Join(dir, "out")

	writeZip(t, archivePath, []zipEntry{
		{name: "../../evil.txt", content: "pwned"},
	})

	err := Extract(archivePath, destDir, "zip")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}

	// Ensure nothing was written outside destDir.
	if _, statErr := os.Stat(filepath.Join(dir, "evil.txt")); statErr == nil {
		t.Error("path traversal succeeded: evil.txt was written outside destDir")
	}
}

func TestExtractZip_AbsolutePathIsContained(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows absolute-path (drive letter) traversal covered by TestSafeJoin_WindowsAbsolute")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "abs.zip")
	destDir := filepath.Join(dir, "out")

	writeZip(t, archivePath, []zipEntry{
		{name: "/etc/evil.txt", content: "contained"},
	})

	if err := Extract(archivePath, destDir, "zip"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Must land inside destDir, never at the real /etc/evil.txt.
	if _, err := os.Stat("/etc/evil.txt"); err == nil {
		t.Fatal("SECURITY: entry escaped destDir and wrote to real /etc/evil.txt")
	}
	got := readFile(t, filepath.Join(destDir, "etc", "evil.txt"))
	if got != "contained" {
		t.Errorf("content = %q, want %q", got, "contained")
	}
}

func TestExtractZip_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.rar")
	destDir := filepath.Join(dir, "out")

	if err := os.WriteFile(archivePath, []byte("not really an archive"), 0o644); err != nil {
		t.Fatalf("write dummy file: %v", err)
	}

	if err := Extract(archivePath, destDir, "rar"); err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestExtractZip_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "empty.zip")
	destDir := filepath.Join(dir, "out")

	writeZip(t, archivePath, nil)

	if err := Extract(archivePath, destDir, "zip"); err != nil {
		t.Fatalf("Extract failed on empty zip: %v", err)
	}
}

func TestExtractZip_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.zip")
	destDir := filepath.Join(dir, "out")

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("mkdir destDir: %v", err)
	}
	preexisting := filepath.Join(destDir, "hello.txt")
	if err := os.WriteFile(preexisting, []byte("old content"), 0o644); err != nil {
		t.Fatalf("write preexisting file: %v", err)
	}

	writeZip(t, archivePath, []zipEntry{
		{name: "hello.txt", content: "new content"},
	})

	if err := Extract(archivePath, destDir, "zip"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := readFile(t, preexisting); got != "new content" {
		t.Errorf("hello.txt = %q, want %q", got, "new content")
	}
}

func TestExtractTarGz_Basic(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "dir/", typeflag: tar.TypeDir},
		{name: "dir/file.txt", typeflag: tar.TypeReg, content: "tar content"},
	})

	if err := Extract(archivePath, destDir, "tar.gz"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := readFile(t, filepath.Join(destDir, "dir", "file.txt")); got != "tar content" {
		t.Errorf("dir/file.txt = %q, want %q", got, "tar content")
	}
}

func TestExtractTarGz_TgzExtension(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tgz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "file.txt", typeflag: tar.TypeReg, content: "tgz content"},
	})

	if err := Extract(archivePath, destDir, "tar.gz"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got := readFile(t, filepath.Join(destDir, "file.txt")); got != "tgz content" {
		t.Errorf("file.txt = %q, want %q", got, "tgz content")
	}
}

func TestExtractTarGz_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil.tar.gz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "../../evil.txt", typeflag: tar.TypeReg, content: "pwned"},
	})

	err := Extract(archivePath, destDir, "tar.gz")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "evil.txt")); statErr == nil {
		t.Error("path traversal succeeded: evil.txt was written outside destDir")
	}
}

func TestExtractTarGz_SymlinkTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "evil_link.tar.gz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "link", typeflag: tar.TypeSymlink, linkname: "../../outside"},
	})

	err := Extract(archivePath, destDir, "tar.gz")
	if err == nil {
		t.Fatal("expected error for symlink target escaping destDir, got nil")
	}
}

func TestExtractTarGz_ValidSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly restricted on windows; fallback behavior covered separately")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "real.txt", typeflag: tar.TypeReg, content: "real content"},
		{name: "link.txt", typeflag: tar.TypeSymlink, linkname: "real.txt"},
	})

	if err := Extract(archivePath, destDir, "tar.gz"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	linkPath := filepath.Join(destDir, "link.txt")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link.txt: %v", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// Real symlink was created; verify it resolves correctly.
		got := readFile(t, linkPath)
		if got != "real content" {
			t.Errorf("link.txt resolved content = %q, want %q", got, "real content")
		}
	} else {
		// Fallback path (copy) was used; content should still match.
		got := readFile(t, linkPath)
		if got != "real content" {
			t.Errorf("link.txt copied content = %q, want %q", got, "real content")
		}
	}
}

func TestExtractTarGz_HardLink(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	destDir := filepath.Join(dir, "out")

	writeTarGz(t, archivePath, []tarEntry{
		{name: "real.txt", typeflag: tar.TypeReg, content: "hard link content"},
		{name: "hardlink.txt", typeflag: tar.TypeLink, linkname: "real.txt"},
	})

	if err := Extract(archivePath, destDir, "tar.gz"); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	got := readFile(t, filepath.Join(destDir, "hardlink.txt"))
	if got != "hard link content" {
		t.Errorf("hardlink.txt = %q, want %q", got, "hard link content")
	}
}

func TestExtractTarGz_CorruptGzip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "corrupt.tar.gz")
	destDir := filepath.Join(dir, "out")

	if err := os.WriteFile(archivePath, []byte("not gzip data at all"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	if err := Extract(archivePath, destDir, "tar.gz"); err == nil {
		t.Fatal("expected error for corrupt gzip data, got nil")
	}
}

func TestExtractTarGz_MissingFile(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "does-not-exist.tar.gz")
	destDir := filepath.Join(dir, "out")

	if err := Extract(archivePath, destDir, "tar.gz"); err == nil {
		t.Fatal("expected error for missing archive file, got nil")
	}
}

func TestSafeJoin(t *testing.T) {
	base := filepath.Clean(string(os.PathSeparator) + filepath.Join("tmp", "dest"))

	cases := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"simple file", "file.txt", false},
		{"nested file", "a/b/c.txt", false},
		{"dot slash prefix", "./file.txt", false},
		{"parent traversal", "../escape.txt", true},
		{"deep parent traversal", "a/../../escape.txt", true},
		{"multi-level up", "../../../etc/passwd", true},
		{"just dotdot", "..", true},
		{"exact base", ".", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := safeJoin(base, tc.entry)
			if tc.wantErr && err == nil {
				t.Errorf("safeJoin(%q, %q): expected error, got nil", base, tc.entry)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("safeJoin(%q, %q): unexpected error: %v", base, tc.entry, err)
			}
		})
	}
}

func TestSafeJoin_WindowsAbsolute(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific absolute path test")
	}
	base := `C:\tmp\dest`
	_, err := safeJoin(base, `C:\Windows\evil.txt`)
	if err == nil {
		t.Error("expected error for absolute windows path escaping base")
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c", "file.txt")

	if err := writeFile(target, bytes.NewBufferString("data"), 0o644); err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}
	if got := readFile(t, target); got != "data" {
		t.Errorf("content = %q, want %q", got, "data")
	}
}

func TestMakeSymlink_FallbackWhenTargetMissing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nonexistent.txt")
	linkPath := filepath.Join(dir, "link.txt")

	if err := makeSymlink(target, linkPath); err != nil {
		t.Fatalf("makeSymlink failed: %v", err)
	}

	// Should have created *something* at linkPath (symlink or empty placeholder).
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("expected linkPath to exist: %v", err)
	}
}
