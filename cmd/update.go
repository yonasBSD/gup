package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/nao1215/gup/internal/config"
	"github.com/nao1215/gup/internal/fileutil"
	"github.com/nao1215/gup/internal/goutil"
	"github.com/nao1215/gup/internal/notify"
	"github.com/nao1215/gup/internal/print"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var (
	getLatestVer        = goutil.GetLatestVer             //nolint:gochecknoglobals // swapped in tests
	installLatest       = goutil.InstallLatest            //nolint:gochecknoglobals // swapped in tests
	installMainOrMaster = goutil.InstallMainOrMaster      //nolint:gochecknoglobals // swapped in tests
	installByVersionUpd = goutil.Install                  //nolint:gochecknoglobals // swapped in tests
	detectModulePathErr = goutil.DetectModulePathMismatch //nolint:gochecknoglobals // swapped in tests
)

const latestKeyword = "latest"

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update binaries installed by 'go install'",
		Long: `Update binaries installed by 'go install'

If you execute '$ gup update', gup gets the package path of all commands
under $GOPATH/bin and automatically updates commands to the latest version,
using the current installed Go toolchain.`,
		Run: func(cmd *cobra.Command, args []string) {
			OsExit(gup(cmd, args))
		},
		ValidArgsFunction: completePathBinaries,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "perform the trial update with no changes")
	cmd.Flags().BoolP("notify", "N", false, "enable desktop notifications")
	cmd.Flags().StringSliceP("exclude", "e", []string{}, "specify binaries which should not be updated (delimiter: ',')")
	if err := cmd.RegisterFlagCompletionFunc("exclude", completePathBinaries); err != nil {
		panic(err)
	}
	cmd.Flags().StringSliceP("main", "m", []string{}, "specify binaries which update by @main or @master (delimiter: ',')")
	if err := cmd.RegisterFlagCompletionFunc("main", completePathBinaries); err != nil {
		panic(err)
	}
	cmd.Flags().StringSlice("master", []string{}, "specify binaries which update by @master (delimiter: ',')")
	if err := cmd.RegisterFlagCompletionFunc("master", completePathBinaries); err != nil {
		panic(err)
	}
	cmd.Flags().StringSlice(latestKeyword, []string{}, "specify binaries which update by @latest (delimiter: ',')")
	if err := cmd.RegisterFlagCompletionFunc(latestKeyword, completePathBinaries); err != nil {
		panic(err)
	}
	// cmd.Flags().BoolP("main-all", "M", false, "update all binaries by @main or @master (delimiter: ',')")
	cmd.Flags().IntP("jobs", "j", runtime.NumCPU(), "Specify the number of CPU cores to use")
	if err := cmd.RegisterFlagCompletionFunc("jobs", completeNCPUs); err != nil {
		panic(err)
	}
	cmd.Flags().Bool("ignore-go-update", false, "Ignore updates to the Go toolchain")

	return cmd
}

