package appdir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// BundleID is the namespace used for OS application and login records.
func BundleID() string {
	if Name == "freemagpie" {
		return "com.freemagpie.app"
	}
	return "com.yetone.magpie"
}

// ImportLegacyConfig explicitly copies a stopped magpie's profile. Existing
// FreeMagpie data is never merged or replaced; the source stays untouched.
// Caches are fetched again under FreeMagpie's own directories.
func ImportLegacyConfig() error {
	if Name != "freemagpie" || Portable() != "" {
		return errors.New("import requires an installed FreeMagpie build")
	}
	dst := Config()
	src := filepath.Join(filepath.Dir(dst), "magpie")
	if _, err := os.Lstat(dst); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("destination must not exist: %s", dst)
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source must be a directory: %s", src)
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dst), ".freemagpie-import-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	// Links might point back into magpie's live data, defeating isolation.
	if err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("import refuses linked or special file: %s", path)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := copyTree(src, tmp); err != nil {
		return err
	}
	// A directory rename publishes the completed copy in one operation.
	// Rename refuses an existing nonempty profile; no original file is moved.
	return os.Rename(tmp, dst)
}
