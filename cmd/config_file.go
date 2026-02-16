package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
)

var writeConfFile = config.WriteConfFile //nolint:gochecknoglobals // swapped in tests

func writeConfigFile(path string, pkgs []goutil.Package) (err error) {
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fileutil.FileModeCreatingDir); err != nil {
		return fmt.Errorf("%s: %w", "can not make config directory", err)
	}

	file, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("%s %s: %w", "can't create temp file for", path, err)
	}
	tmpPath := file.Name()
	defer func() {
		if file != nil {
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if err = writeConfFile(file, pkgs); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("%s %s: %w", "can't sync temp config file for", path, err)
	}
	if err = file.Close(); err != nil {
		file = nil
		return fmt.Errorf("%s %s: %w", "can't close temp config file for", path, err)
	}
	file = nil

	if err = renameWithReplace(tmpPath, path); err != nil {
		return fmt.Errorf("%s %s: %w", "can't update", path, err)
	}

	return nil
}

func renameWithReplace(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		// Windows cannot overwrite an existing file with os.Rename.
		// Retry as remove-then-rename only when the destination likely exists.
		if !shouldRetryRenameWithReplace(err, dst) {
			return err
		}
		if errRemove := os.Remove(dst); errRemove != nil {
			return errRemove
		}
		if errRetry := os.Rename(src, dst); errRetry != nil {
			return errRetry
		}
	}
	return nil
}

func shouldRetryRenameWithReplace(renameErr error, dst string) bool {
	if os.IsExist(renameErr) {
		return true
	}
	if runtime.GOOS != goosWindows {
		return false
	}
	_, err := os.Stat(dst)
	return err == nil
}
