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

package commands

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/errors"
)

// SetupEngine makes the engine available to all command handlers
func SetupEngine(e *engine.Engine) {
	appEngine = e
}

// SetupVersion sets the version for all commands
func SetupVersion(v string) {
	version = v
	rootCmd.Version = v
}

// SetupDebugMode enables debug mode for the CLI
func SetupDebugMode() {
	if debugMode {
		// Create a debug formatter with all details enabled
		debugFormatter := errors.NewDebugCLIFormatter()
		debugFormatter.ShowDebugInfo = true
		debugFormatter.ShowFunctionChain = true
		debugFormatter.ShowTimestamps = true
		errors.DefaultCLIFormatter = debugFormatter
	} else if verboseErrors {
		// For verbose-only mode, create a formatter that just shows function chains
		formatter := errors.NewCLIFormatter()
		formatter.ShowFunctionChain = true
		errors.DefaultCLIFormatter = formatter
	}
}
