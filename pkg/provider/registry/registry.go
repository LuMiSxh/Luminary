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

package registry

import (
	"Luminary/pkg/engine"
	"sync"
)

// ProviderConstructor is a function that creates a provider instance
type ProviderConstructor func(*engine.Engine) engine.Provider

// Registry holds provider constructors
type Registry struct {
	constructors []ProviderConstructor
	mu           sync.RWMutex
}

// global registry instance
var global = &Registry{
	constructors: make([]ProviderConstructor, 0),
}

// Register adds a provider constructor to the global registry
func Register(constructor ProviderConstructor) {
	global.mu.Lock()
	defer global.mu.Unlock()

	global.constructors = append(global.constructors, constructor)
}

// LoadAll creates and registers all providers with the engine
func LoadAll(e *engine.Engine) error {
	global.mu.RLock()
	constructors := make([]ProviderConstructor, len(global.constructors))
	copy(constructors, global.constructors)
	global.mu.RUnlock()

	for _, constructor := range constructors {
		provider := constructor(e)
		if provider == nil {
			continue
		}

		if err := e.RegisterProvider(provider); err != nil {
			e.Logger.Error("Failed to register provider: %v", err)
			// Continue with other providers
		}
	}

	e.Logger.Info("Loaded %d providers", e.ProviderCount())
	return nil
}

// Clear removes all registered constructors (useful for testing)
func Clear() {
	global.mu.Lock()
	defer global.mu.Unlock()

	global.constructors = global.constructors[:0]
}

// Count returns the number of registered constructors
func Count() int {
	global.mu.RLock()
	defer global.mu.RUnlock()

	return len(global.constructors)
}