// gup is main sequence.
// All errors are handled in this function.
func gup(cmd *cobra.Command, args []string) int {
	if err := goutil.CanUseGoCmd(); err != nil {
		print.Err(fmt.Errorf("%s: %w", "you didn't install golang", err))
		return 1
	}

	dryRun, err := getFlagBool(cmd, "dry-run")
	if err != nil {
		print.Err(err)
		return 1
	}

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

	ignoreGoUpdate, err := getFlagBool(cmd, "ignore-go-update")
	if err != nil {
		print.Err(err)
		return 1
	}

	pkgs, err := getPackageInfo()
	if err != nil {
		print.Err(err)
		return 1
	}

	excludePkgList, err := getFlagStringSlice(cmd, "exclude")
	if err != nil {
		print.Err(err)
		return 1
	}

	mainPkgNames, err := getFlagStringSlice(cmd, "main")
	if err != nil {
		print.Err(err)
		return 1
	}
	masterPkgNames, err := getFlagStringSlice(cmd, "master")
	if err != nil {
		print.Err(err)
		return 1
	}
	latestPkgNames, err := getFlagStringSlice(cmd, latestKeyword)
	if err != nil {
		print.Err(err)
		return 1
	}

	pkgs = extractUserSpecifyPkg(pkgs, args)
	pkgs = excludePkgs(excludePkgList, pkgs)

	if len(pkgs) == 0 {
		print.Err("unable to update package: no package information or no package under $GOBIN")
		return 1
	}

	confReadPath := config.ResolveImportFilePath("")
	confWritePath := config.FilePath()
	if fileutil.IsFile(confReadPath) {
		confWritePath = confReadPath
	}

	confPkgs := readConfFileIfExists(confReadPath)

	channelMap, err := resolveUpdateChannels(pkgs, confPkgs, mainPkgNames, masterPkgNames, latestPkgNames)
	if err != nil {
		print.Err(err)
		return 1
	}

	result, succeededPkgs := updateWithChannels(pkgs, dryRun, notify, cpus, ignoreGoUpdate, channelMap)

	if !dryRun && shouldPersistChannels(confPkgs, mainPkgNames, masterPkgNames, latestPkgNames) {
		merged := mergeConfigPackages(confPkgs, succeededPkgs, channelMap)
		if err := writeConfFilePath(confWritePath, merged); err != nil {
			print.Warn("failed to write " + confWritePath + ": " + err.Error())
		}
	}

	return result
}

func excludePkgs(excludePkgList []string, pkgs []goutil.Package) []goutil.Package {
	packageList := []goutil.Package{}
	for _, v := range pkgs {
		if slices.Contains(excludePkgList, v.Name) {
			print.Info(fmt.Sprintf("Exclude '%s' from the update target", v.Name))
			continue
		}
		packageList = append(packageList, v)
	}
	return packageList
}

type updateResult struct {
	updated bool
	pkg     goutil.Package
	err     error
}

// update updates all packages.
// If dryRun is true, it does not update.
// If notification is true, it notifies the result of update.
func update(pkgs []goutil.Package, dryRun, notification bool, cpus int, ignoreGoUpdate bool, mainPkgNames []string) int {
	channelMap := make(map[string]goutil.UpdateChannel, len(pkgs))
	for _, p := range pkgs {
		channelMap[p.Name] = goutil.UpdateChannelLatest
	}
	for _, name := range mainPkgNames {
		channelMap[strings.TrimSpace(name)] = goutil.UpdateChannelMain
	}

	result, _ := updateWithChannels(pkgs, dryRun, notification, cpus, ignoreGoUpdate, channelMap)
	return result
}

