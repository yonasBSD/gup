package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the binary names under $GOPATH/bin and their path info. to gup.json.",
		Long: `Export the binary names under $GOPATH/bin and their path info. to gup.json.
	
Use the export subcommand if you want to install the same golang
binaries across multiple systems. By default, this sub-command 
exports the file to $XDG_CONFIG_HOME/.config/gup/gup.json (e.g. $HOME/.config/gup/gup.json.) 
After you have placed gup.json in the same path hierarchy on
another system, you execute import subcommand. gup start the
installation according to the contents of gup.json.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		Run: func(cmd *cobra.Command, args []string) {
			OsExit(export(cmd, args))
		},
	}
	cmd.Flags().BoolP("output", "o", false, "print command path information at STDOUT")
	cmd.Flags().StringP("file", "f", "", "specify gup.json file path to export")
	if err := cmd.MarkFlagFilename("file", "json"); err != nil {
		panic(err)
	}

	return cmd
}

func export(cmd *cobra.Command, _ []string) int {
	if err := goutil.CanUseGoCmd(); err != nil {
		print.Err(fmt.Errorf("%s: %w", "you didn't install golang", err))
		return 1
	}

	output, err := getFlagBool(cmd, "output")
	if err != nil {
		print.Err(err)
		return 1
	}
	configPath, err := getFlagString(cmd, "file")
	if err != nil {
		print.Err(err)
		return 1
	}
	configPath = config.ResolveExportFilePath(configPath)

	pkgs, err := getPackageInfo()
	if err != nil {
		print.Err(err)
		return 1
	}
	pkgs = validPkgInfo(pkgs)
	confPkgs, err := readConfFileIfExists(configPath)
	if err != nil {
		print.Warn("failed to read " + configPath + ": " + err.Error())
		confPkgs = []goutil.Package{}
	}
	pkgs = applySavedChannels(pkgs, confPkgs)

	if len(pkgs) == 0 {
		print.Err("no package information")
		return 1
	}

	if output {
		err = outputConfig(pkgs)
	} else {
		err = writeConfigFile(configPath, pkgs)
	}
	if err != nil {
		print.Err(err)
		return 1
	}
	return 0
}

func writeConfigFile(path string, pkgs []goutil.Package) (err error) {
	path = filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(path), fileutil.FileModeCreatingDir); err != nil {
		return fmt.Errorf("%s: %w", "can not make config directory", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%s %s: %w", "can't update", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err = config.WriteConfFile(file, pkgs); err != nil {
		return err
	}
	print.Info("Export " + path)
	return nil
}

func outputConfig(pkgs []goutil.Package) error {
	return config.WriteConfFile(print.Stdout, pkgs)
}

func validPkgInfo(pkgs []goutil.Package) []goutil.Package {
	result := []goutil.Package{}
	for _, v := range pkgs {
		if v.ImportPath == "" {
			print.Warn("can't get '" + v.Name + "' package path information. old go version binary")
			continue
		}
		result = append(result, goutil.Package{Name: v.Name, ImportPath: v.ImportPath, Version: v.Version})
	}
	return result
}

func applySavedChannels(pkgs, confPkgs []goutil.Package) []goutil.Package {
	channelByName := make(map[string]goutil.UpdateChannel, len(confPkgs))
	for _, p := range confPkgs {
		channelByName[p.Name] = goutil.NormalizeUpdateChannel(string(p.UpdateChannel))
	}

	result := make([]goutil.Package, 0, len(pkgs))
	for _, p := range pkgs {
		channel, ok := channelByName[p.Name]
		if !ok {
			channel = goutil.UpdateChannelLatest
		}
		p.UpdateChannel = channel
		result = append(result, p)
	}
	return result
}
