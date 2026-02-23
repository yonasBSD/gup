//go:build !int

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBugReport(t *testing.T) {
	t.Parallel()

	t.Run("Check bug-report --help", func(t *testing.T) {
		b := bytes.NewBufferString("")

		copyRootCmd := newRootCmd()

		copyRootCmd.SetOut(b)
		copyRootCmd.SetArgs([]string{"bug-report", "--help"})

		if err := copyRootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
		gotBytes, err := io.ReadAll(b)
		if err != nil {
			t.Fatal(err)
		}
		gotBytes = bytes.ReplaceAll(gotBytes, []byte("\r\n"), []byte("\n"))

		wantBytes, err := os.ReadFile(filepath.Join("testdata", "bug_report", "bug_report.txt"))
		if err != nil {
			t.Fatal(err)
		}
		wantBytes = bytes.ReplaceAll(wantBytes, []byte("\r\n"), []byte("\n"))

		if diff := cmp.Diff(strings.TrimSpace(string(gotBytes)), strings.TrimSpace(string(wantBytes))); diff != "" {
			t.Errorf("value is mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_bugReport(t *testing.T) {
	t.Parallel()

	cmd := newBugReportCmd()
	cmd.Version = "v0.0.0"

	wantReturnVal := 0
	gotReturnVal := bugReport(cmd, nil, func(s string) bool {
		if !strings.Contains(s, "v0.0.0") {
			t.Errorf("Expected bug report to contain version number 'v0.0.0', but got: %s", s)
		}
		return true
	})
	if gotReturnVal != wantReturnVal {
		t.Errorf("bugReport() = %d; want %d", gotReturnVal, wantReturnVal)
	}
}

func Test_bugReport_fallbackVersion(t *testing.T) {
	t.Parallel()

	cmd := newBugReportCmd()
	cmd.Version = ""

	gotReturnVal := bugReport(cmd, nil, func(s string) bool {
		if !strings.Contains(s, "gup+version") {
			t.Errorf("expected bug report URL to contain gup version section, but got: %s", s)
		}
		return true
	})
	if gotReturnVal != 0 {
		t.Errorf("bugReport() = %d; want %d", gotReturnVal, 0)
	}
}

func Test_bugReport_fallbackOutput(t *testing.T) {
	t.Parallel()

	cmd := newBugReportCmd()
	cmd.Version = "v9.9.9"

	orgStdout := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pw
	t.Cleanup(func() {
		os.Stdout = orgStdout
	})

	if got := bugReport(cmd, nil, func(string) bool { return false }); got != 0 {
		t.Fatalf("bugReport() = %d, want 0", got)
	}
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}

	gotOutput := string(body)
	if !strings.Contains(gotOutput, "Please file a new issue at https://github.com/nao1215/gup/issues/new using this template:") {
		t.Fatalf("fallback guide is missing: %s", gotOutput)
	}
	if !strings.Contains(gotOutput, "## gup version\nv9.9.9\n") {
		t.Fatalf("version template is missing: %s", gotOutput)
	}
}
