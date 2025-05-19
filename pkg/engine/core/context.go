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
