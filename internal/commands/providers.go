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
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List all available manga source providers",
	Long:  `Display a list of all configured manga source providers that Luminary can use to search and read manga.`,
	Run: func(cmd *cobra.Command, args []string) {
		allProviders := appEngine.AllProvider()

		// Use the unified formatter
		formatter := cli.DefaultFormatter

		if len(allProviders) == 0 {
			formatter.PrintWarning("No providers are currently available.")
			return
		}

		formatter.PrintProviderList(allProviders)
	},
}

func init() {
	rootCmd.AddCommand(providersCmd)
}
