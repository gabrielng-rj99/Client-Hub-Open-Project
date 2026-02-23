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

package domain

import (
	"time"
)

// Client representa a tabela 'clients'
type Client struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	RegistrationID    *string    `json:"registration_id,omitempty"`    // CPF/CNPJ (opcional)
	Nickname          *string    `json:"nickname,omitempty"`           // Apelido/Nome fantasia (opcional)
	BirthDate         *time.Time `json:"birth_date,omitempty"`         // Data de nascimento/fundação (opcional)
	Email             *string    `json:"email,omitempty"`              // Email do cliente (opcional)
	Phone             *string    `json:"phone,omitempty"`              // Telefone do cliente (opcional)
	Address           *string    `json:"address,omitempty"`            // Endereço (opcional)
	Notes             *string    `json:"notes,omitempty"`              // Observações gerais (opcional)
	Status            string     `json:"status"`                       // Status: ativo, inativo, etc.
	Tags              *string    `json:"tags,omitempty"`               // Tags separadas por vírgula (opcional)
	ContactPreference *string    `json:"contact_preference,omitempty"` // Preferência de contato (whatsapp, email, etc.)
	LastContactDate   *time.Time `json:"last_contact_date,omitempty"`  // Data do último contato (opcional)
	NextActionDate    *time.Time `json:"next_action_date,omitempty"`   // Próxima ação planejada (opcional)
	CreatedAt         time.Time  `json:"created_at"`                   // Data de cadastro
	Documents         *string    `json:"documents,omitempty"`          // Lista de documentos/links (opcional)
	ArchivedAt        *time.Time `json:"archived_at"`                  // Soft delete via archived_at
}

// Affiliate representa a tabela 'affiliates'
type Affiliate struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	ClientID          string     `json:"client_id"`
	Description       *string    `json:"description,omitempty"`        // Descrição livre (opcional)
	BirthDate         *time.Time `json:"birth_date,omitempty"`         // Data de nascimento/fundação (opcional)
	Email             *string    `json:"email,omitempty"`              // Email do afiliado (opcional)
	Phone             *string    `json:"phone,omitempty"`              // Telefone do afiliado (opcional)
	Address           *string    `json:"address,omitempty"`            // Endereço (opcional)
	Notes             *string    `json:"notes,omitempty"`              // Observações (opcional)
	Status            string     `json:"status"`                       // Status: ativo, inativo, etc.
	Tags              *string    `json:"tags,omitempty"`               // Tags separadas por vírgula (opcional)
	ContactPreference *string    `json:"contact_preference,omitempty"` // Preferência de contato (opcional)
	Documents         *string    `json:"documents,omitempty"`          // Lista de documentos/links (opcional)
}

// Category representa a tabela 'categories'
type Category struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	ArchivedAt *time.Time `json:"archived_at"`
}

// Subcategory representa a tabela 'subcategories'
type Subcategory struct {
	ID         string     `json:"id"`
	Name       string     `json:"name" db:"name"`
	CategoryID string     `json:"category_id"`
	ArchivedAt *time.Time `json:"archived_at"`
}

