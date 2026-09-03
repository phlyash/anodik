package packetmanager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Extract(archivePath string, destDir string, archiveType string) error {
	destDir = filepath.Clean(destDir)

	var err error = nil

	switch archiveType {
	case "zip":
		err = extractZip(archivePath, destDir)
	case "tar.gz":
		err = extractTarGz(archivePath, destDir)
	default:
		err = fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	if err != nil {
		return fmt.Errorf("cannot extract: %w", err)
	}
	return nil
}

func safeJoin(base, name string) (string, error) {
	name = filepath.FromSlash(name)
	joined := filepath.Clean(filepath.Join(base, name))
	if !strings.HasPrefix(joined, base+string(os.PathSeparator)) && joined != base {
		return "", fmt.Errorf("path traversal detected: %q escapes destination", name)
	}
	return joined, nil
}

func mkParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writeFile(destPath string, src io.Reader, mode os.FileMode) error {
	if err := mkParent(destPath); err != nil {
		return err
	}
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, src)
	closeErr := f.Close()
	return errors.Join(copyErr, closeErr)
}

func makeSymlink(linkText, linkPath, resolvedTarget string) error {
	if err := mkParent(linkPath); err != nil {
		return err
	}
	if err := os.Symlink(linkText, linkPath); err == nil {
		return nil
	}
	fi, err := os.Lstat(resolvedTarget)
	if err != nil {
		return writeFile(linkPath, strings.NewReader(""), 0o644)
	}
	if !fi.Mode().IsRegular() {
		return nil
	}
	f, err := os.Open(resolvedTarget)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeFile(linkPath, f, fi.Mode())
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractZipEntry(f, destDir); err != nil {
			return fmt.Errorf("zip entry %q: %w", f.Name, err)
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, destDir string) error {
	destPath, err := safeJoin(destDir, f.Name)
	if err != nil {
		return err
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0o755)
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := f.Mode()
	if mode == 0 {
		mode = 0o644
	}
	return writeFile(destPath, rc, mode)
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if err := extractTarEntry(hdr, tr, destDir); err != nil {
			return fmt.Errorf("tar entry %q: %w", hdr.Name, err)
		}
	}
	return nil
}

func extractTarEntry(hdr *tar.Header, r io.Reader, destDir string) error {
	destPath, err := safeJoin(destDir, hdr.Name)
	if err != nil {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destPath, 0o755)

	case tar.TypeSymlink:
		// hdr.Linkname is the literal text that must end up inside the
		// symlink. It is almost always relative to the directory that
		// contains the link itself (destPath's parent), NOT to destDir.
		// We must resolve it against that directory purely to validate
		// that it doesn't escape destDir - but the value written into
		// the symlink must remain the original (relative) Linkname.
		var resolveBase string
		if filepath.IsAbs(hdr.Linkname) {
			resolveBase = destDir
		} else {
			resolveBase = filepath.Dir(destPath)
		}
		resolvedTarget, err := safeJoin(resolveBase, hdr.Linkname)
		if err != nil {
			return fmt.Errorf("symlink target unsafe: %w", err)
		}
		return makeSymlink(hdr.Linkname, destPath, resolvedTarget)

	case tar.TypeLink:
		targetPath, err := safeJoin(destDir, hdr.Linkname)
		if err != nil {
			return fmt.Errorf("hard link target unsafe: %w", err)
		}
		if err := mkParent(destPath); err != nil {
			return err
		}
		if err := os.Link(targetPath, destPath); err != nil {
			src, err := os.Open(targetPath)
			if err != nil {
				return err
			}
			defer src.Close()
			fi, _ := src.Stat()
			mode := os.FileMode(0o644)
			if fi != nil {
				mode = fi.Mode()
			}
			return writeFile(destPath, src, mode)
		}
		return nil

	case tar.TypeReg, tar.TypeRegA:
		mode := hdr.FileInfo().Mode()
		if mode == 0 {
			mode = 0o644
		}
		return writeFile(destPath, r, mode)

	default:
		// Skip devices, fifos, etc.
		return nil
	}
}
