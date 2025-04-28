package cmd

import (
	"Luminary/engine"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiMode bool
var appEngine *engine.Engine

var rootCmd = &cobra.Command{
	Use:   "Luminary",
	Short: "Luminary is a CLI tool for searching and downloading manga.",
	Long:  "Luminary is a CLI tool for searching and downloading manga. It allows you to search for manga by title, author, or genre, and download chapters or volumes directly to your device.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize engine if not already done
		if appEngine == nil {
			appEngine = initializeEngine()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// When no command is specified, display help
		if err := cmd.Help(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

// initializeEngine creates and initializes the engine and all agents
func initializeEngine() *engine.Engine {
	// Create a new engine instance
	e := engine.New()

	// Register all available agents
	// This would typically iterate through all agent packages and call their NewAgent functions
	// For demonstration, we're using placeholder code
	// In a real implementation, each agent package would export a NewAgent function

	// Example of registering agents:
	// mangadex.RegisterAgent(e)
	// other_agent.RegisterAgent(e)

	return e
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
