package sync

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
)

// RsyncSyncer allows copying data from a remote host.
type RsyncSyncer struct {
	host           string
	user           string
	privateKeyPath string
}

// NewRsyncSyncer creates a new RsyncSyncer instance.
func NewRsyncSyncer(host string, user string, privateKeyPath string) *RsyncSyncer {
	return &RsyncSyncer{
		host:           host,
		user:           user,
		privateKeyPath: privateKeyPath,
	}
}

// SyncDogu copies dogu volume data from a remote Cloudogu EcoSystem instance.
func (rs *RsyncSyncer) SyncDogu(_ context.Context, port, source, destination string) error {

	//rsync -avhz --delete -e "ssh -p 7000 -l ces-exporter -i /my-private-key" localhost:/data/ ./destination/

	sshOpts := fmt.Sprintf("ssh -p %s -l %s -i %s -o StrictHostKeyChecking=no -o BatchMode=yes", port, rs.user, rs.privateKeyPath)

	sourceWithHost := fmt.Sprintf("%s:%s", rs.host, source)

	// Define the rsync command and arguments
	cmd := exec.Command("rsync", "-avhz", "-e", sshOpts, sourceWithHost, destination)

	fmt.Println(cmd.String())

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

	fmt.Println("started rsync")

	// Create a channel to signal when output is complete
	done := make(chan struct{})

	// Function to read and print output in real-time
	go func() {
		fmt.Println("started stdout")
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Println("STDOUT:", scanner.Text()) // Print real-time stdout
		}
		done <- struct{}{}
	}()

	// Function to read and print errors in real-time
	go func() {
		fmt.Println("started stderror")
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Println("STDERR:", scanner.Text()) // Print real-time stderr
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
