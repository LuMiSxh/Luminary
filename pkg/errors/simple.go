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

package errors

// T (Track) - Simple wrapper for error tracking
func T(err error) error {
	if err == nil {
		return nil
	}
	return Track(err).Error()
}

// TC (Track with Context) - Track error with context data
func TC(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return Track(err).WithContextMap(context).Error()
}

// TM (Track with Message) - Track error with user-friendly message
func TM(err error, message string) error {
	if err == nil {
		return nil
	}
	return Track(err).WithMessage(message).Error()
}

// TN (Track Network) - Track as network error
func TN(err error) error {
	if err == nil {
		return nil
	}
	return Track(err).AsNetwork().Error()
}

// TP (Track Provider) - Track as provider error
func TP(err error, providerID string) error {
	if err == nil {
		return nil
	}
	return Track(err).AsProvider(providerID).Error()
}
