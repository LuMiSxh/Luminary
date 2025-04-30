package main

import (
	"Luminary/agents/kissmanga"
	"Luminary/agents/mangadex"
	"Luminary/cmd"
	"Luminary/engine"
)

// Version is set during build using -ldflags
var Version = "0.0.0-dev"

// registerAgents registers all available manga source agents with the engine
func registerAgents(e *engine.Engine) {
	// Register MangaDex agent
	err := e.RegisterAgent(mangadex.NewAgent(e))
	if err != nil {
		e.Logger.Error("Failed to register MangaDex agent: %v", err)
	}

	// Register KissManga agent
	err = e.RegisterAgent(kissmanga.NewAgent(e))
	if err != nil {
		e.Logger.Error("Failed to register KissManga agent: %v", err)
	}
}

func main() {
	// Initialize the engine
	e := engine.New()

	// Register all agents
	registerAgents(e)

	// Make the engine available to commands
	cmd.SetupEngine(e)

	// Set the version for the root command
	cmd.SetupVersion(Version)

	// Execute the root command
	cmd.Execute()
}
