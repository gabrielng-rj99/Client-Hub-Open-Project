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

package rolestore

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"Open-Generic-Hub/backend/repository"

	"github.com/google/uuid"
)

// Role represents a user role in the system
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description *string   `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	IsActive    bool      `json:"is_active"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a single permission in the system
type Permission struct {
	ID          string  `json:"id"`
	Resource    string  `json:"resource"`
	Action      string  `json:"action"`
	DisplayName string  `json:"display_name"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
}

// RoleWithPermissions represents a role with its associated permissions
type RoleWithPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}

// UserPermissions represents all permissions for a user
type UserPermissions struct {
	UserID      string              `json:"user_id"`
	Role        string              `json:"role"`
	RoleID      *string             `json:"role_id,omitempty"`
	Permissions map[string]bool     `json:"permissions"` // "resource:action" -> true
	Resources   map[string][]string `json:"resources"`   // resource -> [actions]
}

// RoleStore handles database operations for roles and permissions
type RoleStore struct {
	db repository.DBWithTx
}

// NewRoleStore creates a new RoleStore instance
func NewRoleStore(db repository.DBWithTx) *RoleStore {
	return &RoleStore{db: db}
}

// System role IDs (fixed UUIDs)
const (
	RoleIDRoot   = "a0000000-0000-0000-0000-000000000001"
	RoleIDAdmin  = "a0000000-0000-0000-0000-000000000002"
	RoleIDUser   = "a0000000-0000-0000-0000-000000000003"
	RoleIDViewer = "a0000000-0000-0000-0000-000000000004"
)

// ============================================
// Role Operations
// ============================================

// GetAllRoles returns all active roles ordered by priority
func (s *RoleStore) GetAllRoles() ([]Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system, is_active, priority, created_at, updated_at
		FROM roles
		WHERE is_active = true
		ORDER BY priority DESC, name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var description sql.NullString

		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.DisplayName,
			&description,
			&role.IsSystem,
			&role.IsActive,
			&role.Priority,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		if description.Valid {
			role.Description = &description.String
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// GetRoleByID returns a role by its ID
func (s *RoleStore) GetRoleByID(roleID string) (*Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system, is_active, priority, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	var role Role
	var description sql.NullString

	err := s.db.QueryRow(query, roleID).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&description,
		&role.IsSystem,
		&role.IsActive,
		&role.Priority,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if description.Valid {
		role.Description = &description.String
	}

	return &role, nil
}

// GetRoleByName returns a role by its name
func (s *RoleStore) GetRoleByName(name string) (*Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system, is_active, priority, created_at, updated_at
		FROM roles
		WHERE name = $1 AND is_active = true
	`

	var role Role
	var description sql.NullString

	err := s.db.QueryRow(query, name).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&description,
		&role.IsSystem,
		&role.IsActive,
		&role.Priority,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	if description.Valid {
		role.Description = &description.String
	}

	return &role, nil
}

// GetAnyRoleByName returns a role by its name regardless of active status
func (s *RoleStore) GetAnyRoleByName(name string) (*Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system, is_active, priority, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	var role Role
	var description sql.NullString

	err := s.db.QueryRow(query, name).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&description,
		&role.IsSystem,
		&role.IsActive,
		&role.Priority,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	if description.Valid {
		role.Description = &description.String
	}

	return &role, nil
}

