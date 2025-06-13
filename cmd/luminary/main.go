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

	// Enable debug mode if needed
	commands.SetupDebugMode()

	// Execute the root command
	commands.Execute()
}
