package sync

import (
	"bufio"
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"io"
	"log/slog"
)

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

// SyncDogu copies dogu volume data from a remote Cloudogu EcoSystem instance.
// rsync -avhz --delete -e "ssh -p 7000 -l ces-exporter -i /my-private-key" localhost:/data/ ./destination/
func (rs *RsyncSyncer) SyncDogu(_ context.Context, port int, source, destination string, excludePattern configuration.ExcludePattern, verbose bool) error {
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
	if excludePattern != "" {
		args = append(args, "--exclude="+excludePattern)
	}

	// ssh options
	args = append(args, "-e")
	args = append(args, fmt.Sprintf("ssh -p %d -l %s -i %s -o StrictHostKeyChecking=no -o BatchMode=yes", port, rs.user, rs.privateKeyPath))

	// source with host
	args = append(args, fmt.Sprintf("%s:%s", rs.host, source))

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
