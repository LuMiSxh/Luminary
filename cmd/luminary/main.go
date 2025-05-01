package main

import (
	"Luminary/internal/commands"
	"Luminary/internal/providers"
	"Luminary/pkg/engine"
)

// Version is set during build using -ldflags
var Version = "0.0.0-dev"

// registerProviders registers all available manga source providers with the engine
func registerProviders(e *engine.Engine) {
	// Register MangaDex provider
	err := e.RegisterProvider(providers.NewMangadexProvider(e))
	if err != nil {
		e.Logger.Error("Failed to register MangaDex provider: %v", err)
	}

	// Register KissManga provider
	err = e.RegisterProvider(providers.NewMadaraProvider(e))
	if err != nil {
		e.Logger.Error("Failed to register KissManga provider: %v", err)
	}
}

func main() {
	// Initialize the engine
	e := engine.New()

	// Register all providers
	registerProviders(e)

	// Make the engine available to commands
	commands.SetupEngine(e)

	// Set the version for the root command
	commands.SetupVersion(Version)

	// Execute the root command
	commands.Execute()
}
