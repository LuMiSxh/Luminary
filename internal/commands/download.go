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
	"Luminary/pkg/engine/core"
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	downloadOutput    string
	downloadVolume    int  // Volume flag
	downloadHasVolume bool // Track if the volume flag was provided
)

var downloadCmd = &cobra.Command{
	Use:   "download [chapter-ids...]",
	Short: "Download manga chapters",
	Long:  `Download one or more manga chapters by their IDs.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create base context
		baseCtx := context.Background()

		// Create a context with concurrency settings
		ctx := core.WithConcurrency(baseCtx, maxConcurrency)

		// Add volume override to context if provided
		if downloadHasVolume {
			ctx = core.WithVolumeOverride(ctx, downloadVolume)
		}

		// Use the unified formatter
		formatter := cli.DefaultFormatter

		for _, combinedID := range args {
			// Parse the chapter ID format "provider:id"
			providerID, chapterID, err := core.ParseMangaID(combinedID)
			if formatter.HandleError(err) {
				return
			}

			// Validate that the provider exists
			prov, exists := appEngine.GetProvider(providerID)
			if !exists {
				providerErr := fmt.Errorf("provider '%s' not found", providerID)
				formatter.HandleError(providerErr)
				return
			}

			outputDir := downloadOutput

			// Display download information
			formatter.PrintDownloadInfo(
				chapterID,
				prov.ID(),
				prov.Name(),
				outputDir,
				maxConcurrency,
				downloadVolume,
				downloadHasVolume,
			)

			// Create a context with timeout
			downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			// Perform the download directly to the base output directory
			err = prov.DownloadChapter(downloadCtx, chapterID, outputDir)

			// Always cancel the context when done to release resources
			cancel()

			if formatter.HandleError(err) {
				return
			}

			formatter.PrintSuccess(fmt.Sprintf("Successfully downloaded chapter to %s", outputDir))
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	// Flags
	downloadCmd.Flags().StringVar(&downloadOutput, "output", "./downloads", "Output directory")
	downloadCmd.Flags().IntVar(&downloadVolume, "vol", 0, "Set or override the volume number")

	// Hook to track when the volume flag is explicitly set
	downloadCmd.PreRun = func(cmd *cobra.Command, args []string) {
		downloadHasVolume = cmd.Flags().Changed("vol")
	}
}