// Contract representa a tabela 'contracts'
type Contract struct {
	ID            string     `json:"id"`
	Model         string     `json:"model" db:"name"`
	ItemKey       string     `json:"item_key"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	SubcategoryID string     `json:"subcategory_id"`
	ClientID      string     `json:"client_id"`
	AffiliateID   *string    `json:"affiliate_id"` // Usamos um ponteiro para que possa ser nulo
	ArchivedAt    *time.Time `json:"archived_at"`

	// Enriched fields (populated by JOINs, not persisted)
	ClientName      string  `json:"client_name,omitempty"`
	ClientNickname  *string `json:"client_nickname,omitempty"`
	CategoryName    string  `json:"category_name,omitempty"`
	SubcategoryName string  `json:"subcategory_name,omitempty"`
}

// User representa um usuário do sistema para autenticação
type User struct {
	ID             string     `json:"id"`
	Username       *string    `json:"username"`
	DisplayName    *string    `json:"display_name"`
	PasswordHash   string     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Role           *string    `json:"role"` // "user", "admin", "root"
	FailedAttempts int        `json:"failed_attempts"`
	LockLevel      int        `json:"lock_level"`
	LockedUntil    *time.Time `json:"locked_until"`
	AuthSecret     string     `json:"-"`
}

// GetEffectiveStartDate retorna a data de início efetiva para cálculos.
// Se StartDate for nil, retorna time.Time{} (infinito inferior) para indicar que começou "sempre".
func (c *Contract) GetEffectiveStartDate() time.Time {
	if c.StartDate == nil {
		return time.Time{}
	}
	return *c.StartDate
}

// GetEffectiveEndDate retorna a data de fim efetiva para cálculos.
// Se EndDate for nil, retorna uma data muito distante no futuro (infinito superior) para indicar que nunca expira.
func (c *Contract) GetEffectiveEndDate() time.Time {
	if c.EndDate == nil {
		return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	return *c.EndDate
}

// IsActive verifica se o contrato está ativo no momento especificado.
// StartDate nil = começou sempre (infinito inferior)
// EndDate nil = nunca expira (infinito superior)
func (c *Contract) IsActive(at time.Time) bool {
	effectiveStart := c.GetEffectiveStartDate()
	effectiveEnd := c.GetEffectiveEndDate()

	// Se StartDate é nil (infinito inferior), sempre começou
	startedAlready := effectiveStart.IsZero() || !at.Before(effectiveStart)

	// Se EndDate é nil (infinito superior), nunca expira
	notExpired := at.Before(effectiveEnd)

	return startedAlready && notExpired
}

// Status calcula e retorna o estado atual do contrato (Ativo, Expirando, Expirado, Cancelado).
func (c *Contract) Status() string {
	now := time.Now()

	// EndDate nil = nunca expira, sempre "Ativo"
	if c.EndDate == nil {
		return "Ativo"
	}

	effectiveEnd := c.GetEffectiveEndDate()

	if effectiveEnd.Before(now) {
		return "Expirado"
	}

	// Consideramos "Expirando em Breve" se faltar 30 dias ou menos.
	daysUntilExpiration := effectiveEnd.Sub(now).Hours() / 24
	if daysUntilExpiration <= 30 {
		return "Expirando em Breve"
	}

	return "Ativo"
}

// ContractFinancial representa a tabela 'contract_financial'
// Modelo de financeiro associado a um contrato
type ContractFinancial struct {
	ID             string    `json:"id"`
	ContractID     string    `json:"contract_id"`
	FinancialType  string    `json:"financial_type"`            // 'unico', 'recorrente', 'personalizado'
	RecurrenceType *string   `json:"recurrence_type,omitempty"` // 'mensal', 'trimestral', 'semestral', 'anual'
	DueDay         *int      `json:"due_day,omitempty"`         // Dia do vencimento (1-31)
	ClientValue    *float64  `json:"client_value,omitempty"`    // Quanto o cliente paga
	ReceivedValue  *float64  `json:"received_value,omitempty"`  // Quanto você recebe
	Description    *string   `json:"description,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Campos calculados (não persistidos)
	TotalClientValue   *float64 `json:"total_client_value,omitempty"`
	TotalReceivedValue *float64 `json:"total_received_value,omitempty"`
	TotalInstallments  int      `json:"total_installments,omitempty"`
	PaidInstallments   int      `json:"paid_installments,omitempty"`

	// Enriched fields (populated by JOINs, not persisted)
	ContractModel   string     `json:"contract_model,omitempty"`
	ClientName      string     `json:"client_name,omitempty"`
	ClientNickname  *string    `json:"client_nickname,omitempty"`
	CategoryName    string     `json:"category_name,omitempty"`
	SubcategoryName string     `json:"subcategory_name,omitempty"`
	ContractStart   *time.Time `json:"contract_start,omitempty"`
	ContractEnd     *time.Time `json:"contract_end,omitempty"`

	// Relacionamentos (para responses)
	Installments []FinancialInstallment `json:"installments,omitempty"`
}

