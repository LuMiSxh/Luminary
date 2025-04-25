package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var apiMode bool

var rootCmd = &cobra.Command{
	Use:   "Luminary",
	Short: "Luminary is a CLI tool for searching and downloading manga.",
	Long:  "Luminary is a CLI tool for searching and downloading manga. It allows you to search for manga by title, author, or genre, and download chapters or volumes directly to your device.",
	Run: func(cmd *cobra.Command, args []string) {

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
}
