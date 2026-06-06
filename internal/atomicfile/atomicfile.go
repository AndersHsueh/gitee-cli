// Package atomicfile provides atomic file writes: replace path with
// data via a temp file + rename in the same directory. On POSIX the
// rename is atomic, so concurrent readers always see the old file or
// the complete new file. On Windows, os.Rename replaces an existing
// file, which is also atomic at the filesystem level.
//
// Cleanup: if any step before the final rename fails, the temp file
// is removed so the directory does not accumulate stale ".tmp" files
// from a crashed write.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write writes data to path atomically: it creates path.tmp in the
// same directory, writes data, chmods to perm, closes, and renames
// onto path. The parent directory of path must already exist.
//
// If perm is zero, the file is not chmod'd (useful when the caller
// wants the umask to decide, e.g. for non-secret files).
func Write(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if perm != 0 {
		if err := tmp.Chmod(perm); err != nil {
			_ = tmp.Close()
			cleanup()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("renaming %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}
