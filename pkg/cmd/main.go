package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "v2.5.7"

var rootCmd = &cobra.Command{
	Use:     "leoxy",
	Short:   "Leoxy reverse proxy service",
	Long:    "Leoxy is an HTTP reverse proxy configured with leoxy/config.yaml. Start it in the foreground or use the start and stop commands to manage it in the background.",
	Example: "  leoxy\n  leoxy start\n  leoxy stop\n  leoxy version",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the Leoxy version",
	Long:    "Print the version of the Leoxy reverse proxy executable.",
	Example: "  leoxy version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("leoxy %s\n", version)
	},
}

var runCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the server",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
