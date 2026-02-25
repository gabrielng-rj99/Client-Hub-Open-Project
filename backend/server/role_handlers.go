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
	"strings"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============================================
// Request/Response Types
// ============================================

// CreateRoleRequest represents the request to create a new role
type CreateRoleRequest struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	Priority    int     `json:"priority"`
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	Priority    int     `json:"priority"`
}

// SetRolePermissionsRequest represents the request to set role permissions
type SetRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

// RoleResponse represents a role in API responses
type RoleResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	IsSystem    bool    `json:"is_system"`
	IsActive    bool    `json:"is_active"`
	Priority    int     `json:"priority"`
}

// PermissionResponse represents a permission in API responses
type PermissionResponse struct {
	ID          string  `json:"id"`
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
}

// RoleWithPermissionsResponse represents a role with its permissions
type RoleWithPermissionsResponse struct {
	Role        RoleResponse         `json:"role"`
	Permissions []PermissionResponse `json:"permissions"`
}

// UserPermissionsResponse represents a user's permissions
type UserPermissionsResponse struct {
	UserID      string              `json:"user_id"`
	Role        string              `json:"role"`
	RoleID      *string             `json:"role_id,omitempty"`
	Permissions map[string]bool     `json:"permissions"`
	Resources   map[string][]string `json:"resources"`
}

// ============================================
// Role Handlers
// ============================================

