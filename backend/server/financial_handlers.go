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
	"strconv"
	"strings"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/store"
)

// ============= FINANCIAL HANDLERS =============

// handleFinancial gerencia requisições para /api/financial
func (s *Server) handleFinancial(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListFinancial(w, r)
	case http.MethodPost:
		s.handleCreateFinancial(w, r)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListFinancial lista todos os financeiros
func (s *Server) handleListFinancial(w http.ResponseWriter, r *http.Request) {
	// Parse pagination params (backward compatible: limit=0 or absent returns all)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Clamp values
	if offset < 0 {
		offset = 0
	}

	// limit=0 or absent → return all (backward compatibility)
	if limit <= 0 {
		financials, err := s.financialStore.GetAllFinancials()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		total := len(financials)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"data":   financials,
			"total":  total,
			"limit":  total,
			"offset": 0,
		})
		return
	}

	// Cap limit to a sane max to prevent abuse
	if limit > 500 {
		limit = 500
	}

	// Paginated path: use optimized paged query + separate count
	total, err := s.financialStore.CountFinancials()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	financials, err := s.financialStore.GetAllFinancialsPaged(limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   financials,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// handleCreateFinancial cria um novo modelo de financeiro
func (s *Server) handleCreateFinancial(w http.ResponseWriter, r *http.Request) {
	var req CreateFinancialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validar campos obrigatórios
	if req.ContractID == "" {
		respondError(w, http.StatusBadRequest, "contract_id é obrigatório")
		return
	}
	if err := domain.ValidateUUID(req.ContractID); err != nil {
		respondError(w, http.StatusBadRequest, "contract_id inválido")
		return
	}

	if req.FinancialType == "" {
		respondError(w, http.StatusBadRequest, "financial_type é obrigatório")
		return
	}
	if req.ClientValue == nil {
		respondError(w, http.StatusBadRequest, "client_value é obrigatório")
		return
	}
	if req.ReceivedValue == nil {
		respondError(w, http.StatusBadRequest, "received_value é obrigatório")
		return
	}

	// Criar o financeiro
	financial := domain.ContractFinancial{
		ContractID:     req.ContractID,
		FinancialType:  req.FinancialType,
		RecurrenceType: req.RecurrenceType,
		DueDay:         req.DueDay,
		ClientValue:    req.ClientValue,
		ReceivedValue:  req.ReceivedValue,
		Description:    req.Description,
		IsActive:       true,
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	id, err := s.financialStore.CreateContractFinancial(financial)
	if err != nil {
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(financial)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "create",
				Resource:      "financial",
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

	// Se for personalizado, criar as parcelas
	if req.FinancialType == "personalizado" && len(req.Installments) > 0 {
		err := s.financialStore.CreateInstallmentsBatch(id, req.Installments)
		if err != nil {
			// Rollback: deletar o financeiro criado
			s.financialStore.DeleteContractFinancial(id)
			respondError(w, http.StatusBadRequest, "Erro ao criar parcelas: "+err.Error())
			return
		}
	}

	// Log successful creation
	if claims != nil {
		financial.ID = id
		newValueJSON, _ := json.Marshal(financial)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "create",
			Resource:      "financial",
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
		Message: "Financeiro criado com sucesso",
		Data:    map[string]string{"id": id},
	})
}

// handleFinancialByID gerencia requisições para /api/financial/{id}
func (s *Server) handleFinancialByID(w http.ResponseWriter, r *http.Request) {
	financialID := getIDFromPath(r, "/api/financial/")

	if financialID == "" {
		respondError(w, http.StatusBadRequest, "Financial ID required")
		return
	}

	// SECURITY: Validate UUID format
	if err := domain.ValidateUUID(financialID); err != nil {
		respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		return
	}

	// Verificar se é uma ação específica
	if strings.Contains(r.URL.Path, "/installments") {
		s.handleFinancialInstallments(w, r, financialID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetFinancial(w, r, financialID)
	case http.MethodPut:
		s.handleUpdateFinancial(w, r, financialID)
	case http.MethodDelete:
		s.handleDeleteFinancial(w, r, financialID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetFinancial retorna um financeiro específico
func (s *Server) handleGetFinancial(w http.ResponseWriter, _ *http.Request, financialID string) {
	if err := domain.ValidateUUID(financialID); err != nil {
		respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		return
	}

	financial, err := s.financialStore.GetContractFinancialByID(financialID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "não encontrado") ||
			strings.Contains(errMsg, "invalid input syntax for type uuid") ||
			strings.Contains(errMsg, "invalid UUID") ||
			strings.Contains(errMsg, "SQLSTATE 22P02") {
			respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		} else {
			respondError(w, http.StatusInternalServerError, errMsg)
		}
		return
	}

	if financial == nil {
		respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: financial})
}

// handleUpdateFinancial atualiza um financeiro
func (s *Server) handleUpdateFinancial(w http.ResponseWriter, r *http.Request, financialID string) {
	var req UpdateFinancialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldFinancial, err := s.financialStore.GetContractFinancialByID(financialID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "não encontrado") ||
			strings.Contains(errMsg, "invalid input syntax for type uuid") ||
			strings.Contains(errMsg, "invalid UUID") ||
			strings.Contains(errMsg, "SQLSTATE 22P02") {
			respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		} else {
			respondError(w, http.StatusInternalServerError, errMsg)
		}
		return
	}
	if oldFinancial == nil {
		respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		return
	}
	oldValueJSON, _ := json.Marshal(oldFinancial)

	// Atualizar campos
	financial := domain.ContractFinancial{
		ID:             financialID,
		ContractID:     oldFinancial.ContractID,
		FinancialType:  req.FinancialType,
		RecurrenceType: req.RecurrenceType,
		DueDay:         req.DueDay,
		ClientValue:    req.ClientValue,
		ReceivedValue:  req.ReceivedValue,
		Description:    req.Description,
		IsActive:       req.IsActive,
	}

	if financial.FinancialType == "" {
		financial.FinancialType = oldFinancial.FinancialType
	}

	if err := s.financialStore.UpdateContractFinancial(financial); err != nil {
		errMsg := err.Error()
		if claims != nil {
			newValueJSON, _ := json.Marshal(financial)
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "update",
				Resource:      "financial",
				ResourceID:    financialID,
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

	// Se mudar para personalizado e tiver parcelas, atualizar
	if req.FinancialType == "personalizado" && len(req.Installments) > 0 {
		// Deletar parcelas antigas
		s.financialStore.DeleteAllInstallmentsByFinancialID(financialID)
		// Criar novas
		err := s.financialStore.CreateInstallmentsBatch(financialID, req.Installments)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Erro ao atualizar parcelas: "+err.Error())
			return
		}
	}

	// Log successful update
	if claims != nil {
		newValueJSON, _ := json.Marshal(financial)
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "update",
			Resource:      "financial",
			ResourceID:    financialID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Financeiro atualizado com sucesso"})
}

// handleDeleteFinancial deleta um financeiro
func (s *Server) handleDeleteFinancial(w http.ResponseWriter, r *http.Request, financialID string) {
	if err := domain.ValidateUUID(financialID); err != nil {
		respondError(w, http.StatusNotFound, "Financeiro não encontrado")
		return
	}

	claims, _ := ExtractJWTClaimsUnverified(extractTokenFromHeader(r))

	// Get old value for audit
	oldFinancial, _ := s.financialStore.GetContractFinancialByID(financialID)
	oldValueJSON, _ := json.Marshal(oldFinancial)

	if err := s.financialStore.DeleteContractFinancial(financialID); err != nil {
		errMsg := err.Error()
		if claims != nil {
			s.auditStore.LogOperation(store.AuditLogRequest{
				Operation:     "delete",
				Resource:      "financial",
				ResourceID:    financialID,
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
		if strings.Contains(errMsg, "não encontrado") ||
			strings.Contains(errMsg, "invalid input syntax for type uuid") ||
			strings.Contains(errMsg, "invalid UUID") ||
			strings.Contains(errMsg, "SQLSTATE 22P02") {
			respondError(w, http.StatusNotFound, "Financeiro não encontrado")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log successful delete
	if claims != nil {
		s.auditStore.LogOperation(store.AuditLogRequest{
			Operation:     "delete",
			Resource:      "financial",
			ResourceID:    financialID,
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

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Financeiro deletado com sucesso"})
}

// handleContractFinancial gerencia requisições para /api/contracts/{id}/financial
func (s *Server) handleContractFinancial(w http.ResponseWriter, r *http.Request) {
	// Extrair contract ID do path
	path := strings.TrimPrefix(r.URL.Path, "/api/contracts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" {
		respondError(w, http.StatusBadRequest, "Contract ID required")
		return
	}
	contractID := parts[0]

	switch r.Method {
	case http.MethodGet:
		financial, err := s.financialStore.GetContractFinancialByContractID(contractID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Retornar null se não existir (não é erro)
		respondJSON(w, http.StatusOK, SuccessResponse{Data: financial})

	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ============= INSTALLMENT HANDLERS =============

// handleFinancialInstallments gerencia parcelas de um financeiro
func (s *Server) handleFinancialInstallments(w http.ResponseWriter, r *http.Request, financialID string) {
	// Verificar se tem ID de parcela no path
	path := strings.TrimPrefix(r.URL.Path, "/api/financial/"+financialID+"/installments")
	installmentID := strings.TrimPrefix(path, "/")

	// Remover sufixos de ação (/pay, /unpay) do installmentID
	installmentID = strings.TrimSuffix(installmentID, "/pay")
	installmentID = strings.TrimSuffix(installmentID, "/unpay")

	if installmentID != "" {
		// Operação em parcela específica
		s.handleInstallmentByID(w, r, financialID, installmentID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListInstallments(w, r, financialID)
	case http.MethodPost:
		s.handleCreateInstallment(w, r, financialID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListInstallments lista parcelas de um financeiro
func (s *Server) handleListInstallments(w http.ResponseWriter, _ *http.Request, financialID string) {
	installments, err := s.financialStore.GetInstallmentsByFinancialID(financialID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: installments})
}

// handleCreateInstallment cria uma nova parcela
func (s *Server) handleCreateInstallment(w http.ResponseWriter, r *http.Request, financialID string) {
	var installment domain.FinancialInstallment
	if err := json.NewDecoder(r.Body).Decode(&installment); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if installment.InstallmentNumber == 0 &&
		installment.ClientValue == 0 &&
		installment.ReceivedValue == 0 &&
		installment.DueDate == nil &&
		installment.InstallmentLabel == nil &&
		installment.Notes == nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	installment.ContractFinancialID = financialID

	id, err := s.financialStore.CreateInstallment(installment)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, SuccessResponse{
		Message: "Parcela criada com sucesso",
		Data:    map[string]string{"id": id},
	})
}

// handleInstallmentByID gerencia uma parcela específica
func (s *Server) handleInstallmentByID(w http.ResponseWriter, r *http.Request, _, installmentID string) {
	// Verificar ações especiais
	if strings.HasSuffix(r.URL.Path, "/pay") {
		s.handleMarkInstallmentPaid(w, r, installmentID)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/unpay") {
		s.handleMarkInstallmentPending(w, r, installmentID)
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleUpdateInstallment(w, r, installmentID)
	case http.MethodDelete:
		s.handleDeleteInstallment(w, r, installmentID)
	default:
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleUpdateInstallment atualiza uma parcela
func (s *Server) handleUpdateInstallment(w http.ResponseWriter, r *http.Request, installmentID string) {
	var installment domain.FinancialInstallment
	if err := json.NewDecoder(r.Body).Decode(&installment); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	installment.ID = installmentID

	if err := s.financialStore.UpdateInstallment(installment); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Parcela atualizada com sucesso"})
}

// handleDeleteInstallment deleta uma parcela
func (s *Server) handleDeleteInstallment(w http.ResponseWriter, _ *http.Request, installmentID string) {
	if err := s.financialStore.DeleteInstallment(installmentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Parcela deletada com sucesso"})
}

// handleMarkInstallmentPaid marca uma parcela como paga
func (s *Server) handleMarkInstallmentPaid(w http.ResponseWriter, r *http.Request, installmentID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := s.financialStore.MarkInstallmentAsPaid(installmentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Parcela marcada como paga"})
}

// handleMarkInstallmentPending marca uma parcela como pendente
func (s *Server) handleMarkInstallmentPending(w http.ResponseWriter, r *http.Request, installmentID string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := s.financialStore.MarkInstallmentAsPending(installmentID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Parcela marcada como pendente"})
}

// ============= DASHBOARD HANDLERS =============

// handleFinancialDetailedSummary retorna resumo detalhado de financeiros com dados por período
func (s *Server) handleFinancialDetailedSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Primeiro, atualizar status de parcelas vencidas
	s.financialStore.UpdateOverdueStatus()

	summary, err := s.financialStore.GetFinancialDetailedSummary()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: summary})
}

// handleFinancialSummary retorna resumo de financeiros para dashboard
func (s *Server) handleFinancialSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Verificar se é resumo mensal
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	var summary *domain.FinancialSummary
	var err error

	if yearStr != "" && monthStr != "" {
		year, _ := strconv.Atoi(yearStr)
		month, _ := strconv.Atoi(monthStr)
		if year > 0 && month >= 1 && month <= 12 {
			summary, err = s.financialStore.GetMonthlySummary(year, month)
		} else {
			summary, err = s.financialStore.GetFinancialSummary()
		}
	} else {
		summary, err = s.financialStore.GetFinancialSummary()
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Data: summary})
}

// handleUpcomingFinancial retorna parcelas próximas do vencimento
func (s *Server) handleUpcomingFinancial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Dias à frente (padrão: 30)
	daysAhead := 30
	if days := r.URL.Query().Get("days"); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			daysAhead = d
		}
	}

	financials, err := s.financialStore.GetUpcomingFinancials(daysAhead)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total := len(financials)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   financials,
		"total":  total,
		"limit":  total,
		"offset": 0,
	})
}

// handleOverdueFinancial retorna parcelas em atraso
func (s *Server) handleOverdueFinancial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Primeiro, atualizar status de parcelas vencidas
	s.financialStore.UpdateOverdueStatus()

	financials, err := s.financialStore.GetOverdueFinancials()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total := len(financials)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   financials,
		"total":  total,
		"limit":  total,
		"offset": 0,
	})
}

// ============= REQUEST TYPES =============

// CreateFinancialRequest representa a requisição para criar um financeiro
type CreateFinancialRequest struct {
	ContractID     string                        `json:"contract_id"`
	FinancialType  string                        `json:"financial_type"` // 'unico', 'recorrente', 'personalizado'
	RecurrenceType *string                       `json:"recurrence_type,omitempty"`
	DueDay         *int                          `json:"due_day,omitempty"`
	ClientValue    *float64                      `json:"client_value,omitempty"`
	ReceivedValue  *float64                      `json:"received_value,omitempty"`
	Description    *string                       `json:"description,omitempty"`
	Installments   []domain.FinancialInstallment `json:"installments,omitempty"` // Para tipo personalizado
}

// UpdateFinancialRequest representa a requisição para atualizar um financeiro
type UpdateFinancialRequest struct {
	FinancialType  string                        `json:"financial_type"`
	RecurrenceType *string                       `json:"recurrence_type,omitempty"`
	DueDay         *int                          `json:"due_day,omitempty"`
	ClientValue    *float64                      `json:"client_value,omitempty"`
	ReceivedValue  *float64                      `json:"received_value,omitempty"`
	Description    *string                       `json:"description,omitempty"`
	IsActive       bool                          `json:"is_active"`
	Installments   []domain.FinancialInstallment `json:"installments,omitempty"`
}
