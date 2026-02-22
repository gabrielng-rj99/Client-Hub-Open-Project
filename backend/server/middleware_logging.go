/*
 * Client Hub Open Project
 * Copyright (C) 2025 Client Hub Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"Open-Generic-Hub/backend/store"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// requestLogDataKey is the key context for the RequestLogData pointer
	requestLogDataKey contextKey = "requestLogData"
)

// RequestLogData holds data that subsequent middlewares/handlers can populate
// so the logging middleware can record it at the end of the request.
type RequestLogData struct {
	UserID   string
	Username string
	Role     string
}

// responseWriterWrapper wraps http.ResponseWriter to capture the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w}
}

func (rw *responseWriterWrapper) Status() int {
	return rw.status
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// loggingMiddleware captures request details and logs them after completion
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create shared data container and inject into context
		logData := &RequestLogData{}
		ctx := context.WithValue(r.Context(), requestLogDataKey, logData)
		r = r.WithContext(ctx)

		wrapped := wrapResponseWriter(w)
		// Default to 200 OK if WriteHeader is never called
		wrapped.status = http.StatusOK

		// Process request
		next(wrapped, r)

		duration := time.Since(start)

		// Extract page from Referer header (e.g., "http://localhost:5173/contracts" → "/contracts")
		page := "-"
		if referer := r.Header.Get("Referer"); referer != "" {
			if idx := strings.Index(referer, "//"); idx != -1 {
				rest := referer[idx+2:]
				if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
					page = rest[slashIdx:]
					// Strip query params from page
					if qIdx := strings.Index(page, "?"); qIdx != -1 {
						page = page[:qIdx]
					}
				}
			}
		}

		// Format duration to fixed width (right-aligned, 12 chars)
		durStr := fmt.Sprintf("%12s", duration.Truncate(time.Microsecond))

		// User and role (variable width, at the end)
		userName := "-"
		userRole := "-"
		if logData.Username != "" {
			userName = logData.Username
			userRole = logData.Role
		}

		// Fixed-width columnar format:
		// PAGE(14) | METHOD(6) | API(35) | STATUS(3) | DURATION(12) | user, role
		log.Printf("%-14s | %-6s | %-35s | %3d | %s | %s, %s",
			page,
			r.Method,
			r.URL.Path,
			wrapped.Status(),
			durStr,
			userName,
			userRole,
		)
	}
}

// setRequestUser allows auth middleware to set the user for logging
// Role is fetched from DB since it's no longer in JWT claims
func setRequestUser(r *http.Request, claims *JWTClaims, roleStore *store.RoleStore) {
	if val := r.Context().Value(requestLogDataKey); val != nil {
		if logData, ok := val.(*RequestLogData); ok {
			logData.UserID = claims.UserID
			logData.Username = claims.Username
			// Fetch role from DB since it's not in JWT anymore
			if roleStore != nil {
				role, err := roleStore.GetUserRole(claims.UserID)
				if err == nil {
					logData.Role = role
				} else {
					logData.Role = "unknown"
				}
			} else {
				logData.Role = "unknown"
			}
		}
	}
}
