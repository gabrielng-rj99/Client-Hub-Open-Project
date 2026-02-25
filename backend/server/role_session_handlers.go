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
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"Open-Generic-Hub/backend/store"
)

// ============================================
// Request/Response Types
// ============================================

// RoleSessionPolicyRequest represents session policy for a role
type RoleSessionPolicyRequest struct {
	SessionDurationMinutes      int  `json:"session_duration_minutes"`
	RefreshTokenDurationMinutes int  `json:"refresh_token_duration_minutes"`
	MaxConcurrentSessions       *int `json:"max_concurrent_sessions,omitempty"`
	IdleTimeoutMinutes          *int `json:"idle_timeout_minutes,omitempty"`
	Require2FA                  bool `json:"require_2fa"`
}

// RoleSessionPolicyResponse represents session policy in API responses
type RoleSessionPolicyResponse struct {
	ID                          string `json:"id"`
	RoleID                      string `json:"role_id"`
	RoleName                    string `json:"role_name"`
	SessionDurationMinutes      int    `json:"session_duration_minutes"`
	RefreshTokenDurationMinutes int    `json:"refresh_token_duration_minutes"`
	MaxConcurrentSessions       *int   `json:"max_concurrent_sessions,omitempty"`
	IdleTimeoutMinutes          *int   `json:"idle_timeout_minutes,omitempty"`
	Require2FA                  bool   `json:"require_2fa"`
	IsActive                    bool   `json:"is_active"`
}

// ============================================
// Session Policy Handlers
// ============================================

// HandleRoleSessionPoliciesRoute routes GET requests for all role session policies
func (s *Server) HandleRoleSessionPoliciesRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate token - Root only
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Check if user is root via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		respondError(w, http.StatusForbidden, "Apenas root pode visualizar políticas de sessão")
		return
	}

	s.HandleGetAllRoleSessionPolicies(w, r)
}

