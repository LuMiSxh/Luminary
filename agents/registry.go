package agents

import (
	"context"
	"fmt"
	"sync"
)

var (
	registry      = make(map[string]Agent)
	registryMutex sync.RWMutex
)

// Register adds an agent to the registry
func Register(agent Agent) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	registry[agent.ID()] = agent
}

// Get retrieves an agent by ID
func Get(id string) Agent {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	return registry[id]
}

// All returns all registered agents
func All() []Agent {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	agents := make([]Agent, 0, len(registry))
	for _, a := range registry {
		agents = append(agents, a)
	}
	return agents
}

// FindAgentForURI finds the first agent that can handle the given URI
func FindAgentForURI(uri string) Agent {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	for _, agent := range registry {
		if canHandle, ok := agent.(interface{ CanHandleURI(string) bool }); ok {
			if canHandle.CanHandleURI(uri) {
				return agent
			}
		}
	}
	return nil
}

// Initialize initializes all registered agents
func Initialize(ctx context.Context) error {
	for id, agent := range registry {
		if initializer, ok := agent.(interface{ Initialize(context.Context) error }); ok {
			if err := initializer.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize agent %s: %w", id, err)
			}
		}
	}
	return nil
}
