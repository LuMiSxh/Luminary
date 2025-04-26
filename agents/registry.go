package agents

import (
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
