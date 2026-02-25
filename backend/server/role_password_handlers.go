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

// RolePasswordPolicyRequest represents password policy for a role
type RolePasswordPolicyRequest struct {
	MinLength            int     `json:"min_length"`
	MaxLength            int     `json:"max_length,omitempty"`
	RequireUppercase     bool    `json:"require_uppercase"`
	RequireLowercase     bool    `json:"require_lowercase"`
	RequireNumbers       bool    `json:"require_numbers"`
	RequireSpecial       bool    `json:"require_special"`
	AllowedSpecialChars  *string `json:"allowed_special_chars,omitempty"`
	MaxAgeDays           *int    `json:"max_age_days,omitempty"`            // days before password expires (0 = never)
	HistoryCount         *int    `json:"history_count,omitempty"`           // number of previous passwords to check (0 = none)
	MinAgeHours          *int    `json:"min_age_hours,omitempty"`           // minimum hours between password changes (0 = none)
	MinUniqueChars       *int    `json:"min_unique_chars,omitempty"`        // minimum unique characters required (0 = none)
	NoUsernameInPassword bool    `json:"no_username_in_password,omitempty"` // prevent username in password
	NoCommonPasswords    bool    `json:"no_common_passwords,omitempty"`     // check against common passwords list
	Description          *string `json:"description,omitempty"`
}

// RolePasswordPolicyResponse represents password policy in API responses
type RolePasswordPolicyResponse struct {
	ID                   string  `json:"id"`
	RoleID               string  `json:"role_id"`
	RoleName             string  `json:"role_name"`
	MinLength            int     `json:"min_length"`
	MaxLength            int     `json:"max_length"`
	RequireUppercase     bool    `json:"require_uppercase"`
	RequireLowercase     bool    `json:"require_lowercase"`
	RequireNumbers       bool    `json:"require_numbers"`
	RequireSpecial       bool    `json:"require_special"`
	AllowedSpecialChars  *string `json:"allowed_special_chars,omitempty"`
	MaxAgeDays           int     `json:"max_age_days"`
	HistoryCount         int     `json:"history_count"`
	MinAgeHours          int     `json:"min_age_hours"`
	MinUniqueChars       int     `json:"min_unique_chars"`
	NoUsernameInPassword bool    `json:"no_username_in_password"`
	NoCommonPasswords    bool    `json:"no_common_passwords"`
	Description          *string `json:"description,omitempty"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at,omitempty"`
	UpdatedAt            string  `json:"updated_at,omitempty"`
}

// RolePasswordPolicySummary represents a simplified view for listing
type RolePasswordPolicySummary struct {
	RoleID           string `json:"role_id"`
	RoleName         string `json:"role_name"`
	RolePriority     int    `json:"role_priority"`
	MinLength        int    `json:"min_length"`
	RequireUppercase bool   `json:"require_uppercase"`
	RequireLowercase bool   `json:"require_lowercase"`
	RequireNumbers   bool   `json:"require_numbers"`
	RequireSpecial   bool   `json:"require_special"`
	MaxAgeDays       int    `json:"max_age_days"`
	HistoryCount     int    `json:"history_count"`
	MinAgeHours      int    `json:"min_age_hours"`
	HasPolicy        bool   `json:"has_policy"`
}

// ============================================
// Password Policy Handlers
// ============================================

// HandleRolePasswordPoliciesRoute routes GET requests for all role password policies
func (s *Server) HandleRolePasswordPoliciesRoute(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, http.StatusForbidden, "Apenas root pode visualizar políticas de senha")
		return
	}

	s.HandleGetAllRolePasswordPolicies(w, r)
}