// FinancialInstallment representa a tabela 'financial_installments'
// Parcelas customizadas para financeiro do tipo 'personalizado'
type FinancialInstallment struct {
	ID                  string     `json:"id"`
	ContractFinancialID string     `json:"contract_financial_id"`
	InstallmentNumber   int        `json:"installment_number"` // 0 = Entrada, 1 = 1ª Parcela, etc.
	InstallmentLabel    *string    `json:"installment_label,omitempty"`
	ClientValue         float64    `json:"client_value"`
	ReceivedValue       float64    `json:"received_value"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	Status              string     `json:"status"` // 'pendente', 'pago', 'atrasado', 'cancelado'
	Notes               *string    `json:"notes,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// GetDefaultInstallmentLabel retorna um label padrão para o número da parcela
func GetDefaultInstallmentLabel(number int) string {
	if number == 0 {
		return "Entrada"
	}
	return string(rune('0'+number)) + "ª Parcela"
}

// FinancialSummary representa um resumo de financeiro para dashboard
type FinancialSummary struct {
	TotalToReceive  float64 `json:"total_to_receive"`
	TotalClientPays float64 `json:"total_client_pays"`
	AlreadyReceived float64 `json:"already_received"`
	PendingCount    int     `json:"pending_count"`
	PaidCount       int     `json:"paid_count"`
	OverdueCount    int     `json:"overdue_count"`
}

// PeriodSummary representa o resumo de um período específico (mês)
type PeriodSummary struct {
	Period          string  `json:"period"`           // Ex: "2025-01", "2025-02"
	PeriodLabel     string  `json:"period_label"`     // Ex: "Janeiro 2025", "Fevereiro 2025"
	TotalToReceive  float64 `json:"total_to_receive"` // Valor total a receber no período
	TotalClientPays float64 `json:"total_client_pays"`
	AlreadyReceived float64 `json:"already_received"` // Já recebido no período
	PendingAmount   float64 `json:"pending_amount"`   // Ainda pendente no período
	PendingCount    int     `json:"pending_count"`
	PaidCount       int     `json:"paid_count"`
	OverdueCount    int     `json:"overdue_count"`
	OverdueAmount   float64 `json:"overdue_amount"`
}

// DetailedFinancialSummary representa um resumo detalhado com dados por período
type DetailedFinancialSummary struct {
	// Resumo geral
	TotalToReceive    float64 `json:"total_to_receive"`
	TotalClientPays   float64 `json:"total_client_pays"`
	TotalReceived     float64 `json:"total_received"`
	TotalPending      float64 `json:"total_pending"`
	TotalOverdue      float64 `json:"total_overdue"`
	TotalOverdueCount int     `json:"total_overdue_count"`

	// Resumos por período
	LastMonth    *PeriodSummary `json:"last_month,omitempty"`
	CurrentMonth *PeriodSummary `json:"current_month,omitempty"`
	NextMonth    *PeriodSummary `json:"next_month,omitempty"`

	// Lista de meses com dados (para visão expandida)
	MonthlyBreakdown []PeriodSummary `json:"monthly_breakdown,omitempty"`

	// Metadados
	GeneratedAt string `json:"generated_at"`
	CurrentDate string `json:"current_date"`
}

// UpcomingFinancial representa um financeiro próximo para listagem
type UpcomingFinancial struct {
	InstallmentID       string     `json:"installment_id"`
	ContractFinancialID string     `json:"contract_financial_id"`
	ContractID          string     `json:"contract_id"`
	ClientID            string     `json:"client_id"`
	ClientName          string     `json:"client_name"`
	ContractModel       string     `json:"contract_model"`
	InstallmentLabel    *string    `json:"installment_label,omitempty"`
	ClientValue         float64    `json:"client_value"`
	ReceivedValue       float64    `json:"received_value"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	Status              string     `json:"status"`
}

// AuditLog representa a tabela 'audit_logs' para rastreamento de auditoria
type AuditLog struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Operation       string    `json:"operation"` // 'create', 'update', 'delete', 'read'
	Resource        string    `json:"resource"`  // 'client', 'contract', 'user', 'subcategory', 'category', 'affiliate'
	ResourceID      string    `json:"resource_id"`
	ObjectName      *string   `json:"object_name,omitempty"` // Nome do objeto (cliente, categoria, etc)
	AdminID         *string   `json:"admin_id,omitempty"`
	AdminUsername   *string   `json:"admin_username,omitempty"`
	OldValue        *string   `json:"old_value,omitempty"` // JSON string com valores antigos
	NewValue        *string   `json:"new_value,omitempty"` // JSON string com valores novos
	Status          string    `json:"status"`              // 'success', 'error'
	ErrorMessage    *string   `json:"error_message,omitempty"`
	IPAddress       *string   `json:"ip_address,omitempty"`
	UserAgent       *string   `json:"user_agent,omitempty"`
	RequestMethod   *string   `json:"request_method,omitempty"`    // GET, POST, PUT, DELETE
	RequestPath     *string   `json:"request_path,omitempty"`      // /api/users, /api/clients, etc
	RequestID       *string   `json:"request_id,omitempty"`        // Correlação entre requisições
	ResponseCode    *int      `json:"response_code,omitempty"`     // HTTP status code
	ExecutionTimeMs *int      `json:"execution_time_ms,omitempty"` // Tempo de execução em ms
}
