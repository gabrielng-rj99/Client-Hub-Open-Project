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
	"net/http"
	"strings"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============= AFFILIATE HANDLERS =============

func (s *Server) handleClientAffiliates(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/clients/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	clientID := parts[0]

	switch r.Method {
	case http.MethodGet:
		s.handleListAffiliates(w, r, clientID)
	case http.MethodPost:
		s.handleCreateAffiliate(w, r, clientID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListAffiliates(w http.ResponseWriter, _ *http.Request, clientID string) {
	// SECURITY: Validate UUID format
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	// SECURITY: Check if client exists before listing affiliates
	client, err := s.clientStore.GetClientByID(clientID)
	if err != nil || client == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	affiliates, err := s.affiliateStore.GetAffiliatesByClientID(clientID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: affiliates})
}

func (s *Server) handleCreateAffiliate(w http.ResponseWriter, r *http.Request, clientID string) {
	var affiliate domain.Affiliate
	if err := json.NewDecoder(r.Body).Decode(&affiliate); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	affiliate.ClientID = clientID
	affiliate.Status = "ativo"

	id, err := s.affiliateStore.CreateAffiliate(affiliate)
	if err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(affiliate)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "affiliate",
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
		affiliate.ID = id
		newValueJSON, _ := json.Marshal(affiliate)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "affiliate",
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
		Message: "Affiliate created successfully",
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleAffiliateByID(w http.ResponseWriter, r *http.Request) {
	affiliateID := getIDFromPath(r, "/api/affiliates/")

	if affiliateID == "" {
		respondError(w, http.StatusBadRequest, "Affiliate ID required")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleUpdateAffiliate(w, r, affiliateID)
	case http.MethodDelete:
		s.handleDeleteAffiliate(w, r, affiliateID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleUpdateAffiliate(w http.ResponseWriter, r *http.Request, affiliateID string) {
	var affiliate domain.Affiliate
	if err := json.NewDecoder(r.Body).Decode(&affiliate); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldAffiliate, _ := s.affiliateStore.GetAffiliateByID(affiliateID)
	oldValueJSON, _ := json.Marshal(oldAffiliate)

	// SECURITY: Preserve the original client_id - prevent client_id spoofing via request body
	if oldAffiliate != nil {
		affiliate.ClientID = oldAffiliate.ClientID
	}

	affiliate.ID = affiliateID

	if err := s.affiliateStore.UpdateAffiliate(affiliate); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(affiliate)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "affiliate",
				ResourceID:    affiliateID,
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
		newValueJSON, _ := json.Marshal(affiliate)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "affiliate",
			ResourceID:    affiliateID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Affiliate updated successfully"})
}

func (s *Server) handleDeleteAffiliate(w http.ResponseWriter, r *http.Request, affiliateID string) {
	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldAffiliate, _ := s.affiliateStore.GetAffiliateByID(affiliateID)
	oldValueJSON, _ := json.Marshal(oldAffiliate)

	if err := s.affiliateStore.DeleteAffiliate(affiliateID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "delete",
				Resource:      "affiliate",
				ResourceID:    affiliateID,
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
			Resource:      "affiliate",
			ResourceID:    affiliateID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Affiliate deleted successfully"})
}
