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

import (
	"fmt"
	"strings"
)

// FormatMangaID creates a standardized manga ID in the format "provider:id"
func FormatMangaID(providerID, mangaID string) string {
	return fmt.Sprintf("%s:%s", providerID, mangaID)
}

// ParseMangaID parses a manga ID in the format "provider:id" into its components
func ParseMangaID(combinedID string) (providerID string, mangaID string, err error) {
	parts := strings.SplitN(combinedID, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format, must be 'provider:id'")
	}
	return parts[0], parts[1], nil
}
