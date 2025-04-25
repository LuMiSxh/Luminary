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
	status      string
	tags        []string

	SiteURL string
	ApiURL  string
	Client  *http.Client

	// Configuration options
	ThrottleAPI   time.Duration
	ThrottleImage time.Duration
}

// ID returns the agent's identifier
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

// Status returns the agent's status
func (b *BaseAgent) Status() string {
	return b.status
}

// Tags returns the agent's tags/categories
func (b *BaseAgent) Tags() []string {
	return b.tags
}

// NewBaseAgent creates a new BaseAgent with the provided values
func NewBaseAgent(id, name, description string, status string, tags []string) *BaseAgent {
	return &BaseAgent{
		id:            id,
		name:          name,
		description:   description,
		status:        status,
		tags:          tags,
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
