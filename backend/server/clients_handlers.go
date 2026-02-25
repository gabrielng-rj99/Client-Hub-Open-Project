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

	"github.com/google/uuid"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============= CLIENT HANDLERS =============

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListClients(w, r)
	case http.MethodPost:
		s.handleCreateClient(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	// Check if requesting with contract stats
	includeStats := r.URL.Query().Get("include_stats") == "true"

	includeArchivedParam := r.URL.Query().Get("include_archived")
	includeArchived := true
	if includeArchivedParam != "" {
		includeArchived = includeArchivedParam == "true"
	}

	filter := r.URL.Query().Get("filter")
	search := r.URL.Query().Get("search")

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	useFiltered := filter != "" || search != "" || limit > 0 || includeArchivedParam != ""
	if useFiltered {
		clients, err := s.clientStore.ListClientsFilteredPaged(filter, search, includeArchived, limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		total, err := s.clientStore.CountClientsFiltered(filter, search, includeArchived)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if includeStats {
			// Get contract stats only for the returned clients
			clientIDs := make([]string, 0, len(clients))
			for _, client := range clients {
				clientIDs = append(clientIDs, client.ID)
			}

			statsMap, err := s.contractStore.GetContractStatsForClientIDs(clientIDs)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}

			// Create response with clients and their stats
			type ClientWithStats struct {
				domain.Client
				ActiveContracts   int `json:"active_contracts"`
				ExpiredContracts  int `json:"expired_contracts"`
				ArchivedContracts int `json:"archived_contracts"`
			}

			clientsWithStats := make([]ClientWithStats, 0, len(clients))
			for _, client := range clients {
				cws := ClientWithStats{Client: client}
				if stats, ok := statsMap[client.ID]; ok {
					cws.ActiveContracts = stats.ActiveContracts
					cws.ExpiredContracts = stats.ExpiredContracts
					cws.ArchivedContracts = stats.ArchivedContracts
				}
				clientsWithStats = append(clientsWithStats, cws)
			}

			respondJSON(w, http.StatusOK, map[string]interface{}{
				"data":   clientsWithStats,
				"total":  total,
				"limit":  limit,
				"offset": offset,
			})
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"data":   clients,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	clients, err := s.clientStore.GetAllClientsIncludingArchived()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if includeStats {
		// Get contract stats only for the returned clients
		clientIDs := make([]string, 0, len(clients))
		for _, client := range clients {
			clientIDs = append(clientIDs, client.ID)
		}

		statsMap, err := s.contractStore.GetContractStatsForClientIDs(clientIDs)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Create response with clients and their stats
		type ClientWithStats struct {
			domain.Client
			ActiveContracts   int `json:"active_contracts"`
			ExpiredContracts  int `json:"expired_contracts"`
			ArchivedContracts int `json:"archived_contracts"`
		}

		clientsWithStats := make([]ClientWithStats, 0, len(clients))
		for _, client := range clients {
			cws := ClientWithStats{Client: client}
			if stats, ok := statsMap[client.ID]; ok {
				cws.ActiveContracts = stats.ActiveContracts
				cws.ExpiredContracts = stats.ExpiredContracts
				cws.ArchivedContracts = stats.ArchivedContracts
			}
			clientsWithStats = append(clientsWithStats, cws)
		}

		respondJSON(w, http.StatusOK, SuccessResponse{Data: clientsWithStats})
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: clients})
}

func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var client domain.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Status will be auto-calculated by CreateClient based on active contracts
	// Ignore any status sent from frontend
	client.Status = ""

	// SECURITY: Sanitize all text fields to prevent XSS
	client.Name = sanitizeHTML(client.Name)
	if client.Nickname != nil {
		sanitized := sanitizeHTML(*client.Nickname)
		client.Nickname = &sanitized
	}
	if client.Notes != nil {
		sanitized := sanitizeHTML(*client.Notes)
		client.Notes = &sanitized
	}
	if client.Address != nil {
		sanitized := sanitizeHTML(*client.Address)
		client.Address = &sanitized
	}
	if client.Tags != nil {
		sanitized := sanitizeHTML(*client.Tags)
		client.Tags = &sanitized
	}
	if client.Documents != nil {
		sanitized := sanitizeHTML(*client.Documents)
		client.Documents = &sanitized
	}

	// SECURITY: Validate client input fields
	if err := domain.ValidateClient(&client); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.clientStore.CreateClient(client)
	if err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(client)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "client",
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
		client.ID = id
		newValueJSON, _ := json.Marshal(client)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "client",
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
		Message: "Client created successfully",
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleClientByID(w http.ResponseWriter, r *http.Request) {
	clientID := getIDFromPath(r, "/api/clients/")

	if clientID == "" {
		respondError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(clientID); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid Client ID format")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetClient(w, r, clientID)
	case http.MethodPut:
		s.handleUpdateClient(w, r, clientID)
	case http.MethodDelete:
		s.handleDeleteClient(w, r, clientID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleDeleteClient(w http.ResponseWriter, r *http.Request, clientID string) {
	// SECURITY: Validate UUID format
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	claims, err := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Token inválido ou expirado")
		return
	}

	// Get old value for audit
	oldClient, err := s.clientStore.GetClientByID(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Client not found")
		} else {
			respondError(w, http.StatusInternalServerError, "Database error checking client")
		}
		return
	}
	if oldClient == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	oldValueJSON, _ := json.Marshal(oldClient)

	if err := s.clientStore.DeleteClientPermanently(clientID); err != nil {
		// Log error
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "delete",
				Resource:      "client",
				ResourceID:    clientID,
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

	// Audit success
	if claims != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "delete",
			Resource:      "client",
			ResourceID:    clientID,
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

	respondJSON(w, http.StatusNoContent, nil) // 204 No Content
}

func (s *Server) handleGetClient(w http.ResponseWriter, _ *http.Request, clientID string) {
	// SECURITY: Validate UUID
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	client, err := s.clientStore.GetClientByID(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Client not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if client == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: client})
}

func (s *Server) handleUpdateClient(w http.ResponseWriter, r *http.Request, clientID string) {
	var updateData domain.Client
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SECURITY: Validate UUID
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get existing client for partial update support
	existingClient, err := s.clientStore.GetClientByID(clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Client not found")
		} else {
			respondError(w, http.StatusInternalServerError, "Database error checking client")
		}
		return
	}
	if existingClient == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	oldValueJSON, _ := json.Marshal(existingClient)

	// Merge: use existing values if not provided in update
	client := *existingClient
	if updateData.Name != "" {
		client.Name = sanitizeHTML(updateData.Name)
	}
	if updateData.Nickname != nil {
		sanitized := sanitizeHTML(*updateData.Nickname)
		client.Nickname = &sanitized
	}
	if updateData.Notes != nil {
		sanitized := sanitizeHTML(*updateData.Notes)
		client.Notes = &sanitized
	}
	if updateData.Address != nil {
		sanitized := sanitizeHTML(*updateData.Address)
		client.Address = &sanitized
	}
	if updateData.Tags != nil {
		sanitized := sanitizeHTML(*updateData.Tags)
		client.Tags = &sanitized
	}
	if updateData.Documents != nil {
		sanitized := sanitizeHTML(*updateData.Documents)
		client.Documents = &sanitized
	}
	if updateData.Email != nil {
		client.Email = updateData.Email
	}
	if updateData.Phone != nil {
		client.Phone = updateData.Phone
	}
	if updateData.RegistrationID != nil {
		client.RegistrationID = updateData.RegistrationID
	}
	if updateData.BirthDate != nil {
		client.BirthDate = updateData.BirthDate
	}
	if updateData.NextActionDate != nil {
		client.NextActionDate = updateData.NextActionDate
	}

	// SECURITY: Validate the merged client
	if err := domain.ValidateClient(&client); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	client.ID = clientID
	// Status will be auto-calculated by UpdateClient based on active contracts
	// Ignore any status sent from frontend
	client.Status = ""

	if err := s.clientStore.UpdateClient(client); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(client)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "client",
				ResourceID:    clientID,
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

	// Update client status based on active contracts
	if err := s.clientStore.UpdateClientStatus(clientID); err != nil {
		// Log warning but don't fail the request
		// This is a non-critical operation
	}

	// Log successful update
	if claims != nil {
		newValueJSON, _ := json.Marshal(client)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "client",
			ResourceID:    clientID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Client updated successfully"})
}

func (s *Server) handleClientArchive(w http.ResponseWriter, r *http.Request) {
	// Allow POST and PUT
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/clients/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	clientID := parts[0]
	// SECURITY: Validate UUID
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldClient, err := s.clientStore.GetClientByID(clientID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	if oldClient == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	oldValueJSON, _ := json.Marshal(oldClient)

	if err := s.clientStore.ArchiveClient(clientID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "client",
				ResourceID:    clientID,
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
		newClient, _ := s.clientStore.GetClientByID(clientID)
		newValueJSON, _ := json.Marshal(newClient)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "client",
			ResourceID:    clientID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Client archived successfully"})
}

func (s *Server) handleClientUnarchive(w http.ResponseWriter, r *http.Request) {
	// Allow POST and PUT
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/clients/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Client ID required")
		return
	}

	clientID := parts[0]
	// SECURITY: Validate UUID
	if err := domain.ValidateUUID(clientID); err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldClient, err := s.clientStore.GetClientByID(clientID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	if oldClient == nil {
		respondError(w, http.StatusNotFound, "Client not found")
		return
	}
	oldValueJSON, _ := json.Marshal(oldClient)

	if err := s.clientStore.UnarchiveClient(clientID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "client",
				ResourceID:    clientID,
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
		newClient, _ := s.clientStore.GetClientByID(clientID)
		newValueJSON, _ := json.Marshal(newClient)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "client",
			ResourceID:    clientID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Client unarchived successfully"})
}
