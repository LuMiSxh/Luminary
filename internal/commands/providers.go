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
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List all available manga source providers",
	Long:  `Display a list of all configured manga source providers that Luminary can use to search and read manga.`,
	Run: func(cmd *cobra.Command, args []string) {
		allProviders := appEngine.AllProvider()

		// Sort providers alphabetically
		sort.Slice(allProviders, func(i, j int) bool {
			return allProviders[i].Name() < allProviders[j].Name()
		})

		fmt.Println("Available manga source providers:")
		fmt.Println("")

		format := "%-12s %-20s %s\n"
		fmt.Printf(format, "ID", "NAME", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 80))

		for _, provider := range allProviders {
			fmt.Printf(format,
				provider.ID(),
				provider.Name(),
				provider.Description())
		}

		fmt.Println("")
		fmt.Println("Use --provider flag with the search command to specify a particular provider")
	},
}

func init() {
	rootCmd.AddCommand(providersCmd)
}
