//nolint:paralleltest,errcheck,gosec
package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
)

const testVersionZero = "v0.0.0"
const testVersionNine = "v9.9.9"

func Test_gup(t *testing.T) {
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name   string
		args   args
		want   int
		stderr []string
	}{
		{
			name: "parser --dry-run argument error",
			args: args{
				cmd:  &cobra.Command{},
				args: []string{},
			},
			want: 1,
			stderr: []string{
				"gup:ERROR: can not parse command line argument (--dry-run): flag accessed but not defined: dry-run",
				"",
			},
		},
		{
			name: "parser --notify argument error",
			args: args{
				cmd:  &cobra.Command{},
				args: []string{},
			},
			want: 1,
			stderr: []string{
				"gup:ERROR: can not parse command line argument (--notify): flag accessed but not defined: notify",
				"",
			},
		},
		{
			name: "parser --jobs argument error",
			args: args{
				cmd:  &cobra.Command{},
				args: []string{},
			},
			want: 1,
			stderr: []string{
				"gup:ERROR: can not parse command line argument (--jobs): flag accessed but not defined: jobs",
				"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "parser --dry-run argument error":
				tt.args.cmd.Flags().BoolP("notify", "N", false, "enable desktop notifications")
				tt.args.cmd.Flags().BoolP("jobs", "j", false, "Specify the number of CPU cores to use")
			case "parser --notify argument error":
				tt.args.cmd.Flags().BoolP("dry-run", "n", false, "perform the trial update with no changes")
				tt.args.cmd.Flags().BoolP("jobs", "j", false, "Specify the number of CPU cores to use")
			case "parser --jobs argument error":
				tt.args.cmd.Flags().BoolP("dry-run", "n", false, "perform the trial update with no changes")
				tt.args.cmd.Flags().BoolP("notify", "N", false, "enable desktop notifications")
			}

			OsExit = func(code int) {}
			defer func() {
				OsExit = os.Exit
			}()

			orgStdout := print.Stdout
			orgStderr := print.Stderr
			pr, pw, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			print.Stdout = pw
			print.Stderr = pw

			if got := gup(tt.args.cmd, tt.args.args); got != tt.want {
				t.Errorf("gup() = %v, want %v", got, tt.want)
			}
			if err := pw.Close(); err != nil {
				t.Fatal(err)
			}
			print.Stdout = orgStdout
			print.Stderr = orgStderr

			buf := bytes.Buffer{}
			_, err = io.Copy(&buf, pr)
			if err != nil {
				t.Error(err)
			}
			defer pr.Close()
			got := strings.Split(buf.String(), "\n")

			if diff := cmp.Diff(tt.stderr, got); diff != "" {
				t.Errorf("value is mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_extractUserSpecifyPkg(t *testing.T) {
	type args struct {
		pkgs    []goutil.Package
		targets []string
	}
	tests := []struct {
		name string
		args args
		want []goutil.Package
	}{
		{
			name: "find user specify package",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "test1",
					},
					{
						Name: "test2",
					},
					{
						Name: "test3",
					},
				},
				targets: []string{"test2"},
			},
			want: []goutil.Package{
				{
					Name: "test2",
				},
			},
		},
		{
			name: "can notfind user specify package",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "test1",
					},
					{
						Name: "test2",
					},
					{
						Name: "test3",
					},
				},
				targets: []string{"test4"},
			},
			want: []goutil.Package{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUserSpecifyPkg(tt.args.pkgs, tt.args.targets)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("value is mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_completePathBinaries_prefix(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Setenv("GOBIN", filepath.Join("testdata", "check_success_for_windows"))
	} else {
		t.Setenv("GOBIN", filepath.Join("testdata", "check_success"))
	}

	got, _ := completePathBinaries(nil, nil, "ga")
	if len(got) == 0 {
		t.Fatalf("completion should return at least one candidate")
	}

	for _, name := range got {
		if !strings.HasPrefix(strings.ToLower(name), "ga") {
			t.Fatalf("unexpected completion candidate for prefix ga: %s", name)
		}
	}
}

