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
	"Open-Generic-Hub/backend/store"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ============= AUTH HANDLERS =============

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if database is initialized BEFORE processing anything
	if s.userStore == nil || s.auditStore == nil {
		respondError(w, http.StatusServiceUnavailable, "Database not initialized. Please configure the system first.")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := s.userStore.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		// Log failed login attempt - try to get user ID even on failure
		errMsg := err.Error()
		var userID string
		var username *string = &req.Username

		// Try to get user by username to log with correct ID
		users, _ := s.userStore.GetUsersByName(req.Username)
		if len(users) > 0 {
			userID = users[0].ID
		} else {
			userID = req.Username // Fallback to username if user not found
		}

		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "login",
			Resource:      "auth",
			ResourceID:    userID,
			AdminID:       nil,
			AdminUsername: username,
			OldValue:      nil,
			NewValue: map[string]interface{}{
				"method": "password",
				"result": "failed",
				"reason": errMsg,
			},
			Status:        "failed",
			ErrorMessage:  &errMsg,
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})

		// Check if user is locked (blocked) - return 423 instead of 401
		if strings.Contains(errMsg, "bloqueada") || strings.Contains(errMsg, "bloqueado") || strings.Contains(errMsg, "permanentemente") {
			respondError(w, http.StatusLocked, "Account is locked. Please contact an administrator.")
			return
		}

		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	username := ""
	if user.Username != nil {
		username = *user.Username
	}

	role := ""
	if user.Role != nil {
		role = *user.Role
	}

	displayName := ""
	if user.DisplayName != nil {
		displayName = *user.DisplayName
	}

	accessToken, err := GenerateJWT(user)
	if err != nil {
		log.Printf("Erro ao gerar access token: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao gerar token")
		return
	}

	refreshToken, err := GenerateRefreshToken(user)
	if err != nil {
		log.Printf("Erro ao gerar refresh token: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao gerar refresh token")
		return
	}

	// Log successful login
	s.auditStore.LogOperation(store.AuditLogRequest{
		Operation:     "login",
		Resource:      "auth",
		ResourceID:    user.ID,
		AdminID:       &user.ID,
		AdminUsername: &username,
		OldValue:      nil,
		NewValue: map[string]interface{}{
			"method": "password",
			"result": "success",
		},
		Status:        "success",
		ErrorMessage:  nil,
		IPAddress:     getIPAddress(r),
		UserAgent:     getUserAgent(r),
		RequestMethod: getRequestMethod(r),
		RequestPath:   getRequestPath(r),
	})

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Login successful",
		Data: map[string]interface{}{
			"token":         accessToken,
			"refresh_token": refreshToken,
			"user_id":       user.ID,
			"username":      username,
			"role":          role,
			"display_name":  displayName,
		},
	})
}

// ============= LOGOUT HANDLER =============

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract user info from JWT (already validated by authMiddleware)
	tokenString := extractTokenFromHeader(r)
	claims, err := ValidateJWT(tokenString, s.userStore)
	if err != nil {
		// Token invalid — still respond OK (user is logging out regardless)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Logged out"})
		return
	}

	// Log logout in audit-logs
	s.auditStore.LogOperation(store.AuditLogRequest{
		Operation:     "logout",
		Resource:      "auth",
		ResourceID:    claims.UserID,
		AdminID:       &claims.UserID,
		AdminUsername: &claims.Username,
		OldValue:      nil,
		NewValue: map[string]interface{}{
			"method": "manual",
			"result": "success",
		},
		Status:        "success",
		ErrorMessage:  nil,
		IPAddress:     getIPAddress(r),
		UserAgent:     getUserAgent(r),
		RequestMethod: getRequestMethod(r),
		RequestPath:   getRequestPath(r),
	})

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Logged out"})
}

// ============= REFRESH TOKEN HANDLER =============

func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if database is initialized
	if s.userStore == nil || s.auditStore == nil {
		respondError(w, http.StatusServiceUnavailable, "Database not initialized. Please configure the system first.")
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, err := ValidateRefreshToken(req.RefreshToken, s.userStore)
	if err != nil {
		// Log failed token refresh
		errMsg := err.Error()
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "login",
			Resource:      "auth",
			ResourceID:    "",
			AdminID:       nil,
			AdminUsername: nil,
			OldValue:      nil,
			NewValue: map[string]interface{}{
				"method": "token",
				"result": "failed",
				"reason": errMsg,
			},
			Status:        "failed",
			ErrorMessage:  &errMsg,
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
		respondError(w, http.StatusUnauthorized, "Refresh token inválido ou expirado")
		return
	}

	user, err := s.userStore.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		errMsg := "Usuário não encontrado"
		if err != nil {
			errMsg = err.Error()
		}
		// Try to get username from user if available
		var usernamePtr *string
		if user != nil && user.Username != nil {
			usernamePtr = user.Username
		}
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "login",
			Resource:      "auth",
			ResourceID:    claims.UserID,
			AdminID:       nil,
			AdminUsername: usernamePtr,
			OldValue:      nil,
			NewValue: map[string]interface{}{
				"method": "token",
				"result": "failed",
				"reason": errMsg,
			},
			Status:        "failed",
			ErrorMessage:  &errMsg,
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
		respondError(w, http.StatusUnauthorized, "Usuário não encontrado")
		return
	}

	accessToken, err := GenerateJWT(user)
	if err != nil {
		log.Printf("Erro ao gerar novo access token: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao gerar novo token")
		return
	}

	// Token refresh bem-sucedido - não loga para evitar poluir audit logs
	// (apenas falhas são logadas para segurança)

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Token renovado com sucesso",
		Data: map[string]interface{}{
			"token": accessToken,
		},
	})
}
