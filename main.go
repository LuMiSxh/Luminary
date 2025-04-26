package main

import (
	"Luminary/agents/mangadex"
	"Luminary/cmd"
	"Luminary/engine"
)

// registerAgents registers all available manga source agents with the engine
func registerAgents(e *engine.Engine) {
	// Register MangaDex agent
	err := e.RegisterAgent(mangadex.NewAgent(e))
	if err != nil {
		return
	}
}

func main() {
	// Initialize the engine
	e := engine.New()

	// Register all agents
	registerAgents(e)

	// Make the engine available to commands
	cmd.SetupEngine(e)

	// Execute the root command
	cmd.Execute()
}
