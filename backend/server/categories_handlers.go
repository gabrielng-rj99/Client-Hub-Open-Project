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
	"net/http"
	"strconv"
	"strings"
	"time"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============= CATEGORY HANDLERS =============

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListCategories(w, r)
	case http.MethodPost:
		s.handleCreateCategory(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	searchQuery := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")

	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // Default limit for API performance
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset > 0 {
			offset = parsedOffset
		}
	}

	var categories []domain.Category
	var err error

	// Process fetching
	categories, err = s.categoryStore.SearchCategories(searchQuery, includeArchived, limit, offset)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch subcategories for each category
	type CategoryWithLines struct {
		ID         string               `json:"id"`
		Name       string               `json:"name"`
		Status     string               `json:"status"`
		ArchivedAt *time.Time           `json:"archived_at"`
		Lines      []domain.Subcategory `json:"lines"`
	}

	var categoriesWithLines []CategoryWithLines

	categoryIDs := make([]string, 0, len(categories))
	for _, category := range categories {
		categoryIDs = append(categoryIDs, category.ID)
	}

	linesByCategory, err := s.subcategoryStore.GetSubcategoriesByCategoryIDs(categoryIDs, includeArchived)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, category := range categories {
		subcategories := linesByCategory[category.ID]

		categoriesWithLines = append(categoriesWithLines, CategoryWithLines{
			ID:         category.ID,
			Name:       category.Name,
			Status:     category.Status,
			ArchivedAt: category.ArchivedAt,
			Lines:      subcategories,
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: categoriesWithLines})
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SECURITY: Validate name is not empty
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, "Category name is required")
		return
	}

	// SECURITY: Limit name length to prevent overflow attacks
	if len(req.Name) > 200 {
		respondError(w, http.StatusBadRequest, "Category name too long (max 200 characters)")
		return
	}

	claims, _ := ValidateJWT(extractTokenFromHeader(r), s.userStore)

	category := domain.Category{Name: req.Name}

	id, err := s.categoryStore.CreateCategory(category)
	if err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(category)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "category",
				ResourceID:    "unknown",
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      nil,
				NewValue:      bytesToStringPtr(newValueJSON),
				Status:        "error",
				ErrorMessage:  &errMsg,
				IPAddress:     getIPAddress(r),
				UserAgent:     getUserAgent(r),
				RequestMethod: getRequestMethod(r),
				RequestPath:   getRequestPath(r),
			})
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful creation
	if claims != nil {
		category.ID = id
		newValueJSON, _ := json.Marshal(category)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "category",
			ResourceID:    id,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      nil,
			NewValue:      bytesToStringPtr(newValueJSON),
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusCreated, SuccessResponse{
		Message: "Category created successfully",
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID := getIDFromPath(r, "/api/categories/")

	if categoryID == "" {
		respondError(w, http.StatusBadRequest, "Category ID required")
		return
	}

	// Check for archive/unarchive endpoints
	if strings.HasSuffix(r.URL.Path, "/archive") {
		if r.Method == http.MethodPost {
			s.handleArchiveCategory(w, r, categoryID)
			return
		}
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if strings.HasSuffix(r.URL.Path, "/unarchive") {
		if r.Method == http.MethodPost {
			s.handleUnarchiveCategory(w, r, categoryID)
			return
		}
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetCategory(w, r, categoryID)
	case http.MethodPut:
		s.handleUpdateCategory(w, r, categoryID)
	case http.MethodDelete:
		s.handleDeleteCategory(w, r, categoryID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleGetCategory(w http.ResponseWriter, _ *http.Request, categoryID string) {
	// SECURITY: Validate UUID format before querying database
	if err := domain.ValidateUUID(categoryID); err != nil {
		respondError(w, http.StatusNotFound, "Category not found")
		return
	}

	category, err := s.categoryStore.GetCategoryByID(categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Category not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if category == nil {
		respondError(w, http.StatusNotFound, "Category not found")
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: category})
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request, categoryID string) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ValidateJWT(extractTokenFromHeader(r), s.userStore)

	// Get old value for audit
	oldCategory, _ := s.categoryStore.GetCategoryByID(categoryID)
	oldValueJSON, _ := json.Marshal(oldCategory)

	category := domain.Category{ID: categoryID, Name: req.Name}

	if err := s.categoryStore.UpdateCategory(category); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(category)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "category",
				ResourceID:    categoryID,
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      bytesToStringPtr(oldValueJSON),
				NewValue:      bytesToStringPtr(newValueJSON),
				Status:        "error",
				ErrorMessage:  &errMsg,
				IPAddress:     getIPAddress(r),
				UserAgent:     getUserAgent(r),
				RequestMethod: getRequestMethod(r),
				RequestPath:   getRequestPath(r),
			})
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful update
	if claims != nil {
		newValueJSON, _ := json.Marshal(category)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "category",
			ResourceID:    categoryID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      bytesToStringPtr(oldValueJSON),
			NewValue:      bytesToStringPtr(newValueJSON),
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Category updated successfully"})
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request, categoryID string) {
	claims, _ := ValidateJWT(extractTokenFromHeader(r), s.userStore)

	// Get old value for audit
	oldCategory, _ := s.categoryStore.GetCategoryByID(categoryID)
	oldValueJSON, _ := json.Marshal(oldCategory)

	if err := s.categoryStore.DeleteCategory(categoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "delete",
				Resource:      "category",
				ResourceID:    categoryID,
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      bytesToStringPtr(oldValueJSON),
				NewValue:      nil,
				Status:        "error",
				ErrorMessage:  &errMsg,
				IPAddress:     getIPAddress(r),
				UserAgent:     getUserAgent(r),
				RequestMethod: getRequestMethod(r),
				RequestPath:   getRequestPath(r),
			})
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful deletion
	if claims != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "delete",
			Resource:      "category",
			ResourceID:    categoryID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      bytesToStringPtr(oldValueJSON),
			NewValue:      nil,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Category deleted successfully"})
}

func (s *Server) handleCategorySubcategories(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/categories/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Category ID required")
		return
	}

	categoryID := parts[0]

	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"

	var subcategories []domain.Subcategory
	var err error

	if includeArchived {
		// Get all subcategories for this category including archived
		allLines, err := s.subcategoryStore.GetAllSubcategoriesIncludingArchived()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Filter by category ID
		subcategories = make([]domain.Subcategory, 0)
		for _, line := range allLines {
			if line.CategoryID == categoryID {
				subcategories = append(subcategories, line)
			}
		}
	} else {
		subcategories, err = s.subcategoryStore.GetSubcategoriesByCategoryID(categoryID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: subcategories})
}

func (s *Server) handleArchiveCategory(w http.ResponseWriter, r *http.Request, categoryID string) {
	claims, _ := ValidateJWT(extractTokenFromHeader(r), s.userStore)

	// Get old value for audit
	oldCategory, _ := s.categoryStore.GetCategoryByID(categoryID)
	oldValueJSON, _ := json.Marshal(oldCategory)

	if err := s.categoryStore.ArchiveCategory(categoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "archive",
				Resource:      "category",
				ResourceID:    categoryID,
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      bytesToStringPtr(oldValueJSON),
				NewValue:      nil,
				Status:        "error",
				ErrorMessage:  &errMsg,
				IPAddress:     getIPAddress(r),
				UserAgent:     getUserAgent(r),
				RequestMethod: getRequestMethod(r),
				RequestPath:   getRequestPath(r),
			})
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful archive
	if claims != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "archive",
			Resource:      "category",
			ResourceID:    categoryID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      bytesToStringPtr(oldValueJSON),
			NewValue:      nil,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Category archived successfully"})
}