// HandleRoleSessionPolicyByIDRoute routes GET/PUT requests for a specific role's session policy
func (s *Server) HandleRoleSessionPolicyByIDRoute(w http.ResponseWriter, r *http.Request) {
	// Extract role ID from path: /api/roles/{id}/session-policy
	path := strings.TrimPrefix(r.URL.Path, "/api/roles/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Role ID required")
		return
	}
	roleID := parts[0]

	// Validate token - Root only
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Check if user is root via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		respondError(w, http.StatusForbidden, "Apenas root pode gerenciar políticas de sessão")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.HandleGetRoleSessionPolicy(w, r, roleID)
	case http.MethodPut:
		s.HandleUpdateRoleSessionPolicy(w, r, roleID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleGetAllRoleSessionPolicies returns all role session policies
func (s *Server) HandleGetAllRoleSessionPolicies(w http.ResponseWriter, r *http.Request) {
	db := GetGlobalDB()
	if db == nil {
		respondError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	query := `
		SELECT
			rsp.id, rsp.role_id, r.name as role_name,
			rsp.session_duration_minutes, rsp.refresh_token_duration_minutes,
			rsp.max_concurrent_sessions, rsp.idle_timeout_minutes,
			rsp.require_2fa, rsp.is_active
		FROM role_session_policies rsp
		JOIN roles r ON rsp.role_id = r.id
		WHERE rsp.is_active = TRUE
		ORDER BY r.priority DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Error querying session policies: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar políticas de sessão")
		return
	}
	defer rows.Close()

	var policies []RoleSessionPolicyResponse
	for rows.Next() {
		var policy RoleSessionPolicyResponse
		var maxSessions, idleTimeout sql.NullInt64

		err := rows.Scan(
			&policy.ID,
			&policy.RoleID,
			&policy.RoleName,
			&policy.SessionDurationMinutes,
			&policy.RefreshTokenDurationMinutes,
			&maxSessions,
			&idleTimeout,
			&policy.Require2FA,
			&policy.IsActive,
		)
		if err != nil {
			log.Printf("❌ Error scanning session policy: %v", err)
			continue
		}

		if maxSessions.Valid {
			val := int(maxSessions.Int64)
			policy.MaxConcurrentSessions = &val
		}
		if idleTimeout.Valid {
			val := int(idleTimeout.Int64)
			policy.IdleTimeoutMinutes = &val
		}

		policies = append(policies, policy)
	}

	respondJSON(w, http.StatusOK, policies)
}

// HandleGetRoleSessionPolicy returns session policy for a specific role
func (s *Server) HandleGetRoleSessionPolicy(w http.ResponseWriter, r *http.Request, roleID string) {
	db := GetGlobalDB()
	if db == nil {
		respondError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Check if role exists
	var roleName string
	err := db.QueryRow("SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Role não encontrado")
			return
		}
		log.Printf("❌ Error checking role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar role")
		return
	}

	query := `
		SELECT
			id, role_id, session_duration_minutes, refresh_token_duration_minutes,
			max_concurrent_sessions, idle_timeout_minutes, require_2fa, is_active
		FROM role_session_policies
		WHERE role_id = $1 AND is_active = TRUE
	`

	var policy RoleSessionPolicyResponse
	var maxSessions, idleTimeout sql.NullInt64

	err = db.QueryRow(query, roleID).Scan(
		&policy.ID,
		&policy.RoleID,
		&policy.SessionDurationMinutes,
		&policy.RefreshTokenDurationMinutes,
		&maxSessions,
		&idleTimeout,
		&policy.Require2FA,
		&policy.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default policy if none exists
			policy = RoleSessionPolicyResponse{
				RoleID:                      roleID,
				RoleName:                    roleName,
				SessionDurationMinutes:      60,
				RefreshTokenDurationMinutes: 10080,
				MaxConcurrentSessions:       nil,
				IdleTimeoutMinutes:          nil,
				Require2FA:                  false,
				IsActive:                    false,
			}
			respondJSON(w, http.StatusOK, policy)
			return
		}
		log.Printf("❌ Error querying session policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar política de sessão")
		return
	}

	policy.RoleName = roleName
	if maxSessions.Valid {
		val := int(maxSessions.Int64)
		policy.MaxConcurrentSessions = &val
	}
	if idleTimeout.Valid {
		val := int(idleTimeout.Int64)
		policy.IdleTimeoutMinutes = &val
	}

	respondJSON(w, http.StatusOK, policy)
}

// HandleUpdateRoleSessionPolicy updates or creates session policy for a role
func (s *Server) HandleUpdateRoleSessionPolicy(w http.ResponseWriter, r *http.Request, roleID string) {
	db := GetGlobalDB()
	if db == nil {
		respondError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Get claims for audit
	tokenString := extractTokenFromHeader(r)
	claims, _ := ExtractJWTClaimsUnverified(tokenString)

	// Check if role exists
	var roleName string
	err := db.QueryRow("SELECT name FROM roles WHERE id = $1", roleID).Scan(&roleName)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Role não encontrado")
			return
		}
		log.Printf("❌ Error checking role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar role")
		return
	}

	// Parse request
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req RoleSessionPolicyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate values
	if req.SessionDurationMinutes < 5 || req.SessionDurationMinutes > 1440 {
		respondError(w, http.StatusBadRequest, "Duração de sessão deve estar entre 5 e 1440 minutos")
		return
	}

	if req.RefreshTokenDurationMinutes < 60 || req.RefreshTokenDurationMinutes > 525600 {
		respondError(w, http.StatusBadRequest, "Duração de refresh token deve estar entre 60 minutos e 365 dias")
		return
	}

	if req.MaxConcurrentSessions != nil && *req.MaxConcurrentSessions < 1 {
		respondError(w, http.StatusBadRequest, "Número de sessões simultâneas deve ser maior que 0")
		return
	}

	if req.IdleTimeoutMinutes != nil && *req.IdleTimeoutMinutes < 5 {
		respondError(w, http.StatusBadRequest, "Timeout de inatividade deve ser maior que 5 minutos")
		return
	}

	// Get old value for audit
	var oldPolicy *RoleSessionPolicyResponse
	var oldID, oldRoleID string
	var oldSession, oldRefresh int
	var oldMaxSessions, oldIdle sql.NullInt64
	var old2FA, oldActive bool

	queryOld := `
		SELECT id, role_id, session_duration_minutes, refresh_token_duration_minutes,
		       max_concurrent_sessions, idle_timeout_minutes, require_2fa, is_active
		FROM role_session_policies
		WHERE role_id = $1 AND is_active = TRUE
	`
	err = db.QueryRow(queryOld, roleID).Scan(
		&oldID, &oldRoleID, &oldSession, &oldRefresh,
		&oldMaxSessions, &oldIdle, &old2FA, &oldActive,
	)
	if err == nil {
		oldPolicy = &RoleSessionPolicyResponse{
			ID:                          oldID,
			RoleID:                      oldRoleID,
			SessionDurationMinutes:      oldSession,
			RefreshTokenDurationMinutes: oldRefresh,
			Require2FA:                  old2FA,
			IsActive:                    oldActive,
		}
		if oldMaxSessions.Valid {
			val := int(oldMaxSessions.Int64)
			oldPolicy.MaxConcurrentSessions = &val
		}
		if oldIdle.Valid {
			val := int(oldIdle.Int64)
			oldPolicy.IdleTimeoutMinutes = &val
		}
	}

	// Upsert session policy
	query := `
		INSERT INTO role_session_policies (
			role_id, session_duration_minutes, refresh_token_duration_minutes,
			max_concurrent_sessions, idle_timeout_minutes, require_2fa, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		ON CONFLICT (role_id, is_active) DO UPDATE SET
			session_duration_minutes = EXCLUDED.session_duration_minutes,
			refresh_token_duration_minutes = EXCLUDED.refresh_token_duration_minutes,
			max_concurrent_sessions = EXCLUDED.max_concurrent_sessions,
			idle_timeout_minutes = EXCLUDED.idle_timeout_minutes,
			require_2fa = EXCLUDED.require_2fa,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var policyID string
	err = db.QueryRow(
		query,
		roleID,
		req.SessionDurationMinutes,
		req.RefreshTokenDurationMinutes,
		req.MaxConcurrentSessions,
		req.IdleTimeoutMinutes,
		req.Require2FA,
	).Scan(&policyID)

	if err != nil {
		log.Printf("❌ Error upserting session policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao salvar política de sessão")
		return
	}

	log.Printf("✅ User %s updated session policy for role %s", claims.Username, roleName)

	// Log to audit
	if s.auditStore != nil {
		userAgent := r.UserAgent()
		method := r.Method
		path := r.URL.Path
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "role_session_policy",
			ResourceID:    roleID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      oldPolicy,
			NewValue:      req,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     &userAgent,
			RequestMethod: &method,
			RequestPath:   &path,
		})
	}

	response := RoleSessionPolicyResponse{
		ID:                          policyID,
		RoleID:                      roleID,
		RoleName:                    roleName,
		SessionDurationMinutes:      req.SessionDurationMinutes,
		RefreshTokenDurationMinutes: req.RefreshTokenDurationMinutes,
		MaxConcurrentSessions:       req.MaxConcurrentSessions,
		IdleTimeoutMinutes:          req.IdleTimeoutMinutes,
		Require2FA:                  req.Require2FA,
		IsActive:                    true,
	}

	respondJSON(w, http.StatusOK, response)
}