func updateWithChannels(pkgs []goutil.Package, dryRun, notification bool, cpus int, ignoreGoUpdate bool, channelMap map[string]goutil.UpdateChannel) (int, []goutil.Package) {
	result := 0
	countFmt := "[%" + pkgDigit(pkgs) + "d/%" + pkgDigit(pkgs) + "d]"
	dryRunManager := goutil.NewGoPaths()
	succeededPkgs := make([]goutil.Package, 0, len(pkgs))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	print.Info("update binary under $GOPATH/bin or $GOBIN")
	signals := make(chan os.Signal, 1)
	if dryRun {
		if err := dryRunManager.StartDryRunMode(); err != nil {
			print.Err(fmt.Errorf("can not change to dry run mode: %w", err))
			notify.Warn("gup", "Can not change to dry run mode")
			return 1, nil
		}
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
			syscall.SIGQUIT, syscall.SIGABRT)
		go catchSignal(signals, cancel)
	}

	updater := func(_ context.Context, p goutil.Package) updateResult {
		// Collect online latest version if possible; else always update
		shouldUpdate := true
		modulePathChanged := false
		if p.ModulePath != "" {
			ver, err := getLatestVer(p.ModulePath)
			if err != nil {
				newPkg, changed := resolveModulePathChange(p, err)
				if !changed {
					return updateResult{
						updated: false,
						pkg:     p,
						err:     fmt.Errorf("%s: %w", p.Name, err),
					}
				}
				modulePathChanged = true
				p = newPkg

				ver, err = getLatestVer(p.ModulePath)
				if err != nil {
					return updateResult{
						updated: false,
						pkg:     p,
						err:     fmt.Errorf("%s: %w", p.Name, err),
					}
				}
			}
			p.Version.Latest = ver

			// Check if we should update the package
			shouldUpdate = modulePathChanged || !p.IsPackageUpToDate() || (!ignoreGoUpdate && !p.IsGoUpToDate())
		}

		if !shouldUpdate {
			return updateResult{
				updated: false,
				pkg:     p,
				err:     nil,
			}
		}

		// Run the update
		var updateErr error
		if p.ImportPath == "" {
			updateErr = fmt.Errorf("%s is not installed by 'go install' (or permission incorrect)", p.Name)
		} else {
			originalName := p.Name
			channel := packageUpdateChannel(p.Name, p.UpdateChannel, channelMap)
			p.UpdateChannel = channel

			if err := installWithSelectedVersion(p.ImportPath, channel); err != nil {
				newPkg, changed := resolveModulePathChange(p, err)
				if !changed {
					updateErr = fmt.Errorf("%s: %w", p.Name, err)
				} else {
					p = newPkg
					if retryErr := installWithSelectedVersion(p.ImportPath, channel); retryErr != nil {
						updateErr = fmt.Errorf("%s: %w", originalName, retryErr)
					} else {
						newName := binaryNameFromImportPath(p.ImportPath)
						if err := removeOldBinaryIfRenamed(originalName, newName); err != nil {
							updateErr = fmt.Errorf("%s: %w", originalName, err)
						}
						p.Name = newName
						p.UpdateChannel = channel
					}
				}
			}
		}

		if updateErr == nil {
			p.SetLatestVer()
		}
		return updateResult{
			updated: updateErr == nil,
			pkg:     p,
			err:     updateErr,
		}
	}

	// update all packages
	ch := forEachPackage(ctx, pkgs, cpus, updater)

	// print result
	count := 0
	for v := range ch {
		if v.err == nil {
			print.Info(fmt.Sprintf(countFmt+" %s (%s)",
				count+1, len(pkgs), v.pkg.ImportPath, v.pkg.CurrentToLatestStr()))
			succeededPkgs = append(succeededPkgs, v.pkg)
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
			return 1, nil
		}
		signal.Stop(signals) // stop signal delivery before closing the channel
		close(signals)       // unblock catchSignal goroutine
	}

	desktopNotifyIfNeeded(result, notification)

	return result, succeededPkgs
}

func desktopNotifyIfNeeded(result int, enable bool) {
	if enable {
		if result == 0 {
			notify.Info("gup", "All update success")
		} else {
			notify.Warn("gup", "Some package can't update")
		}
	}
}

func catchSignal(c <-chan os.Signal, cancel context.CancelFunc) {
	if _, ok := <-c; ok {
		cancel()
	}
}

func installWithSelectedVersion(importPath string, channel goutil.UpdateChannel) error {
	switch goutil.NormalizeUpdateChannel(string(channel)) {
	case goutil.UpdateChannelLatest:
		return installLatest(importPath)
	case goutil.UpdateChannelMain:
		return installMainOrMaster(importPath)
	case goutil.UpdateChannelMaster:
		return installByVersionUpd(importPath, "master")
	default:
		return installLatest(importPath)
	}
}

func resolveModulePathChange(pkg goutil.Package, err error) (goutil.Package, bool) {
	declaredPath, requiredPath, ok := detectModulePathErr(err)
	if !ok {
		return pkg, false
	}

	pkg.ImportPath = replaceImportPathPrefix(pkg.ImportPath, requiredPath, declaredPath)
	pkg.ModulePath = declaredPath
	return pkg, true
}

func replaceImportPathPrefix(importPath, oldModulePath, newModulePath string) string {
	switch {
	case importPath == "":
		return newModulePath
	case importPath == oldModulePath:
		return newModulePath
	case strings.HasPrefix(importPath, oldModulePath+"/"):
		return newModulePath + strings.TrimPrefix(importPath, oldModulePath)
	default:
		return newModulePath
	}
}

