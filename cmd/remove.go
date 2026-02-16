package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "Remove the binary under $GOPATH/bin or $GOBIN",
		Long: `Remove command in $GOPATH/bin or $GOBIN.
If you want to specify multiple binaries at once, separate them with space.
[e.g.] gup remove a_cmd b_cmd c_cmd`,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completePathBinaries,
		Run: func(cmd *cobra.Command, args []string) {
			OsExit(remove(cmd, args))
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Forcibly remove the file")

	return cmd
}

func remove(cmd *cobra.Command, args []string) int {
	if len(args) == 0 {
		print.Err("no command name specified")
		return 1
	}

	force, err := getFlagBool(cmd, "force")
	if err != nil {
		print.Err(err)
		return 1
	}

	gobin, err := goutil.GoBin()
	if err != nil {
		print.Err(err)
		return 1
	}

	return removeLoop(gobin, force, args)
}

const goosWindows = "windows"

// GOOS is wrapper for runtime.GOOS variable. It's for unit test.
var GOOS = runtime.GOOS //nolint:gochecknoglobals

func removeLoop(gobin string, force bool, target []string) int {
	result := 0
	for _, v := range target {
		orig := v
		// In Windows, $GOEXE is set to the ".exe" extension.
		// The user-specified command name (arguments) may not have an extension.
		execSuffix := normalizeExecSuffix(GOOS, os.Getenv("GOEXE"))
		if GOOS == goosWindows && !strings.HasSuffix(v, execSuffix) {
			v += execSuffix
		}
		if !isSafeBinaryName(v) {
			print.Err(fmt.Errorf("invalid command name: %s", orig))
			result = 1
			continue
		}

		target := filepath.Join(gobin, v)
		if !fileutil.IsFile(target) {
			print.Err(fmt.Errorf("no such file or directory: %s", target))
			result = 1
			continue
		}
		if !force {
			if !print.Question(fmt.Sprintf("remove %s?", target)) {
				print.Info("cancel removal " + target)
				continue
			}
		}

		if err := os.Remove(target); err != nil {
			print.Err(err)
			result = 1
			continue
		}
		print.Info("removed " + target)
	}
	return result
}

func normalizeExecSuffix(goos, goExe string) string {
	if goos != goosWindows {
		return goExe
	}

	goExe = strings.TrimSpace(goExe)
	if goExe == "" {
		return ".exe"
	}
	return goExe
}

func isSafeBinaryName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	if filepath.IsAbs(name) {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	return true
}
