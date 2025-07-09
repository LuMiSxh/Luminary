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

// cmd/luminary/main.go
package main

import (
	"Luminary/internal/cli"
	_ "Luminary/internal/providers" // Import for side effects (auto-registration)
	"Luminary/pkg/engine"
	"Luminary/pkg/provider/registry"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	Version = "dev"
)

func main() {
	// Create engine
	eng := engine.New()
	defer func(eng *engine.Engine) {
		err := eng.Shutdown()
		if err != nil {

		}
	}(eng)

	// Load all registered providers
	if err := registry.LoadAll(eng); err != nil {
		eng.Logger.Error("Failed to load providers: %v", err)
		os.Exit(1)
	}

	// Initialize providers
	ctx := context.Background()
	if err := eng.InitializeProviders(ctx); err != nil {
		eng.Logger.Error("Failed to initialize providers: %v", err)
		// Continue anyway - some providers might have initialized successfully
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		eng.Logger.Info("Received interrupt signal, shutting down...")
		err := eng.Shutdown()
		if err != nil {
			return
		}
		os.Exit(0)
	}()

	// Create CLI app
	app := cli.NewApp(eng, Version)

	// Run CLI
	if err := app.Run(context.Background(), os.Args); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
