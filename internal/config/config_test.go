package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/nao1215/gup/internal/goutil"
)

func withTempXDG(t *testing.T) func() {
	t.Helper()

	origConfig := xdg.ConfigHome
	origData := xdg.DataHome
	origCache := xdg.CacheHome

	base := t.TempDir()
	xdg.ConfigHome = filepath.Join(base, "config")
	xdg.DataHome = filepath.Join(base, "data")
	xdg.CacheHome = filepath.Join(base, "cache")

	for _, dir := range []string{xdg.ConfigHome, xdg.DataHome, xdg.CacheHome} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("failed to create XDG dir %s: %v", dir, err)
		}
	}

	return func() {
		xdg.ConfigHome = origConfig
		xdg.DataHome = origData
		xdg.CacheHome = origCache
	}
}

func TestDirAndFilePath(t *testing.T) { //nolint:paralleltest // modifies xdg globals
	cleanup := withTempXDG(t)
	defer cleanup()

	if got := DirPath(); got != filepath.Join(xdg.ConfigHome, "gup") {
		t.Fatalf("DirPath() = %s, want %s", got, filepath.Join(xdg.ConfigHome, "gup"))
	}

	if got := FilePath(); got != filepath.Join(xdg.ConfigHome, "gup", ConfigFileName) {
		t.Fatalf("FilePath() = %s, want %s", got, filepath.Join(xdg.ConfigHome, "gup", ConfigFileName))
	}
}

func TestWriteConfFile(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	pkgs := []goutil.Package{
		{
			Name:       "foo",
			ImportPath: "example.com/foo",
			Version:    &goutil.Version{Current: "v1.2.3"},
		},
		{
			Name:       "bar",
			ImportPath: "example.com/bar",
		},
	}

	if err := WriteConfFile(&buf, pkgs); err != nil {
		t.Fatalf("WriteConfFile() error = %v", err)
	}

	want := "foo = example.com/foo@v1.2.3\nbar = example.com/bar@latest\n"
	if got := buf.String(); got != want {
		t.Fatalf("WriteConfFile() output = %q, want %q", got, want)
	}
}

func TestReadConfFile(t *testing.T) { //nolint:paralleltest // modifies xdg globals
	cleanup := withTempXDG(t)
	defer cleanup()

	confPath := filepath.Join(xdg.ConfigHome, "gup.conf")
	content := "foo = example.com/foo@v1.2.3\n# comment\nbar = example.com/bar@v4.5.6\n"
	if err := os.WriteFile(confPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp conf file: %v", err)
	}

	pkgs, err := ReadConfFile(confPath)
	if err != nil {
		t.Fatalf("ReadConfFile() error = %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("ReadConfFile() len = %d, want 2", len(pkgs))
	}
	if pkgs[0].Name != "foo" || pkgs[0].ImportPath != "example.com/foo" {
		t.Fatalf("first pkg mismatch: %+v", pkgs[0])
	}
	if pkgs[0].Version == nil || pkgs[0].Version.Current != "v1.2.3" {
		t.Fatalf("first pkg version mismatch: %+v", pkgs[0].Version)
	}
	if pkgs[1].Name != "bar" || pkgs[1].ImportPath != "example.com/bar" {
		t.Fatalf("second pkg mismatch: %+v", pkgs[1])
	}
	if pkgs[1].Version == nil || pkgs[1].Version.Current != "v4.5.6" {
		t.Fatalf("second pkg version mismatch: %+v", pkgs[1].Version)
	}
}

func TestReadConfFile_OldFormat(t *testing.T) {
	t.Parallel()

	confPath := filepath.Join(t.TempDir(), "gup.conf")
	content := "foo = example.com/foo\n"
	if err := os.WriteFile(confPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp conf file: %v", err)
	}

	if _, err := ReadConfFile(confPath); err == nil {
		t.Fatal("ReadConfFile() should return error for old format")
	}
}

func TestResolveImportFilePath(t *testing.T) { //nolint:paralleltest // changes working dir
	cleanup := withTempXDG(t)
	defer cleanup()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	custom := filepath.Join(t.TempDir(), "custom.conf")
	if got := ResolveImportFilePath(custom); got != custom {
		t.Fatalf("ResolveImportFilePath(explicit) = %s, want %s", got, custom)
	}

	if got := ResolveImportFilePath(""); got != FilePath() {
		t.Fatalf("ResolveImportFilePath(default) = %s, want %s", got, FilePath())
	}

	local := filepath.Join(tmpDir, ConfigFileName)
	if err := os.WriteFile(local, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolveImportFilePath(""); got != LocalFilePath() {
		t.Fatalf("ResolveImportFilePath(local) = %s, want %s", got, LocalFilePath())
	}

	xdgFile := FilePath()
	if err := os.MkdirAll(filepath.Dir(xdgFile), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(xdgFile, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolveImportFilePath(""); got != FilePath() {
		t.Fatalf("ResolveImportFilePath(xdg-priority) = %s, want %s", got, FilePath())
	}
}

func TestResolveExportFilePath(t *testing.T) {
	t.Parallel()

	custom := filepath.Join(t.TempDir(), "custom.conf")
	if got := ResolveExportFilePath(custom); got != custom {
		t.Fatalf("ResolveExportFilePath(explicit) = %s, want %s", got, custom)
	}
	if got := ResolveExportFilePath(""); got != FilePath() {
		t.Fatalf("ResolveExportFilePath(default) = %s, want %s", got, FilePath())
	}
}
