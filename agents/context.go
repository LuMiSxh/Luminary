package agents

import "context"

// ContextKey is a type for context keys specific to agents
type ContextKey string

const (
	ConcurrencyKey ContextKey = "concurrency"
)

// WithConcurrency adds concurrency settings to a context
func WithConcurrency(ctx context.Context, limit int) context.Context {
	return context.WithValue(ctx, ConcurrencyKey, limit)
}

// GetConcurrency retrieves concurrency settings from a context
func GetConcurrency(ctx context.Context, defaultLimit int) int {
	if limit, ok := ctx.Value(ConcurrencyKey).(int); ok && limit > 0 {
		return limit
	}
	return defaultLimit
}
