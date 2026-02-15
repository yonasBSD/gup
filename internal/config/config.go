// Package config define gup command setting.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/nao1215/gup/internal/cmdinfo"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/shogo82148/pointer"
)

// ConfigFileName is gup command configuration file
const ConfigFileName = "gup.conf"

// FilePath return configuration-file path.
func FilePath() string {
	return filepath.Join(DirPath(), ConfigFileName)
}

// LocalFilePath returns the path to gup.conf in the current directory.
func LocalFilePath() string {
	return filepath.Join(".", ConfigFileName)
}

// DirPath return directory path that store configuration-file.
// Default path is $HOME/.config/gup.
func DirPath() string {
	return filepath.Join(xdg.ConfigHome, cmdinfo.Name)
}

// ResolveImportFilePath resolves config file path for import.
// Priority: explicit path > default config path (if exists) > ./gup.conf (if exists) > default config path.
func ResolveImportFilePath(explicitPath string) string {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		return explicitPath
	}

	defaultPath := FilePath()
	if fileutil.IsFile(defaultPath) {
		return defaultPath
	}

	local := LocalFilePath()
	if fileutil.IsFile(local) {
		return local
	}
	return defaultPath
}

// ResolveExportFilePath resolves config file path for export.
// Priority: explicit path > default config path.
func ResolveExportFilePath(explicitPath string) string {
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath != "" {
		return explicitPath
	}
	return FilePath()
}

// ReadConfFile return contents of configuration-file (package information)
func ReadConfFile(path string) ([]goutil.Package, error) {
	contents, err := readFileToList(path)
	if err != nil {
		return nil, fmt.Errorf("can't read %s: %w", path, err)
	}

	pkgs := []goutil.Package{}
	for _, v := range contents {
		pkg := goutil.Package{}
		binVer := goutil.Version{Current: "", Latest: ""}
		goVer := goutil.Version{Current: "<from gup.conf>", Latest: ""}

		v = deleteComment(v)
		if isBlank(v) {
			continue
		}

		name, rest, found := strings.Cut(v, "=")
		if !found {
			return nil, errors.New(path + " is not gup.conf file")
		}
		name = strings.TrimSpace(name)
		rest = strings.TrimSpace(rest)

		importPath, version, found := strings.Cut(rest, "@")
		if !found {
			return nil, fmt.Errorf("%s is old gup.conf format. expected '<name> = <import-path>@<version>'", path)
		}
		importPath = strings.TrimSpace(importPath)
		version = strings.TrimSpace(version)
		if name == "" || importPath == "" || version == "" {
			return nil, errors.New(path + " is not gup.conf file")
		}

		pkg.Name = name
		pkg.ImportPath = importPath
		binVer.Current = version
		pkg.Version = pointer.Ptr(binVer)
		pkg.GoVersion = pointer.Ptr(goVer)
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// WriteConfFile write package information at configuration-file.
func WriteConfFile(file io.Writer, pkgs []goutil.Package) error {
	var builder strings.Builder
	for _, v := range pkgs {
		version := "latest"
		if v.Version != nil && strings.TrimSpace(v.Version.Current) != "" {
			version = strings.TrimSpace(v.Version.Current)
		}
		builder.WriteString(fmt.Sprintf("%s = %s@%s\n", v.Name, v.ImportPath, version))
	}

	_, err := file.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("can't write gup.conf: %w", err)
	}
	return nil
}

func isBlank(line string) bool {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "\n", "")
	return len(line) == 0
}

func deleteComment(line string) string {
	line, _, _ = strings.Cut(line, "#")
	return line
}

// readFileToList convert file content to string list.
func readFileToList(path string) (_ []string, err error) {
	var strList []string
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		strList = append(strList, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return strList, nil
}
