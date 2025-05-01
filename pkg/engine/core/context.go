package core

import "context"

// ContextKey is a type for context keys specific to engine
type ContextKey string

const (
	ConcurrencyKey    ContextKey = "concurrency"
	VolumeOverrideKey ContextKey = "volume_override"
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

// WithVolumeOverride adds volume override to a context
func WithVolumeOverride(ctx context.Context, volume int) context.Context {
	return context.WithValue(ctx, VolumeOverrideKey, volume)
}

// GetVolumeOverride checks if a volume override is set in the context
func GetVolumeOverride(ctx context.Context) (int, bool) {
	if val := ctx.Value(VolumeOverrideKey); val != nil {
		if volNum, ok := val.(int); ok && volNum > 0 {
			return volNum, true
		}
	}
	return 0, false
}
