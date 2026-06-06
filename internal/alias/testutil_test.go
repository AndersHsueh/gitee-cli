package alias

import (
	"os"
	"path/filepath"
)

// writeFileAtomic is a small helper used by TestResolveArgsCorruptFileGraceful.
// It writes data to path via a temp-file in the same directory followed by
// rename — the same atomic-write pattern the package uses for real
// persistence, so the test exercises the same code path as production.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".aliases-test-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
