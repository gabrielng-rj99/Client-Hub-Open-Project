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
	"log"
	"net/http"
	"strings"
	"time"
)

// ============= HEALTH HANDLER =============

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	db := GetGlobalDB()
	if db == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "unhealthy",
			"message":   "Database not connected",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "unhealthy",
			"message":   "Database connection failed",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// standardMiddleware applies all standard security middlewares
func (s *Server) standardMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// Logging should be the outermost to capture everything including CORS/RateLimit effects if possible,
	// but usually CORS is outermost for browser preflight.
	// Order: Logging -> CORS -> SecurityHeaders -> RateLimit -> Handler
	return s.loggingMiddleware(s.corsMiddleware(s.securityHeadersMiddleware(s.rateLimitMiddleware(next))))
}

// ============= ROUTER SETUP =============

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractTokenFromHeader(r)
		if tokenString == "" {
			log.Printf("Requisição sem token para %s %s", r.Method, r.URL.Path)
			respondError(w, http.StatusUnauthorized, "Token não fornecido")
			return
		}

		claims, err := ValidateJWT(tokenString, s.userStore)
		if err != nil {
			log.Printf("Token inválido ou expirado para %s %s: %v", r.Method, r.URL.Path, err)
			respondError(w, http.StatusUnauthorized, "Token inválido ou expirado. Faça login novamente.")
			return
		}

		// Inject user info for logging (role fetched from DB once here)
		setRequestUser(r, claims, s.roleStore)

		next(w, r)
	}
}

// adminOnlyMiddleware restricts access to admin and root users only
// Must be used after authMiddleware
// NOTE: Role is checked via DB lookup, not JWT claims
func (s *Server) adminOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractTokenFromHeader(r)
		claims, err := ValidateJWT(tokenString, s.userStore)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Token inválido")
			return
		}

		// Check role via DB lookup
		isAdminOrRoot, err := s.roleStore.IsUserAdminOrRoot(claims.UserID)
		if err != nil {
			log.Printf("❌ Error checking user role: %v", err)
			respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
			return
		}

		if !isAdminOrRoot {
			role, _ := s.roleStore.GetUserRole(claims.UserID)
			log.Printf("🚫 Acesso negado: usuário %s (role: %s) tentou acessar recurso administrativo %s %s",
				claims.Username, role, r.Method, r.URL.Path)
			respondError(w, http.StatusForbidden, "Acesso negado. Apenas administradores podem acessar este recurso.")
			return
		}

		next(w, r)
	}
}

// rootOnlyMiddleware restricts access to root users only
// Must be used after authMiddleware
// NOTE: Role is checked via DB lookup, not JWT claims
func (s *Server) rootOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractTokenFromHeader(r)
		claims, err := ValidateJWT(tokenString, s.userStore)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Token inválido")
			return
		}

		// Check role via DB lookup
		isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
		if err != nil {
			log.Printf("❌ Error checking user role: %v", err)
			respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
			return
		}

		if !isRoot {
			role, _ := s.roleStore.GetUserRole(claims.UserID)
			log.Printf("🚫 Acesso negado: usuário %s (role: %s) tentou acessar recurso exclusivo de root %s %s",
				claims.Username, role, r.Method, r.URL.Path)
			respondError(w, http.StatusForbidden, "Acesso negado. Apenas usuários root podem acessar este recurso.")
			return
		}

		next(w, r)
	}
}

// Handler returns an http.Handler with all routes configured (for testing)
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return mux
}

// SetupRoutes configures routes on the default ServeMux (for production)
func (s *Server) SetupRoutes() {
	s.registerRoutes(http.DefaultServeMux)
}

