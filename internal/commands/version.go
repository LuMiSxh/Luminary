package commands

import (
	"Luminary/pkg/util"
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display detailed version information for Luminary, including the log file location.`,
	Run: func(cmd *cobra.Command, args []string) {
		if apiMode {
			// API mode - use standardized JSON output
			versionData := map[string]interface{}{
				"version":    version,
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}

			// Add log file location if available
			if appEngine != nil && appEngine.Logger != nil {
				logFile := appEngine.Logger.LogFile
				if logFile != "" {
					versionData["log_file"] = logFile
				} else {
					versionData["log_file"] = "disabled"
				}
			}

			util.OutputJSON("success", versionData, nil)
		} else {
			// Interactive mode - pretty output
			fmt.Printf("Luminary version: %s\n", version)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

			// Add log file location if available
			if appEngine != nil && appEngine.Logger != nil {
				logFile := appEngine.Logger.LogFile
				if logFile != "" {
					fmt.Printf("Log file: %s\n", logFile)
				} else {
					fmt.Println("Logging to file: disabled")
				}
			} else {
				fmt.Println("Logger: not initialized")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
