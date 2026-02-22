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
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"Open-Generic-Hub/backend/config"
	"Open-Generic-Hub/backend/store"

	"golang.org/x/time/rate"
)

// Global database connection (used for health checks and initialization)
var globalDB *sql.DB

// SetGlobalDB sets the global database connection
func SetGlobalDB(db *sql.DB) {
	globalDB = db
}

// GetGlobalDB returns the global database connection
func GetGlobalDB() *sql.DB {
	return globalDB
}

type Server struct {
	db               *sql.DB
	userStore        *store.UserStore
	contractStore    *store.ContractStore
	clientStore      *store.ClientStore
	affiliateStore   *store.AffiliateStore
	categoryStore    *store.CategoryStore
	subcategoryStore *store.SubcategoryStore
	auditStore       *store.AuditStore
	settingsStore    *store.SettingsStore
	userThemeStore   *store.UserThemeStore
	roleStore        *store.RoleStore
	financialStore   *store.FinancialStore
	config           *config.Config
	rateLimiter      *IPRateLimiter
}

// NewServer creates a new Server instance with all stores
func NewServer(db *sql.DB) *Server {
	cfg := config.GetConfig()

	// Only create stores if database is available
	// This allows the server to start for initial setup/deploy panel
	var userStore *store.UserStore
	var contractStore *store.ContractStore
	var clientStore *store.ClientStore
	var affiliateStore *store.AffiliateStore
	var categoryStore *store.CategoryStore
	var subcategoryStore *store.SubcategoryStore
	var auditStore *store.AuditStore
	var settingsStore *store.SettingsStore
	var userThemeStore *store.UserThemeStore
	var roleStore *store.RoleStore
	var financialStore *store.FinancialStore

	if db != nil {
		userStore = store.NewUserStore(db)
		contractStore = store.NewContractStore(db)
		clientStore = store.NewClientStore(db)
		affiliateStore = store.NewAffiliateStore(db)
		categoryStore = store.NewCategoryStore(db)
		subcategoryStore = store.NewSubcategoryStore(db)
		auditStore = store.NewAuditStore(db)
		settingsStore = store.NewSettingsStore(db)
		userThemeStore = store.NewUserThemeStore(db)
		roleStore = store.NewRoleStore(db)
		financialStore = store.NewFinancialStore(db)
	}

	return &Server{
		db:               db,
		userStore:        userStore,
		contractStore:    contractStore,
		clientStore:      clientStore,
		affiliateStore:   affiliateStore,
		categoryStore:    categoryStore,
		subcategoryStore: subcategoryStore,
		auditStore:       auditStore,
		settingsStore:    settingsStore,
		userThemeStore:   userThemeStore,
		roleStore:        roleStore,
		financialStore:   financialStore,
		config:           cfg,
		rateLimiter:      NewIPRateLimiter(rate.Limit(cfg.Security.RateLimit), cfg.Security.RateBurst),
	}
}

// InitializeStores initializes all stores with the provided database connection
func (s *Server) InitializeStores(db *sql.DB) {
	if db != nil {
		s.db = db
		s.userStore = store.NewUserStore(db)
		s.contractStore = store.NewContractStore(db)
		s.clientStore = store.NewClientStore(db)
		s.affiliateStore = store.NewAffiliateStore(db)
		s.categoryStore = store.NewCategoryStore(db)
		s.subcategoryStore = store.NewSubcategoryStore(db)
		s.auditStore = store.NewAuditStore(db)
		s.settingsStore = store.NewSettingsStore(db)
		s.userThemeStore = store.NewUserThemeStore(db)
		s.roleStore = store.NewRoleStore(db)
		s.financialStore = store.NewFinancialStore(db)
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Panic recovery
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(os.Stderr, "PANIC in handler %s %s: %v\n", r.Method, r.URL.Path, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, s.config.Server.CORSOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)

			if s.config.Server.CORSAllowCreds {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.Server.CORSMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.Server.CORSHeaders, ", "))

			if len(s.config.Server.CORSExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(s.config.Server.CORSExposeHeaders, ", "))
			}

			if s.config.Server.CORSMaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.config.Server.CORSMaxAge))
			}
		} else if origin != "" {
			// Origin not allowed - log for security audit
			log.Printf("CORS: Origin not allowed: %s (allowed: %v)", origin, s.config.Server.CORSOrigins)
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}

func getIDFromPath(r *http.Request, prefix string) string {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
