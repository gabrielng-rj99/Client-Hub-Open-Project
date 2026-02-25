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
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"Open-Generic-Hub/backend/store"
)

// ============================================
// Request/Response Types for Security Config
// ============================================

// SecurityConfigRequest represents the full security configuration request
type SecurityConfigRequest struct {
	// Lock Levels Configuration
	LockLevel1Attempts int `json:"lock_level_1_attempts"`
	LockLevel1Duration int `json:"lock_level_1_duration"` // seconds
	LockLevel2Attempts int `json:"lock_level_2_attempts"`
	LockLevel2Duration int `json:"lock_level_2_duration"` // seconds
	LockLevel3Attempts int `json:"lock_level_3_attempts"`
	LockLevel3Duration int `json:"lock_level_3_duration"` // seconds
	LockManualAttempts int `json:"lock_manual_attempts"`  // attempts before permanent lock

	// Password Policy
	PasswordMinLength      int  `json:"password_min_length"`
	PasswordRequireUpper   bool `json:"password_require_uppercase"`
	PasswordRequireLower   bool `json:"password_require_lowercase"`
	PasswordRequireNumbers bool `json:"password_require_numbers"`
	PasswordRequireSpecial bool `json:"password_require_special"`

	// Session Configuration
	SessionDuration      int `json:"session_duration"`       // minutes
	RefreshTokenDuration int `json:"refresh_token_duration"` // minutes

	// Rate Limiting
	RateLimit int `json:"rate_limit"` // requests per second
	RateBurst int `json:"rate_burst"` // burst limit

	// Audit Configuration
	AuditRetentionDays int  `json:"audit_retention_days"`
	AuditLogReads      bool `json:"audit_log_reads"`

	// Notifications (for future use)
	NotificationEmail string `json:"notification_email"`
	NotificationPhone string `json:"notification_phone"`
}

// SecurityConfigResponse represents the full security configuration response
type SecurityConfigResponse struct {
	// Lock Levels Configuration
	LockLevel1Attempts int `json:"lock_level_1_attempts"`
	LockLevel1Duration int `json:"lock_level_1_duration"`
	LockLevel2Attempts int `json:"lock_level_2_attempts"`
	LockLevel2Duration int `json:"lock_level_2_duration"`
	LockLevel3Attempts int `json:"lock_level_3_attempts"`
	LockLevel3Duration int `json:"lock_level_3_duration"`
	LockManualAttempts int `json:"lock_manual_attempts"`

	// Password Policy
	PasswordMinLength      int  `json:"password_min_length"`
	PasswordRequireUpper   bool `json:"password_require_uppercase"`
	PasswordRequireLower   bool `json:"password_require_lowercase"`
	PasswordRequireNumbers bool `json:"password_require_numbers"`
	PasswordRequireSpecial bool `json:"password_require_special"`

	// Session Configuration
	SessionDuration      int `json:"session_duration"`
	RefreshTokenDuration int `json:"refresh_token_duration"`

	// Rate Limiting
	RateLimit int `json:"rate_limit"`
	RateBurst int `json:"rate_burst"`

	// Audit Configuration
	AuditRetentionDays int  `json:"audit_retention_days"`
	AuditLogReads      bool `json:"audit_log_reads"`

	// Notifications
	NotificationEmail string `json:"notification_email"`
	NotificationPhone string `json:"notification_phone"`
}

// ============================================
// Security Config Handlers
// ============================================