// CreateRole creates a new role
func (s *RoleStore) CreateRole(name, displayName string, description *string, priority int) (*Role, error) {
	// Check if name already exists (active or inactive)
	existing, err := s.GetAnyRoleByName(name)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if existing.IsActive {
			return nil, errors.New("role with this name already exists")
		}

		// If inactive (deleted), rename it to free up the name
		timestamp := time.Now().Format("20060102150405")
		newName := fmt.Sprintf("%s_deleted_%s", existing.Name, timestamp)

		// Ensure new name fits in database (truncate if needed, though unlikely for standard names)
		if len(newName) > 255 {
			newName = newName[:255]
		}

		_, err = s.db.Exec("UPDATE roles SET name = $1 WHERE id = $2", newName, existing.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to rename old deleted role: %w", err)
		}
	}

	// Validate priority (custom roles must have priority < 100)
	if priority >= 100 {
		return nil, errors.New("custom roles cannot have priority >= 100")
	}

	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO roles (id, name, display_name, description, is_system, is_active, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, true, $5, $6, $6)
		RETURNING id
	`

	var desc sql.NullString
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}

	err = s.db.QueryRow(query, id, name, displayName, desc, priority, now).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return s.GetRoleByID(id)
}

// UpdateRole updates an existing role (only non-system roles can have name changed)
func (s *RoleStore) UpdateRole(roleID, displayName string, description *string, priority int) (*Role, error) {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("role not found")
	}

	// Root role cannot be modified at all
	if role.Name == "root" {
		return nil, errors.New("root role cannot be modified")
	}

	// Validate priority for non-root roles
	if priority >= 100 {
		return nil, errors.New("only root role can have priority >= 100")
	}

	query := `
		UPDATE roles
		SET display_name = $1, description = $2, priority = $3, updated_at = $4
		WHERE id = $5
	`

	var desc sql.NullString
	if description != nil {
		desc = sql.NullString{String: *description, Valid: true}
	}

	_, err = s.db.Exec(query, displayName, desc, priority, time.Now(), roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return s.GetRoleByID(roleID)
}

// DeleteRole soft-deletes a role (sets is_active = false)
func (s *RoleStore) DeleteRole(roleID string) error {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	if role.IsSystem {
		return errors.New("system roles cannot be deleted")
	}

	// Check if any users have this role
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = $1 AND deleted_at IS NULL", roleID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check users: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete role: %d users have this role", count)
	}

	// Rename role to free up the name for future use
	timestamp := time.Now().Format("20060102150405")
	deletedName := fmt.Sprintf("%s_deleted_%s", role.Name, timestamp)

	// Ensure we don't exceed max length (assuming 255 or similar, safe bet)
	if len(deletedName) > 255 {
		deletedName = deletedName[:255]
	}

	query := `UPDATE roles SET is_active = false, name = $1, updated_at = $2 WHERE id = $3`
	_, err = s.db.Exec(query, deletedName, time.Now(), roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// ============================================
// Permission Operations
// ============================================

// GetAllPermissions returns all permissions grouped by category
func (s *RoleStore) GetAllPermissions() ([]Permission, error) {
	query := `
		SELECT id, resource, action, display_name, description, category
		FROM permissions
		ORDER BY category, resource, action
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var perm Permission
		var description sql.NullString

		err := rows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&perm.DisplayName,
			&description,
			&perm.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		if description.Valid {
			perm.Description = &description.String
		}

		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// GetPermissionsByCategory returns permissions grouped by category
func (s *RoleStore) GetPermissionsByCategory() (map[string][]Permission, error) {
	permissions, err := s.GetAllPermissions()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]Permission)
	for _, perm := range permissions {
		result[perm.Category] = append(result[perm.Category], perm)
	}

	return result, nil
}

