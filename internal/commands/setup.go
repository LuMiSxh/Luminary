package commands

import "Luminary/pkg/engine"

// SetupEngine makes the engine available to all command handlers
func SetupEngine(e *engine.Engine) {
	appEngine = e
}

// SetupVersion sets the version for all commands
func SetupVersion(v string) {
	version = v
	rootCmd.Version = v
}