func Test_catchSignal(t *testing.T) {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})

	go catchSignal(signals, func() {
		close(done)
	})
	signals <- syscall.SIGINT

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("catchSignal should call cancel function")
	}
}

func Test_gup_ignoreGoUpdateFlag(t *testing.T) {
	t.Setenv("GOBIN", filepath.Join("testdata", "check_success"))

	cmd := newUpdateCmd()
	if err := cmd.Flags().Set("ignore-go-update", "true"); err != nil {
		t.Fatalf("failed to set ignore-go-update flag: %v", err)
	}

	origGetLatest := getLatestVer
	origInstallLatest := installLatest
	origInstallMain := installMainOrMaster
	origInstallByVersionUpd := installByVersionUpd
	getLatestVer = func(string) (string, error) { return testVersionZero, nil }
	installLatest = func(string) error { return nil }
	installMainOrMaster = func(string) error { return nil }
	installByVersionUpd = func(string, string) error { return nil }
	defer func() {
		getLatestVer = origGetLatest
		installLatest = origInstallLatest
		installMainOrMaster = origInstallMain
		installByVersionUpd = origInstallByVersionUpd
	}()

	OsExit = func(code int) {}
	defer func() {
		OsExit = os.Exit
	}()

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	if got := gup(cmd, []string{}); got != 0 {
		t.Fatalf("gup() = %v, want %v", got, 0)
	}
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}
	print.Stdout = orgStdout
	print.Stderr = orgStderr

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, pr); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = pr.Close()
	}()

	output := strings.Split(buf.String(), "\n")
	if len(output) == 0 || !strings.Contains(output[0], "update binary under") {
		t.Fatalf("unexpected output: %v", output)
	}
}

