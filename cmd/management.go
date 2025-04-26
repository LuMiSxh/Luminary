package cmd

import (
	"Luminary/agents"
	"Luminary/engine"
	"Luminary/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Global engine instance for management commands
	managementEngine *engine.Engine
)

// managementCmd is the root command for management operations
var managementCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage engine settings and resources",
	Long:  `Commands for managing engine settings, cache, and other resources.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize engine on first use
		if managementEngine == nil {
			managementEngine = engine.New()
		}
	},
}

// cacheCmd handles cache management operations
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the engine cache",
	Long:  `View, clear, and manage the engine's cache.`,
}

// clearCacheCmd clears the entire cache
var clearCacheCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the entire cache",
	Long:  `Remove all cached data, including manga info, search results, and downloaded pages.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := managementEngine.Cache.Clear()
		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Error clearing cache: %v\n", err)
			return
		}

		if apiMode {
			utils.OutputJSON("success", map[string]string{
				"message": "Cache cleared successfully",
			}, nil)
			return
		}
		fmt.Println("Cache cleared successfully")
	},
}

// cleanCacheCmd removes expired cache entries
var cleanCacheCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean expired cache entries",
	Long:  `Remove cache entries that have exceeded their time-to-live.`,
	Run: func(cmd *cobra.Command, args []string) {
		count, err := managementEngine.Cache.CleanExpired()
		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Error cleaning cache: %v\n", err)
			return
		}

		if apiMode {
			utils.OutputJSON("success", map[string]interface{}{
				"message":         "Cache cleaned successfully",
				"entries_removed": count,
			}, nil)
			return
		}
		fmt.Printf("Cache cleaned successfully. Removed %d expired entries.\n", count)
	},
}

// cacheInfoCmd shows information about the cache
var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long:  `Display information about the cache, including size and location.`,
	Run: func(cmd *cobra.Command, args []string) {
		info, err := managementEngine.Cache.GetInfo()
		if err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, err)
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Error getting cache info: %v\n", err)
			return
		}

		if apiMode {
			utils.OutputJSON("success", info, nil)
			return
		}

		fmt.Println("Cache Information:")
		fmt.Printf("Location: %s\n", info.Location)
		fmt.Printf("Size: %.2f MB\n", float64(info.Size)/(1024*1024))
		fmt.Printf("Entry Count: %d\n", info.EntryCount)
		fmt.Printf("TTL: %s\n", info.TTL)
	},
}

// loggingCmd handles logging management operations
var loggingCmd = &cobra.Command{
	Use:   "logging",
	Short: "Manage logging settings",
	Long:  `Configure logging levels, output location, and verbosity.`,
}

// setVerboseCmd toggles verbose logging
var setVerboseCmd = &cobra.Command{
	Use:   "verbose [on|off]",
	Short: "Set verbose logging mode",
	Long:  `Enable or disable verbose logging output.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setting := args[0]
		var verbose bool

		switch setting {
		case "on", "true", "1", "yes":
			verbose = true
		case "off", "false", "0", "no":
			verbose = false
		default:
			if apiMode {
				utils.OutputJSON("error", nil, fmt.Errorf("invalid setting: %s (use 'on' or 'off')", setting))
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Invalid setting: %s (use 'on' or 'off')\n", setting)
			return
		}

		managementEngine.Logger.Verbose = verbose

		// Apply the setting to all registered agents
		for _, agent := range agents.All() {
			if eng := agent.GetEngine(); eng != nil {
				eng.Logger.Verbose = verbose
			}
		}

		if apiMode {
			utils.OutputJSON("success", map[string]interface{}{
				"message": fmt.Sprintf("Verbose logging set to %t", verbose),
				"verbose": verbose,
			}, nil)
			return
		}
		fmt.Printf("Verbose logging set to %t\n", verbose)
	},
}

