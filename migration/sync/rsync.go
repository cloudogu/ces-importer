package sync

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"slices"
	"sync"
)

var (
	subDirsToIgnore = []string{"lost+found", "_private"}
)

type exportDoguApiClient interface {
	GetExportDogu(ctx context.Context) (*exporter.DoguExport, error)
	SetExportDogu(ctx context.Context, doguName string) (*exporter.DoguExport, error)
}

type systemInfoProvider interface {
	GetSystemInfo(ctx context.Context) (systemInfo *exporter.SystemInfo, err error)
}

// RsyncSyncer allows copying data from a remote host.
type RsyncSyncer struct {
	host                string
	user                string
	privateKeyPath      string
	makeCommand         commandMaker
	exportModeApiClient exportDoguApiClient
	systemInfoProvider  systemInfoProvider
	excludePattern      []configuration.ExcludePattern
	doguVolumeBasePath  string
	excludedDogus       []string
}

// NewRsyncSyncer creates a new RsyncSyncer instance.
func NewRsyncSyncer(host string, user string, privateKeyPath string, client exportDoguApiClient, provider systemInfoProvider, excludePattern []configuration.ExcludePattern, doguVolumeBasePath string, excludedDogus []string) *RsyncSyncer {
	commandMaker := func(name string, arg ...string) command {
		return exec.Command(name, arg...)
	}
	return &RsyncSyncer{
		host:                host,
		user:                user,
		privateKeyPath:      privateKeyPath,
		makeCommand:         commandMaker,
		exportModeApiClient: client,
		systemInfoProvider:  provider,
		excludePattern:      excludePattern,
		doguVolumeBasePath:  doguVolumeBasePath,
		excludedDogus:       excludedDogus,
	}
}

type commandMaker func(name string, arg ...string) command

type command interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	String() string
}

// SyncData gets the exporting systems system info and synchronizes the volume data of every dogu
// errors are collected and returned
func (rs *RsyncSyncer) SyncData(ctx context.Context) error {
	var result error

	systemInfo, err := rs.systemInfoProvider.GetSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch exporter system info: %w", err)
	}

	// map exclude patterns to dogu name for easy retrieval
	excludeMap := make(map[string]configuration.ExcludePattern)
	for _, p := range rs.excludePattern {
		excludeMap[p.DoguName] = p
	}

	// sync data for every dogu
	for _, dogu := range systemInfo.Dogus {
		isExcluded := slices.Contains(rs.excludedDogus, dogu.Name)
		if isExcluded {
			slog.Debug(fmt.Sprintf("Not syncing dogu %s as it is not available in k8s", dogu.Name))
			continue
		}
		slog.Info("Starting sync for dogu", "doguName", dogu.Name)
		doguName, err := doguCommons.QualifiedNameFromString(dogu.Name)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to get qualified dogu name from dogu %s: %w", dogu.Name, err))
			continue
		}
		// set the current dogu as export dogu in exporter
		doguExport, err := rs.exportModeApiClient.SetExportDogu(ctx, string(doguName.SimpleName))
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to set dogu %s as export dogu: %w", dogu.Name, err))
			continue
		}

		// exclude pattern might be an empty string
		excludePattern := excludeMap[dogu.Name]
		// default is /data/{doguName}

		doguImportDir := path.Join(rs.doguVolumeBasePath, string(doguName.SimpleName))
		subDirs, err := getSubDirs(doguImportDir)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to get subDirs for dogu %s: %w", dogu.Name, err))
			continue
		}

		for _, subDir := range subDirs {
			importerDestination := path.Join(doguImportDir, subDir)

			// Ensure the exporter path ends with a separator for rsync
			exporterSourcePath := path.Clean(path.Join(doguExport.VolumePath, subDir)) + "/"

			if err := rs.SyncDoguDir(ctx, doguExport.ExporterPort, exporterSourcePath, importerDestination, excludePattern, true); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSourcePath, importerDestination, err))
			}
		}

		slog.Info("Syncing for dogu successful", "doguName", dogu.Name)
	}
	return result
}

// SyncDoguDir copies dogu volume data from a remote Cloudogu EcoSystem instance.
func (rs *RsyncSyncer) SyncDoguDir(_ context.Context, port int, source, destination string, exclude configuration.ExcludePattern, verbose bool) error {

	// Define the rsync command and arguments
	args := rs.buildRSyncArgs(port, source, destination, exclude, verbose)
	cmd := rs.makeCommand("rsync", args...)

	slog.Debug(fmt.Sprintf("executing rsync command: %s", cmd.String()))

	// Get stdout and stderr pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	// Start the rsync process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting rsync: %w", err)
	}

	slog.Info("started rsync")

	// Create a channel to signal when output is complete
	var wg sync.WaitGroup
	// waitGroup for 2 output channels
	wg.Add(2)
	// Function to read and print output in real-time
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			slog.Info(scanner.Text()) // Print real-time stdout
		}
		wg.Done()
	}()

	// Function to read and print errors in real-time
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			slog.Error(scanner.Text()) // Print real-time stderr
		}
		wg.Done()
	}()

	// Wait for both output streams to complete
	wg.Wait()
	// Wait for rsync to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("rsync exited with error: %w", err)
	}
	return nil
}

// buildRSyncArgs builds the arguments for the rsync command based on the given parameters
func (rs *RsyncSyncer) buildRSyncArgs(port int, source, destination string, exclude configuration.ExcludePattern, verbose bool) []string {
	var args []string
	// archive mode
	// verbose
	// human-readable sizes
	// compress file data during transfer
	if verbose {
		args = append(args, "-avhz")
	} else {
		args = append(args, "-ahz")
	}

	// delete extraneous files from dest dirs
	args = append(args, "--delete")

	// turn sequences of nulls into sparse blocks
	args = append(args, "--sparse")

	// print some file-transfer stats
	args = append(args, "--stats")

	// exclude pattern
	if exclude.Pattern != "" {
		args = append(args, "--exclude="+exclude.Pattern)
	}

	// ssh options
	args = append(args, "-e")
	args = append(args, fmt.Sprintf("ssh -p %d -l %s -i %s -o StrictHostKeyChecking=no -o BatchMode=yes", port, rs.user, rs.privateKeyPath))

	// source with host
	args = append(args, fmt.Sprintf("%s:%s", rs.host, source))

	// destination path
	args = append(args, destination)
	return args
}

func getSubDirs(importDestDir string) ([]string, error) {
	result := make([]string, 0)

	entries, err := os.ReadDir(importDestDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return result, fmt.Errorf("error reading dest-dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && !slices.Contains(subDirsToIgnore, entry.Name()) {
			result = append(result, entry.Name())
		}
	}

	slog.Debug("found subdirs for", "dir", importDestDir, "subdirs", result)

	return result, nil
}
