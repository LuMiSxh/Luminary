// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package commands

import (
	"Luminary/pkg/engine"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appEngine      *engine.Engine
	maxConcurrency int
	version        string
	debugMode      bool
	verboseErrors  bool
)

var rootCmd = &cobra.Command{
	Use:   "Luminary",
	Short: "Luminary is a CLI tool for searching and downloading manga.",
	Long:  "Luminary is a CLI tool for searching and downloading manga. It allows you to search for manga by title, author, or genre, and download chapters or volumes directly to your device.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize engine if not already done
		if appEngine == nil {
			appEngine = engine.New()
		}

		// Set up debug mode based on flags
		SetupDebugMode()
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
	rootCmd.PersistentFlags().IntVar(&maxConcurrency, "concurrency", 5, "Maximum number of concurrent operations")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug mode with detailed error information")
	rootCmd.PersistentFlags().BoolVar(&verboseErrors, "verbose-errors", false, "Show function call chains in errors")
}