// HandleRolesRoute routes role requests
func (s *Server) HandleRolesRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.HandleGetRoles(w, r)
	case http.MethodPost:
		s.HandleCreateRole(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleRoleByIDRoute routes single role requests
func (s *Server) HandleRoleByIDRoute(w http.ResponseWriter, r *http.Request) {
	// Extract role ID from path: /api/roles/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/roles/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Role ID required")
		return
	}
	roleID := parts[0]
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}

	// Check if this is a permissions sub-route
	if len(parts) > 1 && parts[1] == "permissions" {
		s.handleRolePermissionsRoute(w, r, roleID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.HandleGetRoleByID(w, r, roleID)
	case http.MethodPut:
		s.HandleUpdateRole(w, r, roleID)
	case http.MethodDelete:
		s.HandleDeleteRole(w, r, roleID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleRolePermissionsRoute routes role permissions requests
func (s *Server) handleRolePermissionsRoute(w http.ResponseWriter, r *http.Request, roleID string) {
	switch r.Method {
	case http.MethodGet:
		s.HandleGetRolePermissions(w, r, roleID)
	case http.MethodPut:
		s.HandleSetRolePermissions(w, r, roleID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleGetRoles returns all roles
func (s *Server) HandleGetRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate token and check permissions
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Check if user has permission to read roles
	hasPermission, err := s.roleStore.HasPermission(claims.UserID, "roles", "read")
	if err != nil {
		log.Printf("❌ Error checking permission: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}

	// Check if user is root via DB (root always has all permissions)
	isRoot, _ := s.roleStore.IsUserRoot(claims.UserID)
	if !hasPermission && !isRoot {
		respondError(w, http.StatusForbidden, "Você não tem permissão para visualizar papéis")
		return
	}

	// Check if we should include permissions
	includePermissions := r.URL.Query().Get("include_permissions") == "true"

	if includePermissions {
		rolesWithPerms, err := s.roleStore.GetAllRolesWithPermissions()
		if err != nil {
			log.Printf("❌ Error getting roles with permissions: %v", err)
			respondError(w, http.StatusInternalServerError, "Erro ao buscar papéis")
			return
		}

		var response []RoleWithPermissionsResponse
		for _, rwp := range rolesWithPerms {
			var perms []PermissionResponse
			for _, p := range rwp.Permissions {
				perms = append(perms, PermissionResponse{
					ID:          p.ID,
					Resource:    p.Resource,
					Action:      p.Action,
					DisplayName: p.DisplayName,
					Description: p.Description,
					Category:    p.Category,
				})
			}
			response = append(response, RoleWithPermissionsResponse{
				Role: RoleResponse{
					ID:          rwp.Role.ID,
					Name:        rwp.Role.Name,
					DisplayName: rwp.Role.DisplayName,
					Description: rwp.Role.Description,
					IsSystem:    rwp.Role.IsSystem,
					IsActive:    rwp.Role.IsActive,
					Priority:    rwp.Role.Priority,
				},
				Permissions: perms,
			})
		}

		respondJSON(w, http.StatusOK, response)
		return
	}

	roles, err := s.roleStore.GetAllRoles()
	if err != nil {
		log.Printf("❌ Error getting roles: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar papéis")
		return
	}

	var response []RoleResponse
	for _, role := range roles {
		response = append(response, RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			IsActive:    role.IsActive,
			Priority:    role.Priority,
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleGetRoleByID returns a single role by ID
func (s *Server) HandleGetRoleByID(w http.ResponseWriter, r *http.Request, roleID string) {
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}
	// Validate token and check permissions
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	hasPermission, _ := s.roleStore.HasPermission(claims.UserID, "roles", "read")
	isRoot, _ := s.roleStore.IsUserRoot(claims.UserID)
	if !hasPermission && !isRoot {
		respondError(w, http.StatusForbidden, "Você não tem permissão para visualizar papéis")
		return
	}

	includePermissions := r.URL.Query().Get("include_permissions") == "true"

	if includePermissions {
		rwp, err := s.roleStore.GetRoleWithPermissions(roleID)
		if err != nil {
			log.Printf("❌ Error getting role with permissions: %v", err)
			respondError(w, http.StatusInternalServerError, "Erro ao buscar papel")
			return
		}
		if rwp == nil {
			respondError(w, http.StatusNotFound, "Papel não encontrado")
			return
		}

		var perms []PermissionResponse
		for _, p := range rwp.Permissions {
			perms = append(perms, PermissionResponse{
				ID:          p.ID,
				Resource:    p.Resource,
				Action:      p.Action,
				DisplayName: p.DisplayName,
				Description: p.Description,
				Category:    p.Category,
			})
		}

		respondJSON(w, http.StatusOK, RoleWithPermissionsResponse{
			Role: RoleResponse{
				ID:          rwp.Role.ID,
				Name:        rwp.Role.Name,
				DisplayName: rwp.Role.DisplayName,
				Description: rwp.Role.Description,
				IsSystem:    rwp.Role.IsSystem,
				IsActive:    rwp.Role.IsActive,
				Priority:    rwp.Role.Priority,
			},
			Permissions: perms,
		})
		return
	}

	role, err := s.roleStore.GetRoleByID(roleID)
	if err != nil {
		log.Printf("❌ Error getting role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar papel")
		return
	}
	if role == nil {
		respondError(w, http.StatusNotFound, "Papel não encontrado")
		return
	}

	respondJSON(w, http.StatusOK, RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsActive:    role.IsActive,
		Priority:    role.Priority,
	})
}

// HandleCreateRole creates a new role
func (s *Server) HandleCreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate token and check permissions
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can create roles - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		role, _ := s.roleStore.GetUserRole(claims.UserID)
		log.Printf("🚫 User %s (role: %s) denied access to create role", claims.Username, role)
		respondError(w, http.StatusForbidden, "Apenas usuários root podem criar papéis")
		return
	}

	// Parse request
	var req CreateRoleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Nome do papel é obrigatório")
		return
	}
	if req.DisplayName == "" {
		respondError(w, http.StatusBadRequest, "Nome de exibição é obrigatório")
		return
	}

	// Create role
	role, err := s.roleStore.CreateRole(req.Name, req.DisplayName, req.Description, req.Priority)
	if err != nil {
		log.Printf("❌ Error creating role: %v", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("✅ Root user %s created role: %s", claims.Username, role.Name)

	// Log to audit
	if s.auditStore != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "role",
			ResourceID:    role.ID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      nil,
			NewValue: map[string]interface{}{
				"name":         role.Name,
				"display_name": role.DisplayName,
				"priority":     role.Priority,
			},
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusCreated, RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsActive:    role.IsActive,
		Priority:    role.Priority,
	})
}

// HandleUpdateRole updates an existing role
func (s *Server) HandleUpdateRole(w http.ResponseWriter, r *http.Request, roleID string) {
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}
	// Validate token and check permissions
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can update roles - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		role, _ := s.roleStore.GetUserRole(claims.UserID)
		log.Printf("🚫 User %s (role: %s) denied access to update role", claims.Username, role)
		respondError(w, http.StatusForbidden, "Apenas usuários root podem editar papéis")
		return
	}

	// Get existing role for audit
	oldRole, err := s.roleStore.GetRoleByID(roleID)
	if err != nil {
		log.Printf("❌ Error getting role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar papel")
		return
	}
	if oldRole == nil {
		respondError(w, http.StatusNotFound, "Papel não encontrado")
		return
	}

	// Parse request
	var req UpdateRoleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	// Validate
	if req.DisplayName == "" {
		respondError(w, http.StatusBadRequest, "Nome de exibição é obrigatório")
		return
	}

	// Update role
	role, err := s.roleStore.UpdateRole(roleID, req.DisplayName, req.Description, req.Priority)
	if err != nil {
		log.Printf("❌ Error updating role: %v", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("✅ Root user %s updated role: %s", claims.Username, role.Name)

	// Log to audit
	if s.auditStore != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "role",
			ResourceID:    role.ID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue: map[string]interface{}{
				"display_name": oldRole.DisplayName,
				"description":  oldRole.Description,
				"priority":     oldRole.Priority,
			},
			NewValue: map[string]interface{}{
				"display_name": role.DisplayName,
				"description":  role.Description,
				"priority":     role.Priority,
			},
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		IsActive:    role.IsActive,
		Priority:    role.Priority,
	})
}

// HandleDeleteRole deletes a role
func (s *Server) HandleDeleteRole(w http.ResponseWriter, r *http.Request, roleID string) {
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}
	// Validate token and check permissions
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can delete roles - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		role, _ := s.roleStore.GetUserRole(claims.UserID)
		log.Printf("🚫 User %s (role: %s) denied access to delete role", claims.Username, role)
		respondError(w, http.StatusForbidden, "Apenas usuários root podem deletar papéis")
		return
	}

	// Get role for audit
	role, err := s.roleStore.GetRoleByID(roleID)
	if err != nil {
		log.Printf("❌ Error getting role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar papel")
		return
	}
	if role == nil {
		respondError(w, http.StatusNotFound, "Papel não encontrado")
		return
	}

	// Delete role
	err = s.roleStore.DeleteRole(roleID)
	if err != nil {
		log.Printf("❌ Error deleting role: %v", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("✅ Root user %s deleted role: %s", claims.Username, role.Name)

	// Log to audit
	if s.auditStore != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "delete",
			Resource:      "role",
			ResourceID:    role.ID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue: map[string]interface{}{
				"name":         role.Name,
				"display_name": role.DisplayName,
			},
			NewValue:      nil,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Papel deletado com sucesso",
	})
}

// ============================================
// Permission Handlers
// ============================================

// HandlePermissionsRoute routes permission requests
func (s *Server) HandlePermissionsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.HandleGetPermissions(w, r)
}

