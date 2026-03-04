package resource

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// copyFiles copies configured files and directories from repo root to the worktree.
// Entries that do not exist in the repo root are silently skipped.
// Paths containing ".." that escape the repo root are rejected.
func (s *Service) copyFiles(wtPath string) error {
	for _, rel := range s.params.CopyFiles {
		src := filepath.Join(s.params.RepoRoot, rel)
		if !strings.HasPrefix(filepath.Clean(src)+string(filepath.Separator), filepath.Clean(s.params.RepoRoot)+string(filepath.Separator)) {
			return fmt.Errorf("copy_files entry %q escapes repository root", rel)
		}
		dst := filepath.Join(wtPath, rel)

		info, err := os.Lstat(src)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue // skip symlinks to prevent following links outside the repo
		}

		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("copying directory %s: %w", rel, err)
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copying file %s: %w", rel, err)
			}
		}
	}
	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil // skip symlinks to prevent following links outside the repo
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

// copyFile copies a single file, preserving its permissions.
func copyFile(src, dst string) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
