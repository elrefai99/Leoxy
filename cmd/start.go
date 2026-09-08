package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

const pidFile = ".leoxy.pid"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server in the background",
	RunE: func(cmd *cobra.Command, args []string) error {
		executable, err := os.Executable()
		if err != nil {
			return err
		}

		process := exec.Command(executable, "run")

		logFile, err := os.OpenFile(
			".leoxy.log",
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			return err
		}
		defer logFile.Close()

		process.Stdout = logFile
		process.Stderr = logFile

		if err := process.Start(); err != nil {
			return err
		}

		err = os.WriteFile(
			pidFile,
			[]byte(strconv.Itoa(process.Process.Pid)),
			0644,
		)
		if err != nil {
			_ = process.Process.Kill()
			return err
		}

		fmt.Printf("Leoxy started with PID %d\n", process.Process.Pid)
		return nil
	},
}
