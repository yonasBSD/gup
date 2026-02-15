package cmd

import (
	"fmt"
	"os"

	"github.com/nao1215/gup/internal/completion"
	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completions (bash, fish, zsh) for gup",
		Long: `Generate shell completions (bash, fish, zsh) for the gup command.
With a shell name as argument, output completion for the shell to standard output.
Use --install to write completion files to the user shell config paths.`,
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "fish", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := newRootCmd()
			install, err := getFlagBool(cmd, "install")
			if err != nil {
				return err
			}
			if install {
				if len(args) != 0 {
					return fmt.Errorf("--install cannot be used with shell argument")
				}
				completion.DeployShellCompletionFileIfNeeded(rootCmd)
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("specify shell (bash|fish|zsh) or use --install to write completion files")
			}
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, false)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, false)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			default:
				return fmt.Errorf("internal error, should not happen with arg %q", args[0])
			}
		},
	}
	cmd.Flags().Bool("install", false, "install completion files to the user shell config paths")
	return cmd
}