func removeOldBinaryIfRenamed(oldName, newName string) error {
	if oldName == "" || newName == "" || oldName == newName {
		return nil
	}

	goBin, err := goutil.GoBin()
	if err != nil {
		return fmt.Errorf("can't find installed binaries: %w", err)
	}

	oldBinaryPath := filepath.Join(goBin, oldName)
	if _, err := os.Stat(oldBinaryPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("can't stat old binary %s: %w", oldBinaryPath, err)
	}

	if err := os.Remove(oldBinaryPath); err != nil {
		return fmt.Errorf("can't remove old binary %s: %w", oldBinaryPath, err)
	}
	return nil
}

func binaryNameFromImportPath(importPath string) string {
	binName := filepath.Base(importPath)
	if runtime.GOOS == goosWindows {
		goExe := os.Getenv("GOEXE")
		if goExe != "" && !strings.HasSuffix(strings.ToLower(binName), strings.ToLower(goExe)) {
			return binName + goExe
		}
	}
	return binName
}

func packageUpdateChannel(name string, fallback goutil.UpdateChannel, channelMap map[string]goutil.UpdateChannel) goutil.UpdateChannel {
	if channel, ok := channelMap[name]; ok {
		return goutil.NormalizeUpdateChannel(string(channel))
	}
	return goutil.NormalizeUpdateChannel(string(fallback))
}

func readConfFileIfExists(path string) []goutil.Package {
	if !fileutil.IsFile(path) {
		return []goutil.Package{}
	}
	pkgs, err := config.ReadConfFile(path)
	if err != nil {
		// Ignore non-JSON / legacy config content and treat as no config.
		return []goutil.Package{}
	}
	return pkgs
}

func shouldPersistChannels(_ []goutil.Package, mainPkgNames, masterPkgNames, latestPkgNames []string) bool {
	return len(mainPkgNames) > 0 || len(masterPkgNames) > 0 || len(latestPkgNames) > 0
}

func resolveUpdateChannels(
	pkgs []goutil.Package,
	confPkgs []goutil.Package,
	mainPkgNames []string,
	masterPkgNames []string,
	latestPkgNames []string,
) (map[string]goutil.UpdateChannel, error) {
	channelMap := make(map[string]goutil.UpdateChannel, len(pkgs))
	exists := make(map[string]struct{}, len(pkgs))
	for _, p := range pkgs {
		channelMap[p.Name] = goutil.UpdateChannelLatest
		exists[p.Name] = struct{}{}
	}
	for _, p := range confPkgs {
		if _, ok := exists[p.Name]; ok {
			channelMap[p.Name] = goutil.NormalizeUpdateChannel(string(p.UpdateChannel))
		}
	}

	assignedByFlag := map[string]string{}
	apply := func(flag string, names []string, channel goutil.UpdateChannel) error {
		for _, raw := range names {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}
			if prevFlag, ok := assignedByFlag[name]; ok && prevFlag != flag {
				return fmt.Errorf("same binary (%s) is specified in both --%s and --%s", name, prevFlag, flag)
			}
			assignedByFlag[name] = flag

			if _, ok := exists[name]; !ok {
				print.Warn("not found '" + name + "' package in update target")
				continue
			}
			channelMap[name] = channel
		}
		return nil
	}

	if err := apply("main", mainPkgNames, goutil.UpdateChannelMain); err != nil {
		return nil, err
	}
	if err := apply("master", masterPkgNames, goutil.UpdateChannelMaster); err != nil {
		return nil, err
	}
	if err := apply(latestKeyword, latestPkgNames, goutil.UpdateChannelLatest); err != nil {
		return nil, err
	}
	return channelMap, nil
}

