package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return fmt.Errorf("server is not running")
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("invalid PID file")
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}

		if err := process.Kill(); err != nil {
			return fmt.Errorf("could not stop server: %w", err)
		}

		_ = os.Remove(pidFile)

		fmt.Printf("Leoxy stopped with PID %d\n", pid)
		return nil
	},
}
