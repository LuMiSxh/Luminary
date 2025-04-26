package cmd

import "Luminary/engine"

// SetupEngine makes the engine available to all command handlers
func SetupEngine(e *engine.Engine) {
	appEngine = e
}
