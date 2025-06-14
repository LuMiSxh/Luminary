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
	"Luminary/pkg/cli"
	"github.com/spf13/cobra"
	"runtime"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display detailed version information for Luminary, including the log file location.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get log file location if available
		logFile := ""
		if appEngine != nil && appEngine.Logger != nil {
			logFile = appEngine.Logger.LogFile
		}

		// Use the unified formatter
		formatter := cli.DefaultFormatter
		formatter.PrintVersionInfo(
			version,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
			logFile,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