func Test_gup_invalidConfigFile(t *testing.T) {
	setupXDGBase(t)
	t.Setenv("GOBIN", filepath.Join("testdata", "check_success"))
	helper_stubUpdateOps(t)

	confPath := config.FilePath()
	if err := os.MkdirAll(filepath.Dir(confPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(confPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := gup(newUpdateCmd(), []string{}); got != 0 {
		t.Fatalf("gup() = %v, want %v (invalid config should be warned, not fatal)", got, 0)
	}
}

func Test_gup_dryRun(t *testing.T) {
	t.Setenv("GOBIN", filepath.Join("testdata", "check_success"))

	cmd := newUpdateCmd()
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("failed to set dry-run flag: %v", err)
	}

	var installCalled atomic.Bool
	origGetLatest := getLatestVer
	origInstallLatest := installLatest
	origInstallMain := installMainOrMaster
	origInstallByVersionUpd := installByVersionUpd
	getLatestVer = func(string) (string, error) { return testVersionNine, nil }
	installLatest = func(string) error {
		installCalled.Store(true)
		return nil
	}
	installMainOrMaster = func(string) error {
		installCalled.Store(true)
		return nil
	}
	installByVersionUpd = func(string, string) error {
		installCalled.Store(true)
		return nil
	}
	defer func() {
		getLatestVer = origGetLatest
		installLatest = origInstallLatest
		installMainOrMaster = origInstallMain
		installByVersionUpd = origInstallByVersionUpd
	}()

	OsExit = func(code int) {}
	defer func() {
		OsExit = os.Exit
	}()

	if got := gup(cmd, []string{}); got != 0 {
		t.Fatalf("gup() = %v, want %v", got, 0)
	}
	if !installCalled.Load() {
		t.Fatalf("expected installer to be invoked in dry-run mode")
	}
	if gobin := os.Getenv("GOBIN"); !strings.Contains(gobin, "check_success") {
		t.Fatalf("GOBIN should be restored after dry-run, got %s", gobin)
	}
}

func Test_excludeUserSpecifiedPkg(t *testing.T) {
	type args struct {
		pkgs           []goutil.Package
		excludePkgList []string
	}
	tests := []struct {
		name string
		args args
		want []goutil.Package
	}{
		{
			name: "find user specify package",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "pkg1",
					},
					{
						Name: "pkg2",
					},
					{
						Name: "pkg3",
					},
				},
				excludePkgList: []string{"pkg1", "pkg3"},
			},
			want: []goutil.Package{
				{
					Name: "pkg2",
				},
			},
		},
		{
			name: "find user specify package (exclude all package)",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "pkg1",
					},
					{
						Name: "pkg2",
					},
					{
						Name: "pkg3",
					},
				},
				excludePkgList: []string{"pkg1", "pkg2", "pkg3"},
			},
			want: []goutil.Package{},
		},
		{
			name: "If the excluded package does not exist",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "pkg1",
					},
					{
						Name: "pkg2",
					},
					{
						Name: "pkg3",
					},
				},
				excludePkgList: []string{"pkg4"},
			},
			want: []goutil.Package{
				{
					Name: "pkg1",
				},
				{
					Name: "pkg2",
				},
				{
					Name: "pkg3",
				},
			},
		},
		{
			name: "exclude package names are trimmed",
			args: args{
				pkgs: []goutil.Package{
					{
						Name: "pkg1",
					},
					{
						Name: "pkg2",
					},
					{
						Name: "pkg3",
					},
				},
				excludePkgList: []string{" pkg1", "pkg3 "},
			},
			want: []goutil.Package{
				{
					Name: "pkg2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excludePkgs(tt.args.excludePkgList, tt.args.pkgs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("value is mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_update_not_use_go_cmd(t *testing.T) {
	t.Run("Not found go command", func(t *testing.T) {
		t.Setenv("PATH", "")

		orgStdout := print.Stdout
		orgStderr := print.Stderr
		pr, pw, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		print.Stdout = pw
		print.Stderr = pw

		if got := gup(newUpdateCmd(), []string{}); got != 1 {
			t.Errorf("gup() = %v, want %v", got, 1)
		}
		if err := pw.Close(); err != nil {
			t.Fatal(err)
		}
		print.Stdout = orgStdout
		print.Stderr = orgStderr

		buf := bytes.Buffer{}
		_, err = io.Copy(&buf, pr)
		if err != nil {
			t.Error(err)
		}
		defer pr.Close()
		got := strings.Split(buf.String(), "\n")

		want := []string{}
		if runtime.GOOS == goosWindows {
			want = append(want, `gup:ERROR: you didn't install golang: exec: "go": executable file not found in %PATH%`)
			want = append(want, "")
		} else {
			want = append(want, `gup:ERROR: you didn't install golang: exec: "go": executable file not found in $PATH`)
			want = append(want, "")
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("value is mismatch (-want +got):\n%s", diff)
		}
	})
}

func Test_desktopNotifyIfNeeded(t *testing.T) {
	type args struct {
		result int
		enable bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Notify update success",
			args: args{
				result: 0,
				enable: true,
			},
		},

		{
			name: "Notify update fail",
			args: args{
				result: 1,
				enable: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desktopNotifyIfNeeded(tt.args.result, tt.args.enable)
		})
	}
}

func TestExtractUserSpecifyPkg(t *testing.T) {
	pkgs := []goutil.Package{
		{Name: "pkg1"},
		{Name: "pkg2.exe"},
		{Name: "pkg3"},
	}
	targets := []string{"pkg1", "pkg2.exe"}
	if runtime.GOOS == goosWindows {
		targets = []string{"pkg1", "pkg2"}
	}

	expected := []goutil.Package{
		{Name: "pkg1"},
		{Name: "pkg2.exe"},
	}
	actual := extractUserSpecifyPkg(pkgs, targets)

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):\n%s", diff)
	}
}

