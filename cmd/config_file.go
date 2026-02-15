package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
)

func writeConfigFile(path string, pkgs []goutil.Package) (err error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(path), fileutil.FileModeCreatingDir); err != nil {
		return fmt.Errorf("%s: %w", "can not make config directory", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s %s: %w", "can't update", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err = config.WriteConfFile(file, pkgs); err != nil {
		return err
	}
	return nil
}
