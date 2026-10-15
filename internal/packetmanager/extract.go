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
		err = withExtractor(destDir, func(x *extractor) error { return x.extractZip(archivePath) })
	case "tar.gz":
		err = withExtractor(destDir, func(x *extractor) error { return x.extractTarGz(archivePath) })
	default:
		err = fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	if err != nil {
		return fmt.Errorf("cannot extract: %w", err)
	}
	return nil
}

func extractInto(archivePath, targetDir, archiveType string) error {
	_, statErr := os.Lstat(targetDir)
	existed := statErr == nil

	err := Extract(archivePath, targetDir, archiveType)
	if err != nil && !existed {
		if rmErr := os.RemoveAll(targetDir); rmErr != nil {
			return errors.Join(err, fmt.Errorf("remove partially extracted %s: %w", targetDir, rmErr))
		}
	}
	return err
}

type extractor struct {
	dest     string
	realDest string
}

func withExtractor(destDir string, run func(*extractor) error) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	realDest, err := filepath.EvalSymlinks(destDir)
	if err != nil {
		return err
	}
	return run(&extractor{dest: destDir, realDest: realDest})
}

func within(base, path string) bool {
	rel, err := filepath.Rel(base, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (x *extractor) resolve(dir, target string, depth int) (string, error) {
	if depth > 40 {
		return "", errors.New("too many levels of symbolic links")
	}
	cur := dir
	for _, part := range strings.Split(filepath.ToSlash(target), "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
		default:
			next := filepath.Join(cur, part)
			fi, err := os.Lstat(next)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				cur = next
				break
			}
			link, err := os.Readlink(next)
			if err != nil {
				return "", err
			}
			if isAbsLinkname(link) {
				cur = filepath.Clean(link)
			} else if cur, err = x.resolve(cur, link, depth+1); err != nil {
				return "", err
			}
		}
		if !within(x.realDest, cur) {
			return "", fmt.Errorf("path traversal detected: %q escapes destination", target)
		}
	}
	return cur, nil
}

func isAbsLinkname(link string) bool {
	return filepath.IsAbs(link) || filepath.VolumeName(link) != "" ||
		strings.HasPrefix(link, "/") || strings.HasPrefix(link, `\`)
}

func (x *extractor) prepare(destPath string) (string, error) {
	rel, err := filepath.Rel(x.dest, filepath.Dir(destPath))
	if err != nil {
		return "", err
	}
	realParent, err := x.resolve(x.realDest, rel, 0)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(realParent, 0o755); err != nil {
		return "", err
	}
	if fi, err := os.Lstat(destPath); err == nil && !fi.IsDir() {
		if err := os.Remove(destPath); err != nil {
			return "", err
		}
	}
	return realParent, nil
}

func (x *extractor) symlink(destPath, linkname string) error {
	if isAbsLinkname(linkname) {
		return fmt.Errorf("symlink target unsafe: absolute target %q", linkname)
	}
	realParent, err := x.prepare(destPath)
	if err != nil {
		return err
	}
	resolvedTarget, err := x.resolve(realParent, linkname, 0)
	if err != nil {
		return fmt.Errorf("symlink target unsafe: %w", err)
	}
	return makeSymlink(linkname, destPath, resolvedTarget)
}

func (x *extractor) mkdir(destPath string) error {
	if _, err := x.prepare(destPath); err != nil {
		return err
	}
	return os.MkdirAll(destPath, 0o755)
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

func (x *extractor) extractZip(archivePath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if err := x.extractZipEntry(f); err != nil {
			return fmt.Errorf("zip entry %q: %w", f.Name, err)
		}
	}
	return nil
}

func (x *extractor) extractZipEntry(f *zip.File) error {
	destPath, err := safeJoin(x.dest, f.Name)
	if err != nil {
		return err
	}

	if f.FileInfo().IsDir() {
		return x.mkdir(destPath)
	}
	if f.Mode()&os.ModeSymlink != 0 {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		target, err := io.ReadAll(io.LimitReader(rc, 4096))
		rc.Close()
		if err != nil {
			return err
		}
		return x.symlink(destPath, string(target))
	}
	if _, err := x.prepare(destPath); err != nil {
		return err
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

func (x *extractor) extractTarGz(archivePath string) error {
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
		if err := x.extractTarEntry(hdr, tr); err != nil {
			return fmt.Errorf("tar entry %q: %w", hdr.Name, err)
		}
	}
	return nil
}

func (x *extractor) extractTarEntry(hdr *tar.Header, r io.Reader) error {
	destPath, err := safeJoin(x.dest, hdr.Name)
	if err != nil {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return x.mkdir(destPath)

	case tar.TypeSymlink:
		return x.symlink(destPath, hdr.Linkname)

	case tar.TypeLink:
		if isAbsLinkname(hdr.Linkname) {
			return fmt.Errorf("hard link target unsafe: absolute target %q", hdr.Linkname)
		}
		targetPath, err := x.resolve(x.realDest, hdr.Linkname, 0)
		if err != nil {
			return fmt.Errorf("hard link target unsafe: %w", err)
		}
		if _, err := x.prepare(destPath); err != nil {
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
		if _, err := x.prepare(destPath); err != nil {
			return err
		}
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
