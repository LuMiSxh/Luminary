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
	"Luminary/pkg/engine/core"
	"Luminary/pkg/errors"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	downloadOutput    string
	downloadVolume    int  // Volume flag
	downloadHasVolume bool // Track if the volume flag was provided
)

// DownloadInfo represents download info for API responses
type DownloadInfo struct {
	ChapterID    string `json:"chapter_id"`
	Provider     string `json:"provider"`
	ProviderName string `json:"provider_name"`
	OutputDir    string `json:"output_dir"`
	Volume       *int   `json:"volume,omitempty"` // Volume in API response
}

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

		for _, combinedID := range args {
			// Parse the chapter ID format "provider:id"
			providerID, chapterID, err := core.ParseMangaID(combinedID)
			if err != nil {
				if apiMode {
					util.OutputJSON("error", nil, err)
					return
				}

				fmt.Println("Error: invalid chapter ID format, must be 'provider:id'")
				return
			}

			// Validate that the provider exists
			prov, exists := appEngine.GetProvider(providerID)
			if !exists {
				if apiMode {
					util.OutputJSON("error", nil, fmt.Errorf("provider '%s' not found", providerID))
					return
				}

				fmt.Printf("Error: Provider '%s' not found\n", providerID)
				fmt.Println("Available Provider:")
				for _, a := range appEngine.AllProvider() {
					fmt.Printf("  - %s (%s)\n", a.ID(), a.Name())
				}
				return
			}

			outputDir := downloadOutput

			if apiMode {
				// Prepare download info for API response
				downloadInfo := DownloadInfo{
					ChapterID:    chapterID,
					Provider:     prov.ID(),
					ProviderName: prov.Name(),
					OutputDir:    outputDir,
				}

				// Add volume info if provided
				if downloadHasVolume {
					downloadInfo.Volume = &downloadVolume
				}

				util.OutputJSON("downloading", downloadInfo, nil)
			} else {
				// Interactive mode for CLI users
				fmt.Printf("Downloading chapter %s from provider %s (%s)...\n",
					chapterID, prov.ID(), prov.Name())
				fmt.Printf("Output directory: %s\n", downloadOutput)
				fmt.Printf("Concurrent downloads: %d\n", maxConcurrency)

				// Print volume info if provided
				if downloadHasVolume {
					fmt.Printf("Volume override: %d\n", downloadVolume)
				}
			}

			// Create a context with timeout
			downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)

			// Perform the download directly to the base output directory
			err = prov.DownloadChapter(downloadCtx, chapterID, outputDir)

			// Always cancel the context when done to release resources
			cancel()

			if err != nil {
				handleDownloadError(err, providerID, chapterID, prov.Name(), outputDir)
				return
			}

			// If we're in CLI mode, provide a success message
			if !apiMode {
				fmt.Printf("Successfully downloaded chapter to %s\n", outputDir)
			} else {
				// Report successful download in API mode
				util.OutputJSON("success", map[string]interface{}{
					"message":    fmt.Sprintf("Successfully downloaded chapter %s", chapterID),
					"chapter_id": chapterID,
					"provider":   prov.ID(),
					"output_dir": outputDir,
				}, nil)
			}
		}
	},
}

// handleDownloadError provides user-friendly error messages based on error type
func handleDownloadError(err error, providerID, chapterID, providerName, outputDir string) {
	// Check for specific error types in order of specificity
	if errors.IsNotFound(err) {
		// Not found error
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("chapter '%s' not found on %s", chapterID, providerName))
		} else {
			fmt.Printf("Error: Chapter '%s' not found on %s\n", chapterID, providerName)
		}
		return
	}

	// Check for server errors
	if errors.IsServerError(err) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("server error from %s: %v", providerName, err))
		} else {
			fmt.Printf("Error: Server error from %s. Please try again later.\n", providerName)
		}
		return
	}

	// Check for rate limiting
	if errors.Is(err, errors.ErrRateLimit) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("rate limit exceeded for %s", providerName))
		} else {
			fmt.Printf("Error: Rate limit exceeded for %s. Please try again later.\n", providerName)
		}
		return
	}

	// File system errors
	var ioErr *os.PathError
	if errors.As(err, &ioErr) {
		if apiMode {
			util.OutputJSON("error", nil, fmt.Errorf("file system error: %v", ioErr))
		} else {
			fmt.Printf("Error: Failed to access output directory '%s': %v\n", outputDir, ioErr)
			fmt.Println("Make sure the directory exists and you have write permissions.")
		}
		return
	}

	// Generic error handling
	if apiMode {
		util.OutputJSON("error", nil, err)
	} else {
		fmt.Printf("Error downloading chapter: %v\n", err)
	}
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