func (s *Server) handleUnarchiveCategory(w http.ResponseWriter, r *http.Request, categoryID string) {
	claims, _ := ValidateJWT(extractTokenFromHeader(r), s.userStore)

	// Get old value for audit
	oldCategory, _ := s.categoryStore.GetCategoryByID(categoryID)
	oldValueJSON, _ := json.Marshal(oldCategory)

	if err := s.categoryStore.UnarchiveCategory(categoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "unarchive",
				Resource:      "category",
				ResourceID:    categoryID,
				AdminID:       &claims.UserID,
				AdminUsername: &claims.Username,
				OldValue:      bytesToStringPtr(oldValueJSON),
				NewValue:      nil,
				Status:        "error",
				ErrorMessage:  &errMsg,
				IPAddress:     getIPAddress(r),
				UserAgent:     getUserAgent(r),
				RequestMethod: getRequestMethod(r),
				RequestPath:   getRequestPath(r),
			})
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful unarchive
	if claims != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "unarchive",
			Resource:      "category",
			ResourceID:    categoryID,
			AdminID:       &claims.UserID,
			AdminUsername: &claims.Username,
			OldValue:      bytesToStringPtr(oldValueJSON),
			NewValue:      nil,
			Status:        "success",
			IPAddress:     getIPAddress(r),
			UserAgent:     getUserAgent(r),
			RequestMethod: getRequestMethod(r),
			RequestPath:   getRequestPath(r),
		})
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Category unarchived successfully"})
}
