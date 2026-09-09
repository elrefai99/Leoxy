package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "v0.2.2"

var rootCmd = &cobra.Command{
	Use:     "leoxy",
	Short:   "Leoxy reverse proxy service",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Leoxy version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("leoxy %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