// GetPermissionByResourceAction returns a permission by resource and action
func (s *RoleStore) GetPermissionByResourceAction(resource, action string) (*Permission, error) {
	query := `
		SELECT id, resource, action, display_name, description, category
		FROM permissions
		WHERE resource = $1 AND action = $2
	`

	var perm Permission
	var description sql.NullString

	err := s.db.QueryRow(query, resource, action).Scan(
		&perm.ID,
		&perm.Resource,
		&perm.Action,
		&perm.DisplayName,
		&description,
		&perm.Category,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	if description.Valid {
		perm.Description = &description.String
	}

	return &perm, nil
}

// ============================================
// Role-Permission Operations
// ============================================

// GetRolePermissions returns all permissions for a role
func (s *RoleStore) GetRolePermissions(roleID string) ([]Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.display_name, p.description, p.category
		FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.category, p.resource, p.action
	`

	rows, err := s.db.Query(query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var perm Permission
		var description sql.NullString

		err := rows.Scan(
			&perm.ID,
			&perm.Resource,
			&perm.Action,
			&perm.DisplayName,
			&description,
			&perm.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		if description.Valid {
			perm.Description = &description.String
		}

		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// GetRoleWithPermissions returns a role with all its permissions
func (s *RoleStore) GetRoleWithPermissions(roleID string) (*RoleWithPermissions, error) {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}

	permissions, err := s.GetRolePermissions(roleID)
	if err != nil {
		return nil, err
	}

	return &RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}, nil
}

// SetRolePermissions sets all permissions for a role (replaces existing)
func (s *RoleStore) SetRolePermissions(roleID string, permissionIDs []string) error {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	// Root role always has all permissions
	if role.Name == "root" {
		return errors.New("root role permissions cannot be modified")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing permissions
	_, err = tx.Exec("DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("failed to delete existing permissions: %w", err)
	}

	// Insert new permissions
	stmt, err := tx.Prepare("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES ($1, $2, $3)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, permID := range permissionIDs {
		_, err = stmt.Exec(roleID, permID, now)
		if err != nil {
			return fmt.Errorf("failed to insert permission %s: %w", permID, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AddPermissionToRole adds a single permission to a role
func (s *RoleStore) AddPermissionToRole(roleID, permissionID string) error {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	if role.Name == "root" {
		return errors.New("root role permissions cannot be modified")
	}

	query := `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`

	_, err = s.db.Exec(query, roleID, permissionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a single permission from a role
func (s *RoleStore) RemovePermissionFromRole(roleID, permissionID string) error {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	if role.Name == "root" {
		return errors.New("root role permissions cannot be modified")
	}

	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err = s.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}

	return nil
}

// ============================================
// User Permission Check Operations
// ============================================

// GetUserPermissions returns all permissions for a user based on their role
func (s *RoleStore) GetUserPermissions(userID string) (*UserPermissions, error) {
	// Get user's role info
	query := `
		SELECT u.id, u.role, u.role_id
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	var uid string
	var roleName sql.NullString
	var roleID sql.NullString

	err := s.db.QueryRow(query, userID).Scan(&uid, &roleName, &roleID)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userPerms := &UserPermissions{
		UserID:      uid,
		Role:        "user", // default
		Permissions: make(map[string]bool),
		Resources:   make(map[string][]string),
	}

	if roleName.Valid {
		userPerms.Role = roleName.String
	}
	if roleID.Valid {
		userPerms.RoleID = &roleID.String
	}

	// Root always has all permissions
	if userPerms.Role == "root" {
		permissions, err := s.GetAllPermissions()
		if err != nil {
			return nil, err
		}
		for _, perm := range permissions {
			key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
			userPerms.Permissions[key] = true
			userPerms.Resources[perm.Resource] = append(userPerms.Resources[perm.Resource], perm.Action)
		}
		return userPerms, nil
	}

	// Get role by name or ID
	var role *Role
	if roleID.Valid {
		role, err = s.GetRoleByID(roleID.String)
	} else if roleName.Valid {
		role, err = s.GetRoleByName(roleName.String)
	}

	if err != nil {
		return nil, err
	}

	// If no role found, return empty permissions
	if role == nil {
		return userPerms, nil
	}

	// Get permissions for the role
	permissions, err := s.GetRolePermissions(role.ID)
	if err != nil {
		return nil, err
	}

	for _, perm := range permissions {
		key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
		userPerms.Permissions[key] = true
		userPerms.Resources[perm.Resource] = append(userPerms.Resources[perm.Resource], perm.Action)
	}

	return userPerms, nil
}

// HasPermission checks if a user has a specific permission
func (s *RoleStore) HasPermission(userID, resource, action string) (bool, error) {
	perms, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	// Root always has permission
	if perms.Role == "root" {
		return true, nil
	}

	key := fmt.Sprintf("%s:%s", resource, action)
	return perms.Permissions[key], nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (s *RoleStore) HasAnyPermission(userID string, permissions []string) (bool, error) {
	perms, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	// Root always has permission
	if perms.Role == "root" {
		return true, nil
	}

	for _, perm := range permissions {
		if perms.Permissions[perm] {
			return true, nil
		}
	}

	return false, nil
}

// HasAllPermissions checks if a user has all of the specified permissions
func (s *RoleStore) HasAllPermissions(userID string, permissions []string) (bool, error) {
	perms, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	// Root always has permission
	if perms.Role == "root" {
		return true, nil
	}

	for _, perm := range permissions {
		if !perms.Permissions[perm] {
			return false, nil
		}
	}

	return true, nil
}

// CanAccessResource checks if a user can perform any action on a resource
func (s *RoleStore) CanAccessResource(userID, resource string) (bool, error) {
	perms, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	// Root always has access
	if perms.Role == "root" {
		return true, nil
	}

	actions, exists := perms.Resources[resource]
	return exists && len(actions) > 0, nil
}

// GetResourceActions returns the actions a user can perform on a resource
func (s *RoleStore) GetResourceActions(userID, resource string) ([]string, error) {
	perms, err := s.GetUserPermissions(userID)
	if err != nil {
		return nil, err
	}

	// Root always has all actions
	if perms.Role == "root" {
		permissions, err := s.GetAllPermissions()
		if err != nil {
			return nil, err
		}
		var actions []string
		for _, perm := range permissions {
			if perm.Resource == resource {
				actions = append(actions, perm.Action)
			}
		}
		return actions, nil
	}

	return perms.Resources[resource], nil
}

// UserAccessResult holds the combined result of authentication and authorization
type UserAccessResult struct {
	AuthSecret  string
	LockedUntil *time.Time
	IsBlocked   bool
	HasAccess   bool
}

// CheckUserAccess validates the user's existence, lock status, and RBAC permission in a single query
func (s *RoleStore) CheckUserAccess(userID, resource, action string) (*UserAccessResult, error) {
	query := `
		SELECT 
			u.auth_secret,
			u.locked_until,
			CASE 
				WHEN u.role = 'root' THEN true
				WHEN EXISTS (
					SELECT 1 FROM roles r
					JOIN role_permissions rp ON r.id = rp.role_id
					JOIN permissions p ON p.id = rp.permission_id
					WHERE r.name = u.role AND p.resource = $2 AND p.action = $3
				) THEN true
				ELSE false 
			END as has_access
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	result := &UserAccessResult{}
	var authSecret sql.NullString
	var lockedUntil sql.NullTime
	var hasAccess sql.NullBool

	err := s.db.QueryRow(query, userID, resource, action).Scan(&authSecret, &lockedUntil, &hasAccess)

	if err == sql.ErrNoRows {
		return nil, errors.New("usuário não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("falha ao verificar acesso: %w", err)
	}

	if authSecret.Valid {
		result.AuthSecret = authSecret.String
	} else {
		return nil, errors.New("auth_secret ausente")
	}

	if lockedUntil.Valid {
		result.LockedUntil = &lockedUntil.Time
		if time.Now().Before(lockedUntil.Time) {
			result.IsBlocked = true
		}
	}

	if hasAccess.Valid {
		result.HasAccess = hasAccess.Bool
	}

	return result, nil
}

// ============================================
// Utility Functions
// ============================================

// GetAllRolesWithPermissions returns all roles with their permissions
func (s *RoleStore) GetAllRolesWithPermissions() ([]RoleWithPermissions, error) {
	roles, err := s.GetAllRoles()
	if err != nil {
		return nil, err
	}

	var result []RoleWithPermissions
	for _, role := range roles {
		permissions, err := s.GetRolePermissions(role.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, RoleWithPermissions{
			Role:        role,
			Permissions: permissions,
		})
	}

	return result, nil
}

// SyncUserRoleID updates the role_id field based on the role name for legacy compatibility
func (s *RoleStore) SyncUserRoleID(userID string) error {
	query := `
		UPDATE users u
		SET role_id = r.id
		FROM roles r
		WHERE u.id = $1
		  AND u.role = r.name
		  AND (u.role_id IS NULL OR u.role_id != r.id)
	`

	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to sync user role_id: %w", err)
	}

	return nil
}

// SyncAllUserRoleIDs syncs role_id for all users based on their role name
func (s *RoleStore) SyncAllUserRoleIDs() error {
	query := `
		UPDATE users u
		SET role_id = r.id
		FROM roles r
		WHERE u.role = r.name
		  AND (u.role_id IS NULL OR u.role_id != r.id)
	`

	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to sync all user role_ids: %w", err)
	}

	return nil
}

// ============================================
// Authorization Helper Functions
// ============================================

// GetUserRole returns the role name for a user from the database
func (s *RoleStore) GetUserRole(userID string) (string, error) {
	query := `SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`

	var role sql.NullString
	err := s.db.QueryRow(query, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", errors.New("user not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	if role.Valid {
		return role.String, nil
	}
	return "user", nil // default role
}

// IsUserRoot checks if a user has the root role
func (s *RoleStore) IsUserRoot(userID string) (bool, error) {
	role, err := s.GetUserRole(userID)
	if err != nil {
		return false, err
	}
	return role == "root", nil
}

// IsUserAdmin checks if a user has the admin role
func (s *RoleStore) IsUserAdmin(userID string) (bool, error) {
	role, err := s.GetUserRole(userID)
	if err != nil {
		return false, err
	}
	return role == "admin", nil
}

// IsUserAdminOrRoot checks if a user has admin or root role
func (s *RoleStore) IsUserAdminOrRoot(userID string) (bool, error) {
	role, err := s.GetUserRole(userID)
	if err != nil {
		return false, err
	}
	return role == "admin" || role == "root", nil
}