// HandleRolePasswordPolicyByIDRoute routes GET/PUT/DELETE requests for a specific role's password policy
func (s *Server) HandleRolePasswordPolicyByIDRoute(w http.ResponseWriter, r *http.Request) {
	// Extract role ID from path: /api/roles/{id}/password-policy
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
		respondError(w, http.StatusForbidden, "Apenas root pode gerenciar políticas de senha")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.HandleGetRolePasswordPolicy(w, r, roleID)
	case http.MethodPut:
		s.HandleUpdateRolePasswordPolicy(w, r, roleID)
	case http.MethodDelete:
		s.HandleDeleteRolePasswordPolicy(w, r, roleID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleGetAllRolePasswordPolicies returns all role password policies with role info
func (s *Server) HandleGetAllRolePasswordPolicies(w http.ResponseWriter, r *http.Request) {
	db := GetGlobalDB()
	if db == nil {
		respondError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Query all roles with their password policies (LEFT JOIN to include roles without policies)
	query := `
		SELECT
			r.id as role_id,
			r.name as role_name,
			r.priority as role_priority,
			COALESCE(rpp.id::text, '') as policy_id,
			COALESCE(rpp.min_length, 16) as min_length,
			COALESCE(rpp.max_length, 128) as max_length,
			COALESCE(rpp.require_uppercase, true) as require_uppercase,
			COALESCE(rpp.require_lowercase, true) as require_lowercase,
			COALESCE(rpp.require_numbers, true) as require_numbers,
			COALESCE(rpp.require_special, true) as require_special,
			rpp.allowed_special_chars,
			COALESCE(rpp.max_age_days, 0) as max_age_days,
			COALESCE(rpp.history_count, 0) as history_count,
			COALESCE(rpp.min_age_hours, 0) as min_age_hours,
			COALESCE(rpp.min_unique_chars, 0) as min_unique_chars,
			COALESCE(rpp.no_username_in_password, true) as no_username_in_password,
			COALESCE(rpp.no_common_passwords, true) as no_common_passwords,
			rpp.description,
			COALESCE(rpp.is_active, false) as is_active,
			CASE WHEN rpp.id IS NOT NULL THEN true ELSE false END as has_policy
		FROM roles r
		LEFT JOIN role_password_policies rpp ON r.id = rpp.role_id AND rpp.is_active = TRUE
		ORDER BY r.priority DESC, r.name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Error querying password policies: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar políticas de senha")
		return
	}
	defer rows.Close()

	var policies []RolePasswordPolicyResponse
	for rows.Next() {
		var policy RolePasswordPolicyResponse
		var rolePriority int
		var allowedSpecialChars, description sql.NullString
		var hasPolicy bool

		err := rows.Scan(
			&policy.RoleID,
			&policy.RoleName,
			&rolePriority,
			&policy.ID,
			&policy.MinLength,
			&policy.MaxLength,
			&policy.RequireUppercase,
			&policy.RequireLowercase,
			&policy.RequireNumbers,
			&policy.RequireSpecial,
			&allowedSpecialChars,
			&policy.MaxAgeDays,
			&policy.HistoryCount,
			&policy.MinAgeHours,
			&policy.MinUniqueChars,
			&policy.NoUsernameInPassword,
			&policy.NoCommonPasswords,
			&description,
			&policy.IsActive,
			&hasPolicy,
		)
		if err != nil {
			log.Printf("❌ Error scanning password policy: %v", err)
			continue
		}

		if allowedSpecialChars.Valid {
			policy.AllowedSpecialChars = &allowedSpecialChars.String
		}
		if description.Valid {
			policy.Description = &description.String
		}

		// Mark as active only if policy exists
		policy.IsActive = hasPolicy

		policies = append(policies, policy)
	}

	if err = rows.Err(); err != nil {
		log.Printf("❌ Error iterating password policies: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao processar políticas de senha")
		return
	}

	respondJSON(w, http.StatusOK, policies)
}

// HandleGetRolePasswordPolicy returns password policy for a specific role
func (s *Server) HandleGetRolePasswordPolicy(w http.ResponseWriter, r *http.Request, roleID string) {
	db := GetGlobalDB()
	if db == nil {
		respondError(w, http.StatusInternalServerError, "Database not available")
		return
	}

	// Check if role exists and get its name
	var roleName string
	var rolePriority int
	err := db.QueryRow("SELECT name, priority FROM roles WHERE id = $1", roleID).Scan(&roleName, &rolePriority)
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
			id, role_id, min_length, max_length,
			require_uppercase, require_lowercase, require_numbers, require_special,
			allowed_special_chars, max_age_days, history_count, min_age_hours,
			min_unique_chars, no_username_in_password, no_common_passwords,
			description, is_active, created_at, updated_at
		FROM role_password_policies
		WHERE role_id = $1 AND is_active = TRUE
	`

	var policy RolePasswordPolicyResponse
	var allowedSpecialChars, description sql.NullString
	var createdAt, updatedAt sql.NullTime

	err = db.QueryRow(query, roleID).Scan(
		&policy.ID,
		&policy.RoleID,
		&policy.MinLength,
		&policy.MaxLength,
		&policy.RequireUppercase,
		&policy.RequireLowercase,
		&policy.RequireNumbers,
		&policy.RequireSpecial,
		&allowedSpecialChars,
		&policy.MaxAgeDays,
		&policy.HistoryCount,
		&policy.MinAgeHours,
		&policy.MinUniqueChars,
		&policy.NoUsernameInPassword,
		&policy.NoCommonPasswords,
		&description,
		&policy.IsActive,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default policy if none exists for this role
			defaultSpecial := "!@#$%^&*()_+-=[]{}|;:,.<>?"
			policy = RolePasswordPolicyResponse{
				ID:                   "",
				RoleID:               roleID,
				RoleName:             roleName,
				MinLength:            16,
				MaxLength:            128,
				RequireUppercase:     true,
				RequireLowercase:     true,
				RequireNumbers:       true,
				RequireSpecial:       true,
				AllowedSpecialChars:  &defaultSpecial,
				MaxAgeDays:           0,
				HistoryCount:         0,
				MinAgeHours:          0,
				MinUniqueChars:       0,
				NoUsernameInPassword: true,
				NoCommonPasswords:    true,
				IsActive:             false,
			}
			respondJSON(w, http.StatusOK, policy)
			return
		}
		log.Printf("❌ Error querying password policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar política de senha")
		return
	}

	policy.RoleName = roleName
	if allowedSpecialChars.Valid {
		policy.AllowedSpecialChars = &allowedSpecialChars.String
	}
	if description.Valid {
		policy.Description = &description.String
	}
	if createdAt.Valid {
		policy.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		policy.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	respondJSON(w, http.StatusOK, policy)
}

// HandleUpdateRolePasswordPolicy updates or creates password policy for a role
func (s *Server) HandleUpdateRolePasswordPolicy(w http.ResponseWriter, r *http.Request, roleID string) {
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
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var req RolePasswordPolicyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		log.Printf("❌ Error decoding request: %v", err)
		respondError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	// Validate values
	if req.MinLength < 8 || req.MinLength > 128 {
		respondError(w, http.StatusBadRequest, "Tamanho mínimo de senha deve estar entre 8 e 128 caracteres")
		return
	}

	// Set default max_length if not provided
	if req.MaxLength == 0 {
		req.MaxLength = 128
	}
	if req.MaxLength < req.MinLength || req.MaxLength > 256 {
		respondError(w, http.StatusBadRequest, "Tamanho máximo de senha deve ser maior que o mínimo e no máximo 256")
		return
	}

	if req.MaxAgeDays != nil && (*req.MaxAgeDays < 0 || *req.MaxAgeDays > 365) {
		respondError(w, http.StatusBadRequest, "Dias de expiração deve estar entre 0 e 365 (0 = nunca expira)")
		return
	}

	if req.HistoryCount != nil && (*req.HistoryCount < 0 || *req.HistoryCount > 24) {
		respondError(w, http.StatusBadRequest, "Histórico de senhas deve estar entre 0 e 24 (0 = não verificar)")
		return
	}

	if req.MinAgeHours != nil && (*req.MinAgeHours < 0 || *req.MinAgeHours > 720) {
		respondError(w, http.StatusBadRequest, "Intervalo mínimo de mudança deve estar entre 0 e 720 horas (0 = sem limite)")
		return
	}

	if req.MinUniqueChars != nil && (*req.MinUniqueChars < 0 || *req.MinUniqueChars > 64) {
		respondError(w, http.StatusBadRequest, "Caracteres únicos mínimos deve estar entre 0 e 64")
		return
	}

	// Get old value for audit
	var oldPolicy *RolePasswordPolicyResponse
	oldPolicy = s.getExistingPolicy(db, roleID)

	// Upsert password policy using ON CONFLICT on role_id
	query := `
		INSERT INTO role_password_policies (
			role_id, min_length, max_length,
			require_uppercase, require_lowercase, require_numbers, require_special,
			allowed_special_chars, max_age_days, history_count, min_age_hours,
			min_unique_chars, no_username_in_password, no_common_passwords,
			description, is_active, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, TRUE, CURRENT_TIMESTAMP)
		ON CONFLICT (role_id) DO UPDATE SET
			min_length = EXCLUDED.min_length,
			max_length = EXCLUDED.max_length,
			require_uppercase = EXCLUDED.require_uppercase,
			require_lowercase = EXCLUDED.require_lowercase,
			require_numbers = EXCLUDED.require_numbers,
			require_special = EXCLUDED.require_special,
			allowed_special_chars = EXCLUDED.allowed_special_chars,
			max_age_days = EXCLUDED.max_age_days,
			history_count = EXCLUDED.history_count,
			min_age_hours = EXCLUDED.min_age_hours,
			min_unique_chars = EXCLUDED.min_unique_chars,
			no_username_in_password = EXCLUDED.no_username_in_password,
			no_common_passwords = EXCLUDED.no_common_passwords,
			description = EXCLUDED.description,
			is_active = TRUE,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	// Handle nullable fields with concrete int values
	var maxAgeDays, historyCount, minAgeHours, minUniqueChars int
	if req.MaxAgeDays != nil {
		maxAgeDays = *req.MaxAgeDays
	} else {
		maxAgeDays = 0
	}
	if req.HistoryCount != nil {
		historyCount = *req.HistoryCount
	} else {
		historyCount = 0
	}
	if req.MinAgeHours != nil {
		minAgeHours = *req.MinAgeHours
	} else {
		minAgeHours = 0
	}
	if req.MinUniqueChars != nil {
		minUniqueChars = *req.MinUniqueChars
	} else {
		minUniqueChars = 0
	}

	var policyID string
	err = db.QueryRow(
		query,
		roleID,
		req.MinLength,
		req.MaxLength,
		req.RequireUppercase,
		req.RequireLowercase,
		req.RequireNumbers,
		req.RequireSpecial,
		req.AllowedSpecialChars,
		maxAgeDays,
		historyCount,
		minAgeHours,
		minUniqueChars,
		req.NoUsernameInPassword,
		req.NoCommonPasswords,
		req.Description,
	).Scan(&policyID)

	if err != nil {
		log.Printf("❌ Error upserting password policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao salvar política de senha")
		return
	}

	log.Printf("✅ User %s updated password policy for role %s (ID: %s)", claims.Username, roleName, policyID)

	// Log to audit
	if s.auditStore != nil {
		userAgent := r.UserAgent()
		method := r.Method
		path := r.URL.Path
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "role_password_policy",
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

	// Return the updated policy
	response := RolePasswordPolicyResponse{
		ID:                   policyID,
		RoleID:               roleID,
		RoleName:             roleName,
		MinLength:            req.MinLength,
		MaxLength:            req.MaxLength,
		RequireUppercase:     req.RequireUppercase,
		RequireLowercase:     req.RequireLowercase,
		RequireNumbers:       req.RequireNumbers,
		RequireSpecial:       req.RequireSpecial,
		AllowedSpecialChars:  req.AllowedSpecialChars,
		MaxAgeDays:           maxAgeDays,
		HistoryCount:         historyCount,
		MinAgeHours:          minAgeHours,
		MinUniqueChars:       minUniqueChars,
		NoUsernameInPassword: req.NoUsernameInPassword,
		NoCommonPasswords:    req.NoCommonPasswords,
		Description:          req.Description,
		IsActive:             true,
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleDeleteRolePasswordPolicy deactivates password policy for a role (soft delete)
func (s *Server) HandleDeleteRolePasswordPolicy(w http.ResponseWriter, r *http.Request, roleID string) {
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

	// Get old value for audit
	oldPolicy := s.getExistingPolicy(db, roleID)
	if oldPolicy == nil {
		respondError(w, http.StatusNotFound, "Política de senha não encontrada para este role")
		return
	}

	// Soft delete by setting is_active to false
	query := `
		UPDATE role_password_policies
		SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE role_id = $1 AND is_active = TRUE
	`

	result, err := db.Exec(query, roleID)
	if err != nil {
		log.Printf("❌ Error deleting password policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao remover política de senha")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, http.StatusNotFound, "Política de senha não encontrada")
		return
	}

	log.Printf("✅ User %s deleted password policy for role %s", claims.Username, roleName)

	// Log to audit
	if s.auditStore != nil {
		userAgent := r.UserAgent()
		method := r.Method
		path := r.URL.Path
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "delete",
			Resource:      "role_password_policy",
			ResourceID:    roleID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      oldPolicy,
			NewValue:      nil,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     &userAgent,
			RequestMethod: &method,
			RequestPath:   &path,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Política de senha removida com sucesso. O role usará as configurações globais.",
	})
}

// getExistingPolicy is a helper to retrieve existing policy for audit logging
func (s *Server) getExistingPolicy(db *sql.DB, roleID string) *RolePasswordPolicyResponse {
	query := `
		SELECT
			id, role_id, min_length, max_length,
			require_uppercase, require_lowercase, require_numbers, require_special,
			allowed_special_chars, max_age_days, history_count, min_age_hours,
			min_unique_chars, no_username_in_password, no_common_passwords,
			description, is_active
		FROM role_password_policies
		WHERE role_id = $1 AND is_active = TRUE
	`

	var policy RolePasswordPolicyResponse
	var allowedSpecialChars, description sql.NullString

	err := db.QueryRow(query, roleID).Scan(
		&policy.ID,
		&policy.RoleID,
		&policy.MinLength,
		&policy.MaxLength,
		&policy.RequireUppercase,
		&policy.RequireLowercase,
		&policy.RequireNumbers,
		&policy.RequireSpecial,
		&allowedSpecialChars,
		&policy.MaxAgeDays,
		&policy.HistoryCount,
		&policy.MinAgeHours,
		&policy.MinUniqueChars,
		&policy.NoUsernameInPassword,
		&policy.NoCommonPasswords,
		&description,
		&policy.IsActive,
	)

	if err != nil {
		return nil
	}

	if allowedSpecialChars.Valid {
		policy.AllowedSpecialChars = &allowedSpecialChars.String
	}
	if description.Valid {
		policy.Description = &description.String
	}

	return &policy
}
