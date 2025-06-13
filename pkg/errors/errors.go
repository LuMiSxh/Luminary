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

import stderrors "errors"

var (
	As     = stderrors.As
	Is     = stderrors.Is
	Unwrap = stderrors.Unwrap
)

var (
	ErrNotFound     = stderrors.New("resource not found")
	ErrUnauthorized = stderrors.New("unauthorized")
	ErrBadRequest   = stderrors.New("bad request")
	ErrServerError  = stderrors.New("server error")
	ErrTimeout      = stderrors.New("operation timed out")
	ErrRateLimit    = stderrors.New("rate limit exceeded")
	ErrInvalidInput = stderrors.New("invalid input")
	ErrNetworkIssue = stderrors.New("network connection issue")
)

func IsNotFound(err error) bool     { return Is(err, ErrNotFound) }
func IsUnauthorized(err error) bool { return Is(err, ErrUnauthorized) }
func IsTimeouted(err error) bool    { return Is(err, ErrTimeout) }
func IsRateLimited(err error) bool  { return Is(err, ErrRateLimit) }
