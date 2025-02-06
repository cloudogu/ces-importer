package rsync

import (
	"bufio"
	"fmt"
	"os/exec"
)

func Sync(source string, destination string) error {

	//rsync -avhz --delete -e "ssh -p 7000 -l ces-exporter -i /my-private-key" localhost:/data/ ./destination/

	sshOpts := "ssh -p 7000 -l ces-exporter -i /home/bernst/.ssh/ces_migrate_key -o StrictHostKeyChecking=no -o BatchMode=yes"

	// Define the rsync command and arguments
	cmd := exec.Command("rsync", "-avhz", "-e", sshOpts, source, destination)

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