// HandleSecurityConfigRoute routes security config requests
func (s *Server) HandleSecurityConfigRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.HandleGetSecurityConfig(w, r)
	case http.MethodPut:
		s.HandleUpdateSecurityConfig(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleGetSecurityConfig handles GET /api/settings/security
func (s *Server) HandleGetSecurityConfig(w http.ResponseWriter, r *http.Request) {
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can view security config - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		respondError(w, http.StatusForbidden, "Apenas usuários root podem acessar configurações de segurança")
		return
	}

	// Get all settings
	settings, err := s.settingsStore.GetSystemSettings()
	if err != nil {
		log.Printf("❌ Error loading security settings: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao carregar configurações")
		return
	}

	// Build response with defaults
	response := SecurityConfigResponse{
		// Lock Levels - defaults
		LockLevel1Attempts: 3,
		LockLevel1Duration: 300, // 5 min
		LockLevel2Attempts: 5,
		LockLevel2Duration: 900, // 15 min
		LockLevel3Attempts: 10,
		LockLevel3Duration: 3600, // 60 min
		LockManualAttempts: 15,

		// Password Policy - defaults
		PasswordMinLength:      16,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireNumbers: true,
		PasswordRequireSpecial: true,

		// Session - defaults
		SessionDuration:      60,    // 1 hour
		RefreshTokenDuration: 10080, // 7 days

		// Rate Limiting - defaults
		RateLimit: 5,
		RateBurst: 10,

		// Audit - defaults
		AuditRetentionDays: 365,
		AuditLogReads:      false,

		// Notifications
		NotificationEmail: "",
		NotificationPhone: "",
	}

	// Override with stored values
	for key, value := range settings {
		switch key {
		// Lock Levels
		case "security.lock_level_1_attempts":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel1Attempts = v
			}
		case "security.lock_level_1_duration":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel1Duration = v
			}
		case "security.lock_level_2_attempts":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel2Attempts = v
			}
		case "security.lock_level_2_duration":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel2Duration = v
			}
		case "security.lock_level_3_attempts":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel3Attempts = v
			}
		case "security.lock_level_3_duration":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockLevel3Duration = v
			}
		case "security.lock_level_manual_attempts":
			if v, err := strconv.Atoi(value); err == nil {
				response.LockManualAttempts = v
			}

		// Password Policy
		case "security.password_min_length":
			if v, err := strconv.Atoi(value); err == nil {
				response.PasswordMinLength = v
			}
		case "security.password_require_uppercase":
			response.PasswordRequireUpper = value == "true"
		case "security.password_require_lowercase":
			response.PasswordRequireLower = value == "true"
		case "security.password_require_numbers":
			response.PasswordRequireNumbers = value == "true"
		case "security.password_require_special":
			response.PasswordRequireSpecial = value == "true"

		// Session
		case "security.session_duration":
			if v, err := strconv.Atoi(value); err == nil {
				response.SessionDuration = v
			}
		case "security.refresh_token_duration":
			if v, err := strconv.Atoi(value); err == nil {
				response.RefreshTokenDuration = v
			}

		// Rate Limiting
		case "security.rate_limit":
			if v, err := strconv.Atoi(value); err == nil {
				response.RateLimit = v
			}
		case "security.rate_burst":
			if v, err := strconv.Atoi(value); err == nil {
				response.RateBurst = v
			}

		// Audit
		case "audit.retention_days":
			if v, err := strconv.Atoi(value); err == nil {
				response.AuditRetentionDays = v
			}
		case "audit.log_reads":
			response.AuditLogReads = value == "true"

		// Notifications
		case "notifications.email":
			response.NotificationEmail = value
		case "notifications.phone":
			response.NotificationPhone = value
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleUpdateSecurityConfig handles PUT /api/settings/security
func (s *Server) HandleUpdateSecurityConfig(w http.ResponseWriter, r *http.Request) {
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can update security config - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		role, _ := s.roleStore.GetUserRole(claims.UserID)
		log.Printf("🚫 User %s (role: %s) denied access to update security config", claims.Username, role)
		respondError(w, http.StatusForbidden, "Apenas usuários root podem alterar configurações de segurança")
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, 8192)

	// Parse request
	var req SecurityConfigRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	// Validate Lock Levels
	if req.LockLevel1Attempts < 1 || req.LockLevel1Attempts > 20 {
		respondError(w, http.StatusBadRequest, "Tentativas do nível 1 devem estar entre 1 e 20")
		return
	}
	if req.LockLevel1Duration < 60 || req.LockLevel1Duration > 3600 {
		respondError(w, http.StatusBadRequest, "Duração do nível 1 deve estar entre 60 e 3600 segundos")
		return
	}
	if req.LockLevel2Attempts < req.LockLevel1Attempts {
		respondError(w, http.StatusBadRequest, "Tentativas do nível 2 devem ser maiores que nível 1")
		return
	}
	if req.LockLevel2Duration < req.LockLevel1Duration {
		respondError(w, http.StatusBadRequest, "Duração do nível 2 deve ser maior que nível 1")
		return
	}
	if req.LockLevel3Attempts < req.LockLevel2Attempts {
		respondError(w, http.StatusBadRequest, "Tentativas do nível 3 devem ser maiores que nível 2")
		return
	}
	if req.LockLevel3Duration < req.LockLevel2Duration {
		respondError(w, http.StatusBadRequest, "Duração do nível 3 deve ser maior que nível 2")
		return
	}
	if req.LockManualAttempts < req.LockLevel3Attempts {
		respondError(w, http.StatusBadRequest, "Tentativas para bloqueio manual devem ser maiores que nível 3")
		return
	}

	// Validate Password Policy
	if req.PasswordMinLength < 8 || req.PasswordMinLength > 128 {
		respondError(w, http.StatusBadRequest, "Tamanho mínimo da senha deve estar entre 8 e 128")
		return
	}

	// Validate Session
	if req.SessionDuration < 5 || req.SessionDuration > 1440 {
		respondError(w, http.StatusBadRequest, "Duração da sessão deve estar entre 5 e 1440 minutos")
		return
	}
	if req.RefreshTokenDuration < 60 || req.RefreshTokenDuration > 43200 {
		respondError(w, http.StatusBadRequest, "Duração do refresh token deve estar entre 60 e 43200 minutos")
		return
	}

	// Validate Rate Limiting
	if req.RateLimit < 1 || req.RateLimit > 100 {
		respondError(w, http.StatusBadRequest, "Rate limit deve estar entre 1 e 100")
		return
	}
	if req.RateBurst < 1 || req.RateBurst > 200 {
		respondError(w, http.StatusBadRequest, "Rate burst deve estar entre 1 e 200")
		return
	}

	// Validate Audit
	if req.AuditRetentionDays < 30 || req.AuditRetentionDays > 3650 {
		respondError(w, http.StatusBadRequest, "Retenção de auditoria deve estar entre 30 e 3650 dias")
		return
	}

	// Get old values for audit log
	oldSettings, _ := s.settingsStore.GetSystemSettings()

	// Save all settings
	settingsToSave := map[string]string{
		// Lock Levels
		"security.lock_level_1_attempts":      strconv.Itoa(req.LockLevel1Attempts),
		"security.lock_level_1_duration":      strconv.Itoa(req.LockLevel1Duration),
		"security.lock_level_2_attempts":      strconv.Itoa(req.LockLevel2Attempts),
		"security.lock_level_2_duration":      strconv.Itoa(req.LockLevel2Duration),
		"security.lock_level_3_attempts":      strconv.Itoa(req.LockLevel3Attempts),
		"security.lock_level_3_duration":      strconv.Itoa(req.LockLevel3Duration),
		"security.lock_level_manual_attempts": strconv.Itoa(req.LockManualAttempts),

		// Password Policy
		"security.password_min_length":        strconv.Itoa(req.PasswordMinLength),
		"security.password_require_uppercase": strconv.FormatBool(req.PasswordRequireUpper),
		"security.password_require_lowercase": strconv.FormatBool(req.PasswordRequireLower),
		"security.password_require_numbers":   strconv.FormatBool(req.PasswordRequireNumbers),
		"security.password_require_special":   strconv.FormatBool(req.PasswordRequireSpecial),

		// Session
		"security.session_duration":       strconv.Itoa(req.SessionDuration),
		"security.refresh_token_duration": strconv.Itoa(req.RefreshTokenDuration),

		// Rate Limiting
		"security.rate_limit": strconv.Itoa(req.RateLimit),
		"security.rate_burst": strconv.Itoa(req.RateBurst),

		// Audit
		"audit.retention_days": strconv.Itoa(req.AuditRetentionDays),
		"audit.log_reads":      strconv.FormatBool(req.AuditLogReads),
	}

	// Save notifications if provided
	if req.NotificationEmail != "" {
		settingsToSave["notifications.email"] = req.NotificationEmail
	}
	if req.NotificationPhone != "" {
		settingsToSave["notifications.phone"] = req.NotificationPhone
	}

	// Save all settings
	for key, value := range settingsToSave {
		if err := s.settingsStore.UpdateSystemSetting(key, value); err != nil {
			log.Printf("❌ Error saving setting %s: %v", key, err)
		}
	}

	log.Printf("✅ Root user %s updated security configuration", claims.Username)

	// Log to audit
	if s.auditStore != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "security_config",
			ResourceID:    "system",
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      oldSettings,
			NewValue:      settingsToSave,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Configurações de segurança atualizadas com sucesso",
	})
}