func Test_gup_jobsClamp(t *testing.T) {
	t.Setenv("GOBIN", filepath.Join("testdata", "check_success"))

	cmd := newUpdateCmd()
	if err := cmd.Flags().Set("jobs", "-1"); err != nil {
		t.Fatalf("failed to set jobs flag: %v", err)
	}

	origGetLatest := getLatestVer
	origInstallLatest := installLatest
	origInstallMain := installMainOrMaster
	origInstallByVersionUpd := installByVersionUpd
	getLatestVer = func(string) (string, error) { return testVersionZero, nil }
	installLatest = func(string) error { return nil }
	installMainOrMaster = func(string) error { return nil }
	installByVersionUpd = func(string, string) error { return nil }
	defer func() {
		getLatestVer = origGetLatest
		installLatest = origInstallLatest
		installMainOrMaster = origInstallMain
		installByVersionUpd = origInstallByVersionUpd
	}()

	OsExit = func(code int) {}
	defer func() {
		OsExit = os.Exit
	}()

	orgStdout := print.Stdout
	orgStderr := print.Stderr
	_, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	print.Stdout = pw
	print.Stderr = pw

	// Should not hang with jobs=-1 (clamped to 1)
	got := gup(cmd, []string{})
	pw.Close()
	print.Stdout = orgStdout
	print.Stderr = orgStderr

	if got != 0 {
		t.Errorf("gup() = %v, want 0", got)
	}
}

func Test_replaceImportPathPrefix(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		oldModule  string
		newModule  string
		wantImport string
	}{
		{
			name:       "same as module root",
			importPath: "github.com/cosmtrek/air",
			oldModule:  "github.com/cosmtrek/air",
			newModule:  "github.com/air-verse/air",
			wantImport: "github.com/air-verse/air",
		},
		{
			name:       "subdir path",
			importPath: "github.com/cosmtrek/air/cmd/air",
			oldModule:  "github.com/cosmtrek/air",
			newModule:  "github.com/air-verse/air",
			wantImport: "github.com/air-verse/air/cmd/air",
		},
		{
			name:       "empty import path",
			importPath: "",
			oldModule:  "github.com/cosmtrek/air",
			newModule:  "github.com/air-verse/air",
			wantImport: "github.com/air-verse/air",
		},
		{
			name:       "no prefix match keeps original import path",
			importPath: "github.com/example/tool",
			oldModule:  "github.com/cosmtrek/air",
			newModule:  "github.com/air-verse/air",
			wantImport: "github.com/example/tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceImportPathPrefix(tt.importPath, tt.oldModule, tt.newModule)
			if got != tt.wantImport {
				t.Errorf("replaceImportPathPrefix() = %q, want %q", got, tt.wantImport)
			}
		})
	}
}

