package agents

import (
	"net/http"
	"time"
)

// BaseAgent provides common functionality for implementing agents
type BaseAgent struct {
	id          string
	name        string
	description string

	SiteURL string
	ApiURL  string
	Client  *http.Client

	// Configuration options
	ThrottleAPI   time.Duration
	ThrottleImage time.Duration
}

// ID returns the agent's identifier - Used for creating the manga UID
func (b *BaseAgent) ID() string {
	return b.id
}

// Name returns the agent's display name
func (b *BaseAgent) Name() string {
	return b.name
}

// Description returns the agent's description
func (b *BaseAgent) Description() string {
	return b.description
}

// NewBaseAgent creates a new BaseAgent with the provided values
func NewBaseAgent(id, name, description string) *BaseAgent {
	return &BaseAgent{
		id:            id,
		name:          name,
		description:   description,
		Client:        &http.Client{Timeout: 30 * time.Second},
		ThrottleAPI:   2 * time.Second,
		ThrottleImage: 500 * time.Millisecond,
	}
}

// Wait pauses execution for the configured throttle duration
func (b *BaseAgent) Wait(isAPI bool) {
	if isAPI {
		time.Sleep(b.ThrottleAPI)
	} else {
		time.Sleep(b.ThrottleImage)
	}
}
