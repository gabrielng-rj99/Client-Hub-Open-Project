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
	"strings"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============= LINE HANDLERS =============

func (s *Server) handleSubcategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListLines(w, r)
	case http.MethodPost:
		s.handleCreateSubcategory(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListLines(w http.ResponseWriter, _ *http.Request) {
	subcategories, err := s.subcategoryStore.GetAllSubcategories()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: subcategories})
}

func (s *Server) handleCreateSubcategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		CategoryID string `json:"category_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	line := domain.Subcategory{
		Name:       req.Name,
		CategoryID: req.CategoryID,
	}

	// SECURITY: Validate subcategory input
	if err := domain.ValidateSubcategory(&line); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.subcategoryStore.CreateSubcategory(line)
	if err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(line)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "subcategory",
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
		line.ID = id
		newValueJSON, _ := json.Marshal(line)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "subcategory",
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
		Message: "Subcategory created successfully",
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleSubcategoryByID(w http.ResponseWriter, r *http.Request) {
	subcategoryID := getIDFromPath(r, "/api/subcategories/")

	if subcategoryID == "" {
		respondError(w, http.StatusBadRequest, "Subcategory ID required")
		return
	}

	// SECURITY: Validate UUID
	if err := domain.ValidateUUID(subcategoryID); err != nil {
		respondError(w, http.StatusNotFound, "Subcategory not found")
		return
	}

	// Check for archive/unarchive endpoints
	if strings.HasSuffix(r.URL.Path, "/archive") {
		if r.Method == http.MethodPost {
			s.handleArchiveSubcategory(w, r, subcategoryID)
			return
		}
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if strings.HasSuffix(r.URL.Path, "/unarchive") {
		if r.Method == http.MethodPost {
			s.handleUnarchiveSubcategory(w, r, subcategoryID)
			return
		}
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetLine(w, r, subcategoryID)
	case http.MethodPut:
		s.handleUpdateSubcategory(w, r, subcategoryID)
	case http.MethodDelete:
		s.handleDeleteSubcategory(w, r, subcategoryID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleGetLine(w http.ResponseWriter, _ *http.Request, subcategoryID string) {
	line, err := s.subcategoryStore.GetSubcategoryByID(subcategoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Subcategory not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if line == nil {
		respondError(w, http.StatusNotFound, "Subcategory not found")
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: line})
}

func (s *Server) handleUpdateSubcategory(w http.ResponseWriter, r *http.Request, subcategoryID string) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get existing line to preserve category_id and for audit
	existingLine, err := s.subcategoryStore.GetSubcategoryByID(subcategoryID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Subcategory not found")
		return
	}

	oldValueJSON, _ := json.Marshal(existingLine)

	line := domain.Subcategory{
		ID:         subcategoryID,
		Name:       req.Name,
		CategoryID: existingLine.CategoryID,
	}

	if err := s.subcategoryStore.UpdateSubcategory(line); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(line)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "subcategory",
				ResourceID:    subcategoryID,
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
		newValueJSON, _ := json.Marshal(line)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "subcategory",
			ResourceID:    subcategoryID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Subcategory updated successfully"})
}

func (s *Server) handleDeleteSubcategory(w http.ResponseWriter, r *http.Request, subcategoryID string) {
	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldLine, _ := s.subcategoryStore.GetSubcategoryByID(subcategoryID)
	oldValueJSON, _ := json.Marshal(oldLine)

	if err := s.subcategoryStore.DeleteSubcategory(subcategoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "delete",
				Resource:      "subcategory",
				ResourceID:    subcategoryID,
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
			Resource:      "subcategory",
			ResourceID:    subcategoryID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Subcategory deleted successfully"})
}

func (s *Server) handleArchiveSubcategory(w http.ResponseWriter, r *http.Request, subcategoryID string) {
	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldLine, _ := s.subcategoryStore.GetSubcategoryByID(subcategoryID)
	oldValueJSON, _ := json.Marshal(oldLine)

	if err := s.subcategoryStore.ArchiveSubcategory(subcategoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "archive",
				Resource:      "subcategory",
				ResourceID:    subcategoryID,
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
			Resource:      "subcategory",
			ResourceID:    subcategoryID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Subcategory archived successfully"})
}

func (s *Server) handleUnarchiveSubcategory(w http.ResponseWriter, r *http.Request, subcategoryID string) {
	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldLine, _ := s.subcategoryStore.GetSubcategoryByID(subcategoryID)
	oldValueJSON, _ := json.Marshal(oldLine)

	if err := s.subcategoryStore.UnarchiveSubcategory(subcategoryID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "unarchive",
				Resource:      "subcategory",
				ResourceID:    subcategoryID,
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
			Resource:      "subcategory",
			ResourceID:    subcategoryID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Subcategory unarchived successfully"})
}