func Test_removeOldBinaryIfRenamed(t *testing.T) {
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)

	oldBinaryPath := filepath.Join(gobin, "old-tool")
	if err := os.WriteFile(oldBinaryPath, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := removeOldBinaryIfRenamed("old-tool", "new-tool"); err != nil {
		t.Fatalf("removeOldBinaryIfRenamed() error = %v", err)
	}
	if _, err := os.Stat(oldBinaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old binary should be removed. err = %v", err)
	}

	sameBinaryPath := filepath.Join(gobin, "same-tool")
	if err := os.WriteFile(sameBinaryPath, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeOldBinaryIfRenamed("same-tool", "same-tool"); err != nil {
		t.Fatalf("removeOldBinaryIfRenamed() should not fail for same name: %v", err)
	}
	if _, err := os.Stat(sameBinaryPath); err != nil {
		t.Fatalf("binary should remain when names are same. err = %v", err)
	}
}

func Test_update_modulePathChangedOnGetLatest(t *testing.T) {
	const (
		oldModule = "github.com/cosmtrek/air"
		newModule = "github.com/air-verse/air"
		oldImport = "github.com/cosmtrek/air/cmd/air"
		newImport = "github.com/air-verse/air/cmd/air"
	)

	origGetLatest := getLatestVer
	origInstallLatest := installLatest
	origInstallMain := installMainOrMaster
	origInstallByVersionUpd := installByVersionUpd
	defer func() {
		getLatestVer = origGetLatest
		installLatest = origInstallLatest
		installMainOrMaster = origInstallMain
		installByVersionUpd = origInstallByVersionUpd
	}()

	var latestCalls []string
	getLatestVer = func(modulePath string) (string, error) {
		latestCalls = append(latestCalls, modulePath)
		if modulePath == oldModule {
			return "", modulePathMismatchErr(oldModule, newModule)
		}
		if modulePath == newModule {
			return "v1.2.3", nil
		}
		return "", errors.New("unexpected module path")
	}

	var installCalls []string
	installLatest = func(importPath string) error {
		installCalls = append(installCalls, importPath)
		return nil
	}
	installMainOrMaster = func(string) error {
		t.Fatal("installMainOrMaster should not be called")
		return nil
	}
	installByVersionUpd = func(string, string) error {
		t.Fatal("installByVersionUpd should not be called")
		return nil
	}

	pkgs := []goutil.Package{
		{
			Name:       "air",
			ImportPath: oldImport,
			ModulePath: oldModule,
			Version: &goutil.Version{
				Current: "v1.2.3",
			},
			GoVersion: &goutil.Version{
				Current: "go1.22.4",
				Latest:  "go1.22.4",
			},
		},
	}

	channelMap := map[string]goutil.UpdateChannel{"air": goutil.UpdateChannelLatest}
	if got, _, _ := updateWithChannels(pkgs, false, false, 1, true, channelMap); got != 0 {
		t.Fatalf("updateWithChannels() = %d, want 0", got)
	}
	if diff := cmp.Diff([]string{oldModule, newModule}, latestCalls); diff != "" {
		t.Errorf("latest module path calls mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{newImport}, installCalls); diff != "" {
		t.Errorf("install import path calls mismatch (-want +got):\n%s", diff)
	}
}

func Test_update_modulePathChangedOnInstall(t *testing.T) {
	const (
		oldModule = "github.com/cosmtrek/air"
		newModule = "github.com/air-verse/air"
		oldImport = "github.com/cosmtrek/air"
		newImport = "github.com/air-verse/air"
	)

	origGetLatest := getLatestVer
	origInstallLatest := installLatest
	origInstallMain := installMainOrMaster
	origInstallByVersionUpd := installByVersionUpd
	defer func() {
		getLatestVer = origGetLatest
		installLatest = origInstallLatest
		installMainOrMaster = origInstallMain
		installByVersionUpd = origInstallByVersionUpd
	}()

	getLatestVer = func(string) (string, error) { return testVersionNine, nil }
	installMainOrMaster = func(string) error {
		t.Fatal("installMainOrMaster should not be called")
		return nil
	}
	installByVersionUpd = func(string, string) error {
		t.Fatal("installByVersionUpd should not be called")
		return nil
	}

	var installCalls []string
	installLatest = func(importPath string) error {
		installCalls = append(installCalls, importPath)
		switch len(installCalls) {
		case 1:
			return modulePathMismatchErr(oldModule, newModule)
		case 2:
			return nil
		default:
			return errors.New("unexpected install call")
		}
	}

	pkgs := []goutil.Package{
		{
			Name:       "air",
			ImportPath: oldImport,
			ModulePath: newModule,
			Version: &goutil.Version{
				Current: "v1.0.0",
			},
			GoVersion: &goutil.Version{
				Current: "go1.22.4",
				Latest:  "go1.22.4",
			},
		},
	}

	channelMap := map[string]goutil.UpdateChannel{"air": goutil.UpdateChannelLatest}
	if got, _, _ := updateWithChannels(pkgs, false, false, 1, true, channelMap); got != 0 {
		t.Fatalf("updateWithChannels() = %d, want 0", got)
	}
	if diff := cmp.Diff([]string{oldImport, newImport}, installCalls); diff != "" {
		t.Errorf("install import path calls mismatch (-want +got):\n%s", diff)
	}
}

func modulePathMismatchErr(requiredPath, declaredPath string) error {
	return errors.New("version constraints conflict:\n" +
		"module declares its path as: " + declaredPath + "\n" +
		"but was required as: " + requiredPath)
}
