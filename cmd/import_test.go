//nolint:paralleltest,errcheck,gosec
package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
				c.Flags().String("input", "gup.conf", "")
				return c
			}(),
			want: 1,
		},
		{
			name: "missing jobs flag",
			cmd: func() *cobra.Command {
				c := &cobra.Command{}
				c.Flags().Bool("dry-run", false, "")
				c.Flags().String("input", "gup.conf", "")
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

func Test_runImport_fileNotFound(t *testing.T) {
	cmd := newImportCmd()
	if err := cmd.Flags().Set("input", "/no/such/file.conf"); err != nil {
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
	confPath := filepath.Join(tmpDir, "empty.conf")
	if err := os.WriteFile(confPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newImportCmd()
	if err := cmd.Flags().Set("input", confPath); err != nil {
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
	confPath := filepath.Join(tmpDir, "test.conf")
	if err := os.WriteFile(confPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := newImportCmd()
	if err := cmd.Flags().Set("input", confPath); err != nil {
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
