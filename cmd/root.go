// Package cmd define subcommands provided by the gup command
package cmd

import (
	"os"
	"runtime"
	"strconv"

	"github.com/nao1215/gup/internal/assets"
	"github.com/nao1215/gup/internal/completion"
	"github.com/spf13/cobra"
)

// OsExit is wrapper for  os.Exit(). It's for unit test.
var OsExit = os.Exit

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "gup",
		Short: `gup command update binaries installed by 'go install'.
If you update all binaries, just run '$ gup update'`,
		Long: `gup command update binaries installed by "go install" to the latest version.

gup updates all binaries in parallel, so it is very fast. It also provides
subcommands for manipulating binaries under $GOPATH/bin ($GOBIN).
gup is a cross-platform software that runs on Windows, Mac and Linux.

If you are using oh-my-zsh, then gup has an alias set up. The alias
is gup - git pull --rebase. Therefore, please make sure that the
oh-my-zsh alias is disabled (e.g. $ \gup update).
`,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newBugReportCmd())

	if !completion.IsWindows() {
		cmd.AddCommand(newManCmd())
	}

	return cmd
}

// Execute run gup process.
func Execute() error {
	assets.DeployIconIfNeeded()
	rootCmd := newRootCmd()
	return rootCmd.Execute()
}

// completeNCPUs returns the number of CPU cores as a string.
func completeNCPUs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	n := runtime.NumCPU()
	ret := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		ret = append(ret, strconv.FormatInt(int64(i), 10))
	}
	return ret, cobra.ShellCompDirectiveNoFileComp
}
