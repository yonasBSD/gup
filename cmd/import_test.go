//nolint:paralleltest,errcheck,gosec
package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
)

func Test_runImport_flagErrors(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		want int
	}{
		{
			name: "missing dry-run flag",
			cmd:  &cobra.Command{},
			want: 1,
		},
		{
			name: "missing notify flag",
			cmd: func() *cobra.Command {
				c := &cobra.Command{}
				c.Flags().Bool("dry-run", false, "")
				c.Flags().String("file", "gup.json", "")
				return c
			}(),
			want: 1,
		},
		{
			name: "missing jobs flag",
			cmd: func() *cobra.Command {
				c := &cobra.Command{}
				c.Flags().Bool("dry-run", false, "")
				c.Flags().String("file", "gup.json", "")
				c.Flags().Bool("notify", false, "")
				return c
			}(),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgStdout := print.Stdout
			orgStderr := print.Stderr
			pr, pw, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			print.Stdout = pw
			print.Stderr = pw

			got := runImport(tt.cmd, nil)
			pw.Close()
			print.Stdout = orgStdout
			print.Stderr = orgStderr
			pr.Close()

			if got != tt.want {
				t.Errorf("runImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runImport_notUseGoCmd(t *testing.T) {
	t.Setenv("PATH", "")

	cmd := newImportCmd()

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	got := runImport(cmd, nil)
	pw.Close()
	print.Stdout = orgStdout
	print.Stderr = orgStderr

	if got != 1 {
		t.Errorf("runImport() = %v, want 1", got)
	}

	buf := bytes.Buffer{}
	_, _ = io.Copy(&buf, pr)
	pr.Close()
	if !strings.Contains(buf.String(), "you didn't install golang") {
		t.Errorf("expected go command error, got: %s", buf.String())
	}
}

func Test_runImport_fileNotFound(t *testing.T) {
	cmd := newImportCmd()
	if err := cmd.Flags().Set("file", "/no/such/file.json"); err != nil {
		t.Fatal(err)
	}

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	got := runImport(cmd, nil)
	pw.Close()
	print.Stdout = orgStdout
	print.Stderr = orgStderr

	if got != 1 {
		t.Errorf("runImport() = %v, want 1", got)
	}

	buf := bytes.Buffer{}
	_, _ = io.Copy(&buf, pr)
	pr.Close()

	if !strings.Contains(buf.String(), "is not found") {
		t.Errorf("expected 'is not found' error, got: %s", buf.String())
	}
}

func Test_runImport_emptyConf(t *testing.T) {
	// Create a temporary conf file with no packages
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(confPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newImportCmd()
	if err := cmd.Flags().Set("file", confPath); err != nil {
		t.Fatal(err)
	}

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	got := runImport(cmd, nil)
	pw.Close()
	print.Stdout = orgStdout
	print.Stderr = orgStderr

	if got != 1 {
		t.Errorf("runImport() = %v, want 1", got)
	}

	buf := bytes.Buffer{}
	_, _ = io.Copy(&buf, pr)
	pr.Close()

	if !strings.Contains(buf.String(), "unable to import package") {
		t.Errorf("expected 'unable to import package' error, got: %s", buf.String())
	}
}

func Test_runImport_jobsClamp(t *testing.T) {
	// Create a conf file that will be found but has no packages
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(confPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newImportCmd()
	if err := cmd.Flags().Set("file", confPath); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("jobs", "0"); err != nil {
		t.Fatal(err)
	}

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	// Should not panic with jobs=0 (clamped to 1)
	got := runImport(cmd, nil)
	pw.Close()
	print.Stdout = orgStdout
	print.Stderr = orgStderr
	pr.Close()

	// Expect exit code 1 because the conf file has no packages
	if got != 1 {
		t.Errorf("runImport() = %v, want 1", got)
	}
}

func Test_installFromConfig_UseVersion(t *testing.T) {
	originalInstaller := installByVersion
	t.Cleanup(func() {
		installByVersion = originalInstaller
	})

	var gotImportPath string
	var gotVersion string
	installByVersion = func(importPath, version string) error {
		gotImportPath = importPath
		gotVersion = version
		return nil
	}

	pkgs := []goutil.Package{
		{
			Name:       "gup",
			ImportPath: "github.com/nao1215/gup",
			Version:    &goutil.Version{Current: "v1.0.0"},
		},
	}

	if got := installFromConfig(pkgs, false, false, 1); got != 0 {
		t.Fatalf("installFromConfig() = %d, want 0", got)
	}

	if gotImportPath != "github.com/nao1215/gup" {
		t.Fatalf("install import path = %s, want %s", gotImportPath, "github.com/nao1215/gup")
	}
	if gotVersion != "v1.0.0" {
		t.Fatalf("install version = %s, want %s", gotVersion, "v1.0.0")
	}
}

func Test_versionFromConfig_NormalizeDevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pkg  goutil.Package
		want string
	}{
		{
			name: "devel with parentheses",
			pkg: goutil.Package{
				Version: &goutil.Version{Current: "(devel)"},
			},
			want: "latest",
		},
		{
			name: "devel without parentheses",
			pkg: goutil.Package{
				Version: &goutil.Version{Current: "devel"},
			},
			want: "latest",
		},
		{
			name: "regular version",
			pkg: goutil.Package{
				Version: &goutil.Version{Current: "v1.2.3"},
			},
			want: "v1.2.3",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := versionFromConfig(tt.pkg)
			if err != nil {
				t.Fatalf("versionFromConfig() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("versionFromConfig() = %s, want %s", got, tt.want)
			}
		})
	}
}