// ============================================
// Password Policy Handlers
// ============================================

// PasswordPolicyResponse represents the password policy configuration
type PasswordPolicyResponse struct {
	MinLength      int  `json:"min_length"`
	RequireUpper   bool `json:"require_uppercase"`
	RequireLower   bool `json:"require_lowercase"`
	RequireNumbers bool `json:"require_numbers"`
	RequireSpecial bool `json:"require_special"`
}

// HandleGetPasswordPolicy handles GET /api/settings/password-policy
// This endpoint is accessible to all authenticated users to show password requirements
func (s *Server) HandleGetPasswordPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate token (any authenticated user can see password policy)
	tokenString := extractTokenFromHeader(r)
	_, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Get settings
	settings, err := s.settingsStore.GetSystemSettings()
	if err != nil {
		log.Printf("❌ Error loading password policy: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao carregar política de senha")
		return
	}

	// Build response with defaults
	response := PasswordPolicyResponse{
		MinLength:      16,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumbers: true,
		RequireSpecial: true,
	}

	// Override with stored values
	for key, value := range settings {
		switch key {
		case "security.password_min_length":
			if v, err := strconv.Atoi(value); err == nil {
				response.MinLength = v
			}
		case "security.password_require_uppercase":
			response.RequireUpper = value == "true"
		case "security.password_require_lowercase":
			response.RequireLower = value == "true"
		case "security.password_require_numbers":
			response.RequireNumbers = value == "true"
		case "security.password_require_special":
			response.RequireSpecial = value == "true"
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// HandlePasswordPolicyRoute routes password policy requests
func (s *Server) HandlePasswordPolicyRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.HandleGetPasswordPolicy(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
