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

// ============= CONTRACT HANDLERS =============

func (s *Server) handleContracts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListContracts(w, r)
	case http.MethodPost:
		s.handleCreateContract(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListContracts(w http.ResponseWriter, _ *http.Request) {
	contracts, err := s.contractStore.GetAllContractsIncludingArchived()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: contracts})
}

func (s *Server) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	var contract domain.Contract
	if err := json.NewDecoder(r.Body).Decode(&contract); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SECURITY: Validate contract fields BEFORE inserting into database
	if err := domain.ValidateContract(&contract); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// SECURITY: Limit field lengths to prevent overflow attacks
	if len(contract.Model) > 500 {
		respondError(w, http.StatusBadRequest, "Model field too long (max 500 characters)")
		return
	}
	if len(contract.ItemKey) > 200 {
		respondError(w, http.StatusBadRequest, "ItemKey field too long (max 200 characters)")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	id, err := s.contractStore.CreateContract(contract)
	if err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(contract)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "contract",
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

	// Update client status based on active contracts
	if err := s.clientStore.UpdateClientStatus(contract.ClientID); err != nil {
		// Log warning but don't fail the request
		// This is a non-critical operation
	}

	// Update category status based on usage
	contractData, _ := s.contractStore.GetContractByID(id)
	if contractData != nil {
		lineData, _ := s.subcategoryStore.GetSubcategoryByID(contractData.SubcategoryID)
		if lineData != nil {
			if err := s.categoryStore.UpdateCategoryStatus(lineData.CategoryID); err != nil {
				// Log warning but don't fail the request
			}
		}
	}

	// Log successful creation
	if claims != nil {
		contract.ID = id
		newValueJSON, _ := json.Marshal(contract)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "contract",
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
		Message: "Contract created successfully",
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleContractByID(w http.ResponseWriter, r *http.Request) {
	contractID := getIDFromPath(r, "/api/contracts/")

	if contractID == "" {
		respondError(w, http.StatusBadRequest, "Contract ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetContract(w, r, contractID)
	case http.MethodPut:
		s.handleUpdateContract(w, r, contractID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleGetContract(w http.ResponseWriter, _ *http.Request, contractID string) {
	// SECURITY: Validate UUID format before querying database
	if err := domain.ValidateUUID(contractID); err != nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}

	contract, err := s.contractStore.GetContractByID(contractID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "Contract not found")
		} else {
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if contract == nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: contract})
}

func (s *Server) handleUpdateContract(w http.ResponseWriter, r *http.Request, contractID string) {
	var contract domain.Contract
	if err := json.NewDecoder(r.Body).Decode(&contract); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldContract, _ := s.contractStore.GetContractByID(contractID)
	oldValueJSON, _ := json.Marshal(oldContract)

	contract.ID = contractID

	if err := s.contractStore.UpdateContract(contract); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(contract)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "contract",
				ResourceID:    contractID,
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
	if err := s.clientStore.UpdateClientStatus(contract.ClientID); err != nil {
		// Log warning but don't fail the request
		// This is a non-critical operation
	}

	// Update category status based on usage
	lineData, _ := s.subcategoryStore.GetSubcategoryByID(contract.SubcategoryID)
	if lineData != nil {
		if err := s.categoryStore.UpdateCategoryStatus(lineData.CategoryID); err != nil {
			// Log warning but don't fail the request
		}
	}

	// Log successful update
	if claims != nil {
		newValueJSON, _ := json.Marshal(contract)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "contract",
			ResourceID:    contractID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Contract updated successfully"})
}

func (s *Server) handleContractArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/contracts/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Contract ID required")
		return
	}

	contractID := parts[0]

	// SECURITY: Validate UUID format before querying database
	if err := domain.ValidateUUID(contractID); err != nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldContract, err := s.contractStore.GetContractByID(contractID)
	if err != nil || oldContract == nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}
	oldValueJSON, _ := json.Marshal(oldContract)

	// Soft-delete: set archived_at instead of hard delete
	if err := s.contractStore.ArchiveContract(contractID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "archive",
				Resource:      "contract",
				ResourceID:    contractID,
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

	// Update client status based on active contracts
	if err := s.clientStore.UpdateClientStatus(oldContract.ClientID); err != nil {
		// Log warning but don't fail the request
	}

	// Update category status based on usage
	lineData, _ := s.subcategoryStore.GetSubcategoryByID(oldContract.SubcategoryID)
	if lineData != nil {
		if err := s.categoryStore.UpdateCategoryStatus(lineData.CategoryID); err != nil {
			// Log warning but don't fail the request
		}
	}

	// Log successful archive
	if claims != nil {
		newContract, _ := s.contractStore.GetContractByID(contractID)
		newValueJSON, _ := json.Marshal(newContract)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "archive",
			Resource:      "contract",
			ResourceID:    contractID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Contract archived successfully"})
}

func (s *Server) handleContractUnarchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/contracts/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Contract ID required")
		return
	}

	contractID := parts[0]

	// SECURITY: Validate UUID format before querying database
	if err := domain.ValidateUUID(contractID); err != nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldContract, err := s.contractStore.GetContractByID(contractID)
	if err != nil || oldContract == nil {
		respondError(w, http.StatusNotFound, "Contract not found")
		return
	}
	oldValueJSON, _ := json.Marshal(oldContract)

	if err := s.contractStore.UnarchiveContract(contractID); err != nil {
		// Log failed attempt
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "unarchive",
				Resource:      "contract",
				ResourceID:    contractID,
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

	// Update client status based on active contracts
	if err := s.clientStore.UpdateClientStatus(oldContract.ClientID); err != nil {
		// Log warning but don't fail the request
	}

	// Update category status based on usage
	lineData, _ := s.subcategoryStore.GetSubcategoryByID(oldContract.SubcategoryID)
	if lineData != nil {
		if err := s.categoryStore.UpdateCategoryStatus(lineData.CategoryID); err != nil {
			// Log warning but don't fail the request
		}
	}

	// Log successful unarchive
	if claims != nil {
		newContract, _ := s.contractStore.GetContractByID(contractID)
		newValueJSON, _ := json.Marshal(newContract)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "unarchive",
			Resource:      "contract",
			ResourceID:    contractID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Contract unarchived successfully"})
}
