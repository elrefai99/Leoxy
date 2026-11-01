package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	platform "github.com/elrefai99/Leoxy/app/pkg/core"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "ReStart the server in the background",
	Long:    "ReStart Leoxy in the background. Server output is appended to leoxy/log/leoxy.log, and the process ID is stored under leoxy/keys/.",
	Example: "  leoxy restart",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return fmt.Errorf("read PID file: %w", err)
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("invalid PID file")
		}
		if pid <= 1 || pid == os.Getpid() {
			return fmt.Errorf("invalid PID file")
		}

		if !platform.ProcessRunning(pid) || !platform.ProcessMatches(pid, "leoxy") {
			_ = os.Remove(pidFile)
			return fmt.Errorf("server is not running")
		}

		if err := platform.RequestStop(pid); err != nil {
			if forceErr := platform.ForceStop(pid); forceErr != nil {
				_ = os.Remove(pidFile)
				return fmt.Errorf("could not stop server with PID %d: %w", pid, forceErr)
			}
		}
		for i := 0; i < 20; i++ {
			if !platform.ProcessRunning(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if platform.ProcessRunning(pid) {
			if !platform.ProcessMatches(pid, "leoxy") {
				_ = os.Remove(pidFile)
				return fmt.Errorf("process with PID %d no longer matches Leoxy", pid)
			}
			if err := platform.ForceStop(pid); err != nil {
				_ = os.Remove(pidFile)
				return fmt.Errorf("could not stop server with PID %d: %w", pid, err)
			}
		}

		_ = os.Remove(pidFile)

		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate Leoxy executable: %w", err)
		}

		runArgs := []string{"run"}
		if configFile != "" {
			runArgs = append(runArgs, "--config", configFile)
		}
		if dataDir != "" {
			runArgs = append(runArgs, "--data-dir", dataDir)
		}
		process := exec.Command(executable, runArgs...)

		logFile, err := os.OpenFile("leoxy/log/leoxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("open Leoxy log file: %w", err)
		}
		defer func() {
			if err := logFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "close log file: %v\n", err)
			}
		}()

		process.Stdout = logFile
		process.Stderr = logFile
		detach(process)

		if err := process.Start(); err != nil {
			return fmt.Errorf("start Leoxy process: %w", err)
		}

		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(process.Process.Pid)), 0600); err != nil {
			_ = process.Process.Kill()
			return fmt.Errorf("write PID file: %w", err)
		}
		_ = process.Process.Release()

		fmt.Println("Leoxy started proxy")
		return nil
	},
}
