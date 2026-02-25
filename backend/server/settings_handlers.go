package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"Open-Generic-Hub/backend/store"
)

// HandleGetSettings returns all system settings
// GET /api/settings
// Public endpoint (some settings might need to be filtered if they are sensitive, but for now assuming all are public branding/labels)
// Or maybe restricted to authenticated users? The requirements say "available to all users".
// Branding like "Logo" needs to be available even before login potentially?
// For now, let's make it public or at least accessible.
func (s *Server) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	settings, err := s.settingsStore.GetSystemSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve settings")
		return
	}

	// Settings rarely change — allow browser to cache for 5 minutes
	w.Header().Set("Cache-Control", "private, max-age=300")
	respondJSON(w, http.StatusOK, settings)
}

// HandleUpdateSettings updates system settings
// PUT /api/settings
// Restricted to Admin/Root
type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings"`
}

func (s *Server) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user info for audit logging
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get old values for audit logging
	oldSettings, _ := s.settingsStore.GetSystemSettings()

	// Categorize settings for audit
	brandingChanges := make(map[string]interface{})
	labelsChanges := make(map[string]interface{})
	otherChanges := make(map[string]interface{})

	// Security validation and update
	for key, value := range req.Settings {
		if err := validateSetting(key, value); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid setting %s: %v", key, err))
			return
		}

		if err := s.settingsStore.UpdateSystemSetting(key, value); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update settings")
			return
		}

		// Categorize for audit log
		oldValue := oldSettings[key]
		changeInfo := map[string]interface{}{
			"old": oldValue,
			"new": value,
		}

		if strings.HasPrefix(key, "branding.") {
			// Não logamos o valor completo de imagens base64 para evitar logs enormes
			if strings.Contains(key, "logo") || strings.Contains(key, "Logo") {
				if len(value) > 100 {
					changeInfo["new"] = "[imagem base64 atualizada]"
				}
				if len(oldValue) > 100 {
					changeInfo["old"] = "[imagem base64 anterior]"
				}
			}
			brandingChanges[key] = changeInfo
		} else if strings.HasPrefix(key, "labels.") {
			labelsChanges[key] = changeInfo
		} else {
			otherChanges[key] = changeInfo
		}
	}

	// Log audit entries for each category of changes
	if s.auditStore != nil {
		userAgent := r.UserAgent()
		method := r.Method
		path := r.URL.Path

		if len(brandingChanges) > 0 {
			log.Printf("✅ User %s updated branding settings", claims.Username)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "settings_branding",
				ResourceID:    "branding",
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      nil,
				NewValue:      brandingChanges,
				Status:        "success",
				IPAddress:     getIPAddress(r),
				UserAgent:     &userAgent,
				RequestMethod: &method,
				RequestPath:   &path,
			})
		}

		if len(labelsChanges) > 0 {
			log.Printf("✅ User %s updated labels settings", claims.Username)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "settings_labels",
				ResourceID:    "labels",
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      nil,
				NewValue:      labelsChanges,
				Status:        "success",
				IPAddress:     getIPAddress(r),
				UserAgent:     &userAgent,
				RequestMethod: &method,
				RequestPath:   &path,
			})
		}

		if len(otherChanges) > 0 {
			log.Printf("✅ User %s updated system settings", claims.Username)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "settings_system",
				ResourceID:    "system",
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      nil,
				NewValue:      otherChanges,
				Status:        "success",
				IPAddress:     getIPAddress(r),
				UserAgent:     &userAgent,
				RequestMethod: &method,
				RequestPath:   &path,
			})
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "settings updated successfully"})
}

func validateSetting(key, value string) error {
	// 1. Length check
	// Base64 images for branding can be large.
	// 500KB image ~ 670KB Base64.
	// DB TEXT column is effectively unlimited (1GB).
	// Let's set a safe upper limit for branding (e.g. 1MB) and keep strict limit for others.
	limit := 2000
	if strings.HasPrefix(key, "branding.") {
		limit = 1_000_000 // 1MB for logos
	}

	if len(key) > 100 || len(value) > limit {
		return fmt.Errorf("value too long for key %s (max %d)", key, limit)
	}

	// 2. Theme color validation
	if strings.HasPrefix(key, "theme.") && strings.Contains(strings.ToLower(key), "color") {
		// Simple hex check
		isHex := regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}){1,2}$`).MatchString(value)
		if !isHex {
			return fmt.Errorf("invalid color format (expected hex)")
		}
	}

	// 3. Simple XSS check (very basic, relying mostly on frontend escaping)
	// Reject scripts
	if strings.Contains(value, "<script") || strings.Contains(value, "javascript:") {
		return fmt.Errorf("potential XSS detected")
	}

	return nil
}
