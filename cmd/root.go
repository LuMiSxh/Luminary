package cmd

import (
	"Luminary/engine"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiMode bool
var maxConcurrency int
var appEngine *engine.Engine
var version string

var rootCmd = &cobra.Command{
	Use:   "Luminary",
	Short: "Luminary is a CLI tool for searching and downloading manga.",
	Long:  "Luminary is a CLI tool for searching and downloading manga. It allows you to search for manga by title, author, or genre, and download chapters or volumes directly to your device.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize engine if not already done
		if appEngine == nil {
			appEngine = engine.New()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// When no command is specified, display help
		if err := cmd.Help(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Oops. An error while executing Luminary '%s'\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&apiMode, "api", false, "Output machine-readable JSON only")
	rootCmd.PersistentFlags().IntVar(&maxConcurrency, "concurrency", 5, "Maximum number of concurrent operations")
}
