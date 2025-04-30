package cmd

import (
	"Luminary/utils"
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display detailed version information for Luminary.`,
	Run: func(cmd *cobra.Command, args []string) {
		if apiMode {
			// API mode - use standardized JSON output
			versionData := map[string]string{
				"version":    version,
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}
			utils.OutputJSON("success", versionData, nil)
		} else {
			// Interactive mode - pretty output
			fmt.Printf("Luminary version: %s\n", version)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
