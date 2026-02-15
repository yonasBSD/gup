package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
)

var installByVersion = goutil.Install //nolint:gochecknoglobals // swapped in tests

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Install command according to gup.json.",
		Long: `Install command according to gup.json.
	
Use the export subcommand if you want to install the same golang
binaries across multiple systems. After you create gup.json by 
import subcommand in another environment, you save conf-file in
$XDG_CONFIG_HOME/.config/gup/gup.json (e.g. $HOME/.config/gup/gup.json.)
Finally, you execute the export subcommand in this state.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		Run: func(cmd *cobra.Command, args []string) {
			OsExit(runImport(cmd, args))
		},
	}

	cmd.Flags().BoolP("dry-run", "n", false, "perform the trial update with no changes")
	cmd.Flags().BoolP("notify", "N", false, "enable desktop notifications")
	cmd.Flags().StringP("file", "f", "", "specify gup.json file path to import")
	if err := cmd.MarkFlagFilename("file", "json"); err != nil {
		panic(err)
	}
	cmd.Flags().IntP("jobs", "j", runtime.NumCPU(), "Specify the number of CPU cores to use")
	if err := cmd.RegisterFlagCompletionFunc("jobs", completeNCPUs); err != nil {
		panic(err)
	}

	return cmd
}

func runImport(cmd *cobra.Command, _ []string) int {
	dryRun, err := getFlagBool(cmd, "dry-run")
	if err != nil {
		print.Err(err)
		return 1
	}

	confFile, err := getFlagString(cmd, "file")
	if err != nil {
		print.Err(err)
		return 1
	}
	confFile = config.ResolveImportFilePath(confFile)

	notify, err := getFlagBool(cmd, "notify")
	if err != nil {
		print.Err(err)
		return 1
	}

	cpus, err := getFlagInt(cmd, "jobs")
	if err != nil {
		print.Err(err)
		return 1
	}
	if cpus < 1 {
		cpus = 1
	}

	if !fileutil.IsFile(confFile) {
		print.Err(fmt.Errorf("%s is not found", confFile))
		return 1
	}

	pkgs, err := config.ReadConfFile(confFile)
	if err != nil {
		print.Err(err)
		return 1
	}

	if len(pkgs) == 0 {
		print.Err("unable to import package: no package information")
		return 1
	}

	print.Info("start import based on " + confFile)
	return installFromConfig(pkgs, dryRun, notify, cpus)
}

func installFromConfig(pkgs []goutil.Package, dryRun, notification bool, cpus int) int {
	result := 0
	countFmt := "[%" + pkgDigit(pkgs) + "d/%" + pkgDigit(pkgs) + "d]"
	dryRunManager := goutil.NewGoPaths()

	signals := make(chan os.Signal, 1)
	if dryRun {
		if err := dryRunManager.StartDryRunMode(); err != nil {
			print.Err(fmt.Errorf("can not change to dry run mode: %w", err))
			return 1
		}
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
			syscall.SIGQUIT, syscall.SIGABRT)
		go catchSignal(signals, dryRunManager)
	}

	installer := func(_ context.Context, p goutil.Package) updateResult {
		ver, err := versionFromConfig(p)
		if err != nil {
			return updateResult{
				updated: false,
				pkg:     p,
				err:     fmt.Errorf("%s: %w", p.Name, err),
			}
		}
		if p.ImportPath == "" {
			return updateResult{
				updated: false,
				pkg:     p,
				err:     fmt.Errorf("%s: import path is empty", p.Name),
			}
		}

		if err := installByVersion(p.ImportPath, ver); err != nil {
			return updateResult{
				updated: false,
				pkg:     p,
				err:     fmt.Errorf("%s: %w", p.Name, err),
			}
		}

		return updateResult{
			updated: true,
			pkg:     p,
			err:     nil,
		}
	}

	ch := forEachPackage(context.Background(), pkgs, cpus, installer)

	count := 0
	for v := range ch {
		version, err := versionFromConfig(v.pkg)
		if err != nil {
			version = "unknown"
		}

		if v.err == nil {
			print.Info(fmt.Sprintf(countFmt+" %s@%s", count+1, len(pkgs), v.pkg.ImportPath, version))
		} else {
			result = 1
			print.Err(fmt.Errorf(countFmt+" %s", count+1, len(pkgs), v.err.Error()))
		}
		count++
		if count == len(pkgs) {
			break
		}
	}

	if dryRun {
		if err := dryRunManager.EndDryRunMode(); err != nil {
			print.Err(fmt.Errorf("can not change dry run mode to normal mode: %w", err))
			return 1
		}
		signal.Stop(signals)
		close(signals)
	}

	desktopNotifyIfNeeded(result, notification)
	return result
}

func versionFromConfig(pkg goutil.Package) (string, error) {
	if pkg.Version == nil {
		return "", errors.New("version is missing in gup.json")
	}
	ver := strings.TrimSpace(pkg.Version.Current)
	if ver == "" {
		return "", errors.New("version is empty in gup.json")
	}
	if ver == "(devel)" || ver == "devel" {
		return latestKeyword, nil
	}
	return ver, nil
}