func mergeConfigPackages(confPkgs []goutil.Package, succeededPkgs []goutil.Package, channelMap map[string]goutil.UpdateChannel) []goutil.Package {
	pkgByName := map[string]goutil.Package{}
	for _, p := range confPkgs {
		pkgByName[p.Name] = sanitizeConfigPackage(p)
	}
	for _, p := range succeededPkgs {
		if p.Name == "" || p.ImportPath == "" {
			continue
		}
		channel := packageUpdateChannel(p.Name, p.UpdateChannel, channelMap)
		pkgByName[p.Name] = goutil.Package{
			Name:          p.Name,
			ImportPath:    p.ImportPath,
			Version:       &goutil.Version{Current: persistedVersion(p)},
			UpdateChannel: channel,
		}
	}
	for name, channel := range channelMap {
		p, ok := pkgByName[name]
		if !ok {
			continue
		}
		p.UpdateChannel = goutil.NormalizeUpdateChannel(string(channel))
		pkgByName[name] = sanitizeConfigPackage(p)
	}

	names := make([]string, 0, len(pkgByName))
	for name := range pkgByName {
		names = append(names, name)
	}
	sort.Strings(names)

	merged := make([]goutil.Package, 0, len(names))
	for _, name := range names {
		merged = append(merged, pkgByName[name])
	}
	return merged
}

func sanitizeConfigPackage(p goutil.Package) goutil.Package {
	version := latestKeyword
	if p.Version != nil {
		v := strings.TrimSpace(p.Version.Current)
		if v != "" {
			version = v
		}
	}

	return goutil.Package{
		Name:          strings.TrimSpace(p.Name),
		ImportPath:    strings.TrimSpace(p.ImportPath),
		Version:       &goutil.Version{Current: version},
		UpdateChannel: goutil.NormalizeUpdateChannel(string(p.UpdateChannel)),
	}
}

func persistedVersion(p goutil.Package) string {
	if p.Version == nil {
		return latestKeyword
	}
	if latest := strings.TrimSpace(p.Version.Latest); latest != "" && latest != "unknown" {
		return latest
	}
	if current := strings.TrimSpace(p.Version.Current); current != "" {
		return current
	}
	return latestKeyword
}

func writeConfFilePath(path string, pkgs []goutil.Package) (err error) {
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
	return nil
}

func pkgDigit(pkgs []goutil.Package) string {
	return strconv.Itoa(len(strconv.Itoa(len(pkgs))))
}

func getBinaryPathList() ([]string, error) {
	goBin, err := goutil.GoBin()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "can't find installed binaries", err)
	}

	binList, err := goutil.BinaryPathList(goBin)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "can't get binary-paths installed by 'go install'", err)
	}

	return binList, nil
}

func getPackageInfo() ([]goutil.Package, error) {
	binList, err := getBinaryPathList()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "can't get package info", err)
	}

	return goutil.GetPackageInformation(binList), nil
}

func extractUserSpecifyPkg(pkgs []goutil.Package, targets []string) []goutil.Package {
	result := []goutil.Package{}
	tmp := []string{}
	if len(targets) == 0 {
		return pkgs
	}

	if runtime.GOOS == "windows" {
		for i, target := range targets {
			if strings.HasSuffix(strings.ToLower(target), ".exe") {
				targets[i] = strings.TrimSuffix(strings.ToLower(target), ".exe")
			}
		}
	}

	for _, v := range pkgs {
		pkg := v.Name
		if runtime.GOOS == "windows" {
			if strings.HasSuffix(strings.ToLower(pkg), ".exe") {
				pkg = strings.TrimSuffix(strings.ToLower(pkg), ".exe")
			}
		}
		if slices.Contains(targets, pkg) {
			result = append(result, v)
			tmp = append(tmp, pkg)
		}
	}

	if len(tmp) != len(targets) {
		for _, target := range targets {
			if !slices.Contains(tmp, target) {
				print.Warn("not found '" + target + "' package in $GOPATH/bin or $GOBIN")
			}
		}
	}
	return result
}

func completePathBinaries(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	binList, _ := getBinaryPathList()
	for i, b := range binList {
		binList[i] = filepath.Base(b)
	}
	return binList, cobra.ShellCompDirectiveNoFileComp
}
