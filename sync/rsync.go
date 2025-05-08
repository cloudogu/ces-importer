package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/configuration"
	"io"
	"log/slog"
)

var hostProtocolScheme = "https://"

type ApiCli interface {
	DoPostRequest(ctx context.Context, exporterUrl string, body io.Reader, pathParams []string) (result []byte, err error)
	DoGetRequest(ctx context.Context, exporterUrl string) (result []byte, err error)
}

// RsyncSyncer allows copying data from a remote host.
type RsyncSyncer struct {
	host           string
	user           string
	privateKeyPath string
	makeCommand    commandMaker
}

// NewRsyncSyncer creates a new RsyncSyncer instance.
func NewRsyncSyncer(host string, user string, privateKeyPath string, makeCommand commandMaker) *RsyncSyncer {
	return &RsyncSyncer{
		host:           host,
		user:           user,
		privateKeyPath: privateKeyPath,
		makeCommand:    makeCommand,
	}
}

type commandMaker func(name string, arg ...string) Command

type Command interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	String() string
}

// SyncData gets the exporting systems system info and synchronizes the volume data of every dogu
// errors are collected and returned
func (rs *RsyncSyncer) SyncData(ctx context.Context, apiCli ApiCli, config configuration.Job) error {
	var result error

	systemInfo, err := fetchExporterSystemInfo(ctx, config.ExporterHost, apiCli)
	if err != nil {
		return err
	}

	// map exclude patterns to dogu name for easy retrieval
	excludeMap := make(map[string]configuration.Exclude)
	for _, p := range config.Exclude {
		excludeMap[p.DoguName] = p
	}

	// sync data for every dogu
	for _, dogu := range systemInfo.Dogus {
		slog.Info("Starting sync for dogu ", "doguName", dogu.Name)

		// set the current dogu as export dogu in exporter
		pathParams := []string{
			dogu.Name,
		}
		exporterUrl := hostProtocolScheme + config.ExporterHost + exporter.EndpointExportDogu
		doguExportResult, err := apiCli.DoPostRequest(ctx, exporterUrl, nil, pathParams)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to set dogu %s as export dogu: %w", dogu.Name, err))
			continue
		}

		var doguExport exporter.DoguExport
		err = json.Unmarshal(doguExportResult, &doguExport)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to parse dogu export response: %q: %w", doguExportResult, err))
			continue
		}

		// exclude pattern might be an empty string
		excludePattern := excludeMap[dogu.Name]
		// default is /data/{doguName}
		importerDestination := config.DoguVolumeBasePath + dogu.Name

		if err := rs.SyncDogu(ctx, doguExport.ExporterPort, doguExport.VolumePath, importerDestination, excludePattern, true); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to sync source %s to destination %s: %w", doguExport.VolumePath, importerDestination, err))
		}

		slog.Info("Syncing for dogu successful", "doguName", dogu.Name)
	}

	return result
}

// SyncDogu copies dogu volume data from a remote Cloudogu EcoSystem instance.
func (rs *RsyncSyncer) SyncDogu(_ context.Context, port int, source, destination string, exclude configuration.Exclude, verbose bool) error {
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

	// Define the rsync command and arguments
	cmd := rs.makeCommand("rsync", args...)

	slog.Info(fmt.Sprintf("executing rsync command: %s", cmd.String()))

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
	done := make(chan struct{})

	// Function to read and print output in real-time
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			slog.Info(scanner.Text()) // Print real-time stdout
		}
		done <- struct{}{}
	}()

	// Function to read and print errors in real-time
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			slog.Error(scanner.Text()) // Print real-time stderr
		}
		done <- struct{}{}
	}()

	// Wait for both output streams to complete
	<-done
	<-done

	// Wait for rsync to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("rsync exited with error: %w", err)
	}

	return nil
}

// fetchExporterSystemInfo gets the exporting systems system info
func fetchExporterSystemInfo(ctx context.Context, hostname string, apiCli ApiCli) (*exporter.SystemInfo, error) {
	exporterUrl := hostProtocolScheme + hostname + exporter.EndpointSystemInfo

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exporter system info: %w", err)
	}

	var systemInfo exporter.SystemInfo
	err = json.Unmarshal(result, &systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return &systemInfo, nil
}