// registerRoutes registers all routes on the provided ServeMux
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", s.standardMiddleware(s.handleHealth))

	// Auth
	mux.HandleFunc("/api/login", s.standardMiddleware(s.handleLogin))
	mux.HandleFunc("/api/logout", s.standardMiddleware(s.authMiddleware(s.handleLogout)))
	mux.HandleFunc("/api/refresh-token", s.standardMiddleware(s.handleRefreshToken))

	// Users - admin/root only for management operations
	// IMPORTANT: /api/users must be registered BEFORE /api/users/ to match exact path first
	mux.HandleFunc("/api/users", s.standardMiddleware(s.authMiddleware(s.adminOnlyMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Se o path é exatamente /api/users (sem ID), chama handleUsers
		if r.URL.Path == "/api/users" {
			s.handleUsers(w, r)
		} else {
			// Caso contrário, trata como /api/users/{username}
			if strings.HasSuffix(r.URL.Path, "/block") {
				s.handleUserBlock(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/unlock") {
				s.handleUserUnlock(w, r)
			} else {
				s.handleUserByUsername(w, r)
			}
		}
	}))))
	mux.HandleFunc("/api/users/", s.standardMiddleware(s.authMiddleware(s.adminOnlyMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/block") {
			s.handleUserBlock(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/unlock") {
			s.handleUserUnlock(w, r)
		} else {
			s.handleUserByUsername(w, r)
		}
	}))))

	// Clients (Clients)
	// /api/clients -> /api/clients
	mux.HandleFunc("/api/clients/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/archive") {
			s.handleClientArchive(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/unarchive") {
			s.handleClientUnarchive(w, r)
		} else if strings.Contains(r.URL.Path, "/affiliates") {
			s.handleClientAffiliates(w, r)
		} else {
			s.handleClientByID(w, r)
		}
	})))
	mux.HandleFunc("/api/clients", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/clients" {
			s.handleClients(w, r)
		} else {
			if strings.HasSuffix(r.URL.Path, "/archive") {
				s.handleClientArchive(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/unarchive") {
				s.handleClientUnarchive(w, r)
			} else if strings.Contains(r.URL.Path, "/affiliates") {
				s.handleClientAffiliates(w, r)
			} else {
				s.handleClientByID(w, r)
			}
		}
	})))

	// Affiliates
	// /api/affiliates
	mux.HandleFunc("/api/affiliates/", s.standardMiddleware(s.authMiddleware(s.handleAffiliateByID)))

	// Contracts (Contracts)
	mux.HandleFunc("/api/contracts/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/archive") {
			s.handleContractArchive(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/unarchive") {
			s.handleContractUnarchive(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/financial") {
			s.handleContractFinancial(w, r)
		} else {
			s.handleContractByID(w, r)
		}
	})))
	mux.HandleFunc("/api/contracts", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/contracts" {
			s.handleContracts(w, r)
		} else {
			if strings.HasSuffix(r.URL.Path, "/archive") {
				s.handleContractArchive(w, r)
			} else if strings.HasSuffix(r.URL.Path, "/unarchive") {
				s.handleContractUnarchive(w, r)
			} else {
				s.handleContractByID(w, r)
			}
		}
	})))

	// Categories
	mux.HandleFunc("/api/categories/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/subcategories") {
			s.handleCategorySubcategories(w, r)
		} else {
			s.handleCategoryByID(w, r)
		}
	})))
	mux.HandleFunc("/api/categories", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/categories" {
			s.handleCategories(w, r)
		} else {
			if strings.Contains(r.URL.Path, "/subcategories") {
				s.handleCategorySubcategories(w, r)
			} else {
				s.handleCategoryByID(w, r)
			}
		}
	})))

	// Subcategories (Lines)
	mux.HandleFunc("/api/subcategories/", s.standardMiddleware(s.authMiddleware(s.handleSubcategoryByID)))
	mux.HandleFunc("/api/subcategories", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/subcategories" {
			s.handleSubcategories(w, r)
		} else {
			s.handleSubcategoryByID(w, r)
		}
	})))

	// Dashboard Counts - Aggregated counts endpoint (replaces loading all records for counts)
	mux.HandleFunc("/api/dashboard/counts", s.standardMiddleware(s.authMiddleware(s.handleDashboardCounts)))

	// Client Counts - Client status breakdown for filter buttons
	mux.HandleFunc("/api/clients/counts", s.standardMiddleware(s.authMiddleware(s.handleClientCounts)))

	// Financial - Contract financial management
	mux.HandleFunc("/api/financial/summary", s.standardMiddleware(s.authMiddleware(s.handleFinancialSummary)))
	mux.HandleFunc("/api/financial/detailed-summary", s.standardMiddleware(s.authMiddleware(s.handleFinancialDetailedSummary)))
	mux.HandleFunc("/api/financial/upcoming", s.standardMiddleware(s.authMiddleware(s.handleUpcomingFinancial)))
	mux.HandleFunc("/api/financial/overdue", s.standardMiddleware(s.authMiddleware(s.handleOverdueFinancial)))
	mux.HandleFunc("/api/financial/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		s.handleFinancialByID(w, r)
	})))
	mux.HandleFunc("/api/financial", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/financial" {
			s.handleFinancial(w, r)
		} else {
			s.handleFinancialByID(w, r)
		}
	})))

	// Audit Logs (only accessible to root)
	mux.HandleFunc("/api/audit-logs/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Apenas root pode acessar - check via DB
		claims, err := ValidateJWT(extractTokenFromHeader(r), s.userStore)
		if err != nil {
			respondError(w, http.StatusForbidden, "Apenas root pode acessar logs de auditoria")
			return
		}
		isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
		if err != nil || !isRoot {
			respondError(w, http.StatusForbidden, "Apenas root pode acessar logs de auditoria")
			return
		}

		if strings.HasSuffix(r.URL.Path, "/export") {
			s.handleAuditLogsExport(w, r)
		} else if strings.Contains(r.URL.Path, "/resource/") {
			// client -> resource
			s.handleAuditLogsByResource(w, r)
		} else {
			s.handleAuditLogDetail(w, r)
		}
	})))
	mux.HandleFunc("/api/audit-logs", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Apenas root pode acessar - check via DB
		claims, err := ValidateJWT(extractTokenFromHeader(r), s.userStore)
		if err != nil {
			respondError(w, http.StatusForbidden, "Apenas root pode acessar logs de auditoria")
			return
		}
		isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
		if err != nil || !isRoot {
			respondError(w, http.StatusForbidden, "Apenas root pode acessar logs de auditoria")
			return
		}

		if r.URL.Path == "/api/audit-logs" {
			s.handleAuditLogs(w, r)
		} else {
			if strings.HasSuffix(r.URL.Path, "/export") {
				s.handleAuditLogsExport(w, r)
			} else if strings.Contains(r.URL.Path, "/resource/") {
				s.handleAuditLogsByResource(w, r)
			} else {
				s.handleAuditLogDetail(w, r)
			}
		}
	})))

	// Settings routes - Root Only
	mux.HandleFunc("/api/settings", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.handleSettingsRoute))))

	// User Theme routes - Authenticated users (permission checked in handler)
	mux.HandleFunc("/api/user/theme", s.standardMiddleware(s.authMiddleware(s.HandleUserThemeRoute)))

	// Theme Permissions routes - Root Only
	mux.HandleFunc("/api/settings/theme-permissions", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleThemePermissionsRoute))))

	// Global Theme routes - Root Only
	mux.HandleFunc("/api/settings/global-theme", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleGlobalThemeRoute))))

	// Allowed Themes routes - Authenticated users can read, Root Only can write
	mux.HandleFunc("/api/settings/allowed-themes", s.standardMiddleware(s.authMiddleware(s.HandleAllowedThemesRoute)))

	// System Config routes - Root Only
	mux.HandleFunc("/api/settings/system-config", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleSystemConfigRoute))))

	// Dashboard Config routes - Admin+ (dashboard display settings)
	mux.HandleFunc("/api/system-config/dashboard", s.standardMiddleware(s.authMiddleware(s.adminOnlyMiddleware(s.HandleDashboardConfigRoute))))

	// Security Config routes - Root Only (expanded security settings)
	mux.HandleFunc("/api/settings/security", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleSecurityConfigRoute))))

	// Password Policy route - Authenticated users can read (to show requirements)
	mux.HandleFunc("/api/settings/password-policy", s.standardMiddleware(s.authMiddleware(s.HandlePasswordPolicyRoute)))

	// Roles routes - Root Only for create/update/delete, Admin+ for read
	mux.HandleFunc("/api/roles", s.standardMiddleware(s.authMiddleware(s.HandleRolesRoute)))

	// Permissions routes - Admin+ for read
	mux.HandleFunc("/api/permissions", s.standardMiddleware(s.authMiddleware(s.HandlePermissionsRoute)))

	// User Permissions routes - Authenticated users can get their own permissions
	mux.HandleFunc("/api/user/permissions", s.standardMiddleware(s.authMiddleware(s.HandleUserPermissionsRoute)))
	mux.HandleFunc("/api/user/check-permission", s.standardMiddleware(s.authMiddleware(s.HandleCheckPermission)))

	// Role Session Policies routes - Root Only
	mux.HandleFunc("/api/roles/session-policies", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleRoleSessionPoliciesRoute))))

	// Role Password Policies routes - Root Only
	mux.HandleFunc("/api/roles/password-policies", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleRolePasswordPoliciesRoute))))

	// Role by ID with sub-routes (must be last for /api/roles/*)
	mux.HandleFunc("/api/roles/", s.standardMiddleware(s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/session-policy") {
			s.rootOnlyMiddleware(s.HandleRoleSessionPolicyByIDRoute)(w, r)
		} else if strings.HasSuffix(path, "/password-policy") {
			s.rootOnlyMiddleware(s.HandleRolePasswordPolicyByIDRoute)(w, r)
		} else {
			s.HandleRoleByIDRoute(w, r)
		}
	})))

	// Upload route - Root Only
	mux.HandleFunc("/api/upload", s.standardMiddleware(s.authMiddleware(s.rootOnlyMiddleware(s.HandleUpload))))

	// Static files for uploads
	// CAUTION: This exposes the directory. Validate security if needed.
	// http.StripPrefix strips the "/uploads/" part so we serve from current dir "./uploads"
	fileServer := http.FileServer(http.Dir("./uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fileServer))

	// Deploy Configuration (accessible without auth in development, with token in production)
	mux.HandleFunc("/api/deploy/config", s.standardMiddleware(s.HandleDeployConfig))
	mux.HandleFunc("/api/deploy/config/defaults", s.standardMiddleware(s.HandleDeployConfigDefaults))
	mux.HandleFunc("/api/deploy/status", s.standardMiddleware(s.HandleDeployStatus))
	mux.HandleFunc("/api/deploy/validate", s.corsMiddleware(s.HandleDeployValidate))

	// Initialize Admin (accessible only when database is completely empty)
	mux.HandleFunc("/api/initialize/admin", s.corsMiddleware(s.HandleInitializeAdmin))
	mux.HandleFunc("/api/initialize/status", s.corsMiddleware(s.HandleInitializeStatus))
}

func (s *Server) handleSettingsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.HandleGetSettings(w, r)
	case http.MethodPut:
		s.HandleUpdateSettings(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