// HandleGetPermissions returns all available permissions
func (s *Server) HandleGetPermissions(w http.ResponseWriter, r *http.Request) {
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only admin and root can view all permissions - check via DB
	isAdminOrRoot, err := s.roleStore.IsUserAdminOrRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isAdminOrRoot {
		respondError(w, http.StatusForbidden, "Você não tem permissão para visualizar permissões")
		return
	}

	// Check if grouped by category
	groupByCategory := r.URL.Query().Get("group_by") == "category"

	if groupByCategory {
		permsByCategory, err := s.roleStore.GetPermissionsByCategory()
		if err != nil {
			log.Printf("❌ Error getting permissions by category: %v", err)
			respondError(w, http.StatusInternalServerError, "Erro ao buscar permissões")
			return
		}

		response := make(map[string][]PermissionResponse)
		for category, perms := range permsByCategory {
			for _, p := range perms {
				response[category] = append(response[category], PermissionResponse{
					ID:          p.ID,
					Resource:    p.Resource,
					Action:      p.Action,
					DisplayName: p.DisplayName,
					Description: p.Description,
					Category:    p.Category,
				})
			}
		}

		respondJSON(w, http.StatusOK, response)
		return
	}

	permissions, err := s.roleStore.GetAllPermissions()
	if err != nil {
		log.Printf("❌ Error getting permissions: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar permissões")
		return
	}

	var response []PermissionResponse
	for _, p := range permissions {
		response = append(response, PermissionResponse{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Category:    p.Category,
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleGetRolePermissions returns permissions for a specific role
func (s *Server) HandleGetRolePermissions(w http.ResponseWriter, r *http.Request, roleID string) {
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Check permission - DB based
	hasPermission, _ := s.roleStore.HasPermission(claims.UserID, "roles", "read")
	isRoot, _ := s.roleStore.IsUserRoot(claims.UserID)
	if !hasPermission && !isRoot {
		respondError(w, http.StatusForbidden, "Você não tem permissão para visualizar permissões de papéis")
		return
	}

	permissions, err := s.roleStore.GetRolePermissions(roleID)
	if err != nil {
		log.Printf("❌ Error getting role permissions: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar permissões do papel")
		return
	}

	response := make([]PermissionResponse, 0, len(permissions))
	for _, p := range permissions {
		response = append(response, PermissionResponse{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Category:    p.Category,
		})
	}

	if len(response) == 0 {
		respondJSON(w, http.StatusOK, []PermissionResponse{})
		return
	}

	respondJSON(w, http.StatusOK, response)
}

// HandleSetRolePermissions sets all permissions for a role
func (s *Server) HandleSetRolePermissions(w http.ResponseWriter, r *http.Request, roleID string) {
	// SECURITY: Validate UUID (generic format to accept hardcoded role IDs)
	if err := domain.ValidateUUIDGeneric(roleID); err != nil {
		respondError(w, http.StatusNotFound, "Role not found")
		return
	}
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Only root can modify role permissions - check via DB
	isRoot, err := s.roleStore.IsUserRoot(claims.UserID)
	if err != nil {
		log.Printf("❌ Error checking user role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissões")
		return
	}
	if !isRoot {
		role, _ := s.roleStore.GetUserRole(claims.UserID)
		log.Printf("🚫 User %s (role: %s) denied access to set role permissions", claims.Username, role)
		respondError(w, http.StatusForbidden, "Apenas usuários root podem alterar permissões de papéis")
		return
	}

	// Get role for audit
	role, err := s.roleStore.GetRoleByID(roleID)
	if err != nil {
		log.Printf("❌ Error getting role: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar papel")
		return
	}
	if role == nil {
		respondError(w, http.StatusNotFound, "Papel não encontrado")
		return
	}

	// Get old permissions for audit
	oldPerms, _ := s.roleStore.GetRolePermissions(roleID)
	var oldPermIDs []string
	for _, p := range oldPerms {
		oldPermIDs = append(oldPermIDs, p.ID)
	}

	// Parse request
	var req SetRolePermissionsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Corpo da requisição inválido")
		return
	}

	// Validate permissions IDs (use generic UUID validation for hardcoded permission IDs)
	for _, permID := range req.PermissionIDs {
		if err := domain.ValidateUUIDGeneric(permID); err != nil {
			respondError(w, http.StatusBadRequest, "ID de permissão inválido: "+permID)
			return
		}
	}

	// Set permissions
	err = s.roleStore.SetRolePermissions(roleID, req.PermissionIDs)
	if err != nil {
		log.Printf("❌ Error setting role permissions: %v", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("✅ Root user %s updated permissions for role: %s", claims.Username, role.Name)

	// Log to audit
	if s.auditStore != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "role_permissions",
			ResourceID:    role.ID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue: map[string]interface{}{
				"permission_ids": oldPermIDs,
			},
			NewValue: map[string]interface{}{
				"permission_ids": req.PermissionIDs,
			},
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	// Return updated permissions
	newPerms, err := s.roleStore.GetRolePermissions(roleID)
	if err != nil {
		log.Printf("❌ Error getting updated permissions: %v", err)
		respondError(w, http.StatusInternalServerError, "Permissões atualizadas, mas erro ao retornar lista")
		return
	}

	var response []PermissionResponse
	for _, p := range newPerms {
		response = append(response, PermissionResponse{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Category:    p.Category,
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Permissões atualizadas com sucesso",
		Data:    response,
	})
}

// ============================================
// User Permission Handlers
// ============================================

// HandleUserPermissionsRoute routes user permission requests
func (s *Server) HandleUserPermissionsRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.HandleGetCurrentUserPermissions(w, r)
}

// HandleGetCurrentUserPermissions returns the current user's permissions
func (s *Server) HandleGetCurrentUserPermissions(w http.ResponseWriter, r *http.Request) {
	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	perms, err := s.roleStore.GetUserPermissions(claims.UserID)
	if err != nil {
		log.Printf("❌ Error getting user permissions: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao buscar permissões do usuário")
		return
	}

	respondJSON(w, http.StatusOK, UserPermissionsResponse{
		UserID:      perms.UserID,
		Role:        perms.Role,
		RoleID:      perms.RoleID,
		Permissions: perms.Permissions,
		Resources:   perms.Resources,
	})
}

// HandleCheckPermission checks if the current user has a specific permission
func (s *Server) HandleCheckPermission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate token
	tokenString := extractTokenFromHeader(r)
	claims, err := ExtractJWTClaimsUnverified(tokenString)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	resource := r.URL.Query().Get("resource")
	action := r.URL.Query().Get("action")

	if resource == "" || action == "" {
		respondError(w, http.StatusBadRequest, "resource e action são obrigatórios")
		return
	}

	hasPermission, err := s.roleStore.HasPermission(claims.UserID, resource, action)
	if err != nil {
		log.Printf("❌ Error checking permission: %v", err)
		respondError(w, http.StatusInternalServerError, "Erro ao verificar permissão")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"resource":       resource,
		"action":         action,
		"has_permission": hasPermission,
	})
}