// setLogFileCmd sets the log file location
var setLogFileCmd = &cobra.Command{
	Use:   "logfile [filepath]",
	Short: "Set log file location",
	Long:  `Set the location where logs will be written. Use 'off' to disable file logging.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logPath := args[0]

		if logPath == "off" || logPath == "disable" || logPath == "none" {
			// Disable logging to file
			managementEngine.Logger.LogFile = ""

			// Apply to all agents
			for _, agent := range agents.All() {
				if eng := agent.GetEngine(); eng != nil {
					eng.Logger.LogFile = ""
				}
			}

			if apiMode {
				utils.OutputJSON("success", map[string]string{
					"message": "File logging disabled",
					"logfile": "",
				}, nil)
				return
			}
			fmt.Println("File logging disabled")
			return
		}

		// Ensure the directory exists
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			if apiMode {
				utils.OutputJSON("error", nil, fmt.Errorf("failed to create log directory: %v", err))
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
			return
		}

		// Set the log file path
		managementEngine.Logger.LogFile = logPath

		// Apply to all agents
		for _, agent := range agents.All() {
			if eng := agent.GetEngine(); eng != nil {
				eng.Logger.LogFile = logPath
			}
		}

		// Test writing to the log file
		managementEngine.Logger.Info("Log file location changed to %s", logPath)

		if apiMode {
			utils.OutputJSON("success", map[string]string{
				"message": fmt.Sprintf("Log file location set to %s", logPath),
				"logfile": logPath,
			}, nil)
			return
		}
		fmt.Printf("Log file location set to %s\n", logPath)
	},
}

// setDebugCmd toggles debug logging
var setDebugCmd = &cobra.Command{
	Use:   "debug [on|off]",
	Short: "Set debug logging mode",
	Long:  `Enable or disable debug logging output.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		setting := args[0]
		var debug bool

		switch setting {
		case "on", "true", "1", "yes":
			debug = true
		case "off", "false", "0", "no":
			debug = false
		default:
			if apiMode {
				utils.OutputJSON("error", nil, fmt.Errorf("invalid setting: %s (use 'on' or 'off')", setting))
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "Invalid setting: %s (use 'on' or 'off')\n", setting)
			return
		}

		// Set debug mode in the logger
		managementEngine.Logger.Debug("Setting debug mode to %t", debug)
		managementEngine.Logger.DebugMode = debug

		// Apply the setting to all registered agents
		for _, agent := range agents.All() {
			if eng := agent.GetEngine(); eng != nil {
				eng.Logger.DebugMode = debug
			}
		}

		if apiMode {
			utils.OutputJSON("success", map[string]interface{}{
				"message": fmt.Sprintf("Debug logging set to %t", debug),
				"debug":   debug,
			}, nil)
			return
		}
		fmt.Printf("Debug logging set to %t\n", debug)
	},
}

// httpSettingsCmd handles HTTP client settings
var httpSettingsCmd = &cobra.Command{
	Use:   "http",
	Short: "Manage HTTP client settings",
	Long:  `Configure HTTP client settings like user agent and timeouts.`,
}

// setUserAgentCmd sets the User-Agent header
var setUserAgentCmd = &cobra.Command{
	Use:   "user-agent [string]",
	Short: "Set the User-Agent header",
	Long:  `Set the User-Agent header used for HTTP requests.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		userAgent := args[0]

		// Set in management engine
		managementEngine.HTTP.RequestOptions.UserAgent = userAgent

		// Apply to all agents
		for _, agent := range agents.All() {
			if eng := agent.GetEngine(); eng != nil {
				eng.HTTP.RequestOptions.UserAgent = userAgent
			}
		}

		if apiMode {
			utils.OutputJSON("success", map[string]string{
				"message":    fmt.Sprintf("User-Agent set to %s", userAgent),
				"user_agent": userAgent,
			}, nil)
			return
		}
		fmt.Printf("User-Agent set to %s\n", userAgent)
	},
}

// init adds all management commands to the root command
func init() {
	rootCmd.AddCommand(managementCmd)

	// Add cache commands
	managementCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(clearCacheCmd)
	cacheCmd.AddCommand(cleanCacheCmd)
	cacheCmd.AddCommand(cacheInfoCmd)

	// Add logging commands
	managementCmd.AddCommand(loggingCmd)
	loggingCmd.AddCommand(setVerboseCmd)
	loggingCmd.AddCommand(setLogFileCmd)

	// Add HTTP settings commands
	managementCmd.AddCommand(httpSettingsCmd)
	httpSettingsCmd.AddCommand(setUserAgentCmd)
}
