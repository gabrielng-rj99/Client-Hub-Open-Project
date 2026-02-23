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

package contractstore

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/repository"

	"github.com/google/uuid"
)

// ContractStats representa estatísticas de contratos por cliente
type ContractStats struct {
	ClientID          string `json:"client_id"`
	ActiveContracts   int    `json:"active_contracts"`
	ExpiredContracts  int    `json:"expired_contracts"`
	ArchivedContracts int    `json:"archived_contracts"`
}

// ContractStore é a nossa "caixa de ferramentas" para operações com a tabela contracts.
type ContractStore struct {
	db repository.DBInterface
}

// NewContractStore cria uma nova instância de ContractStore.
func NewContractStore(db repository.DBInterface) *ContractStore {
	return &ContractStore{
		db: db,
	}
}

// CreateContract insere um novo contrato no banco de dados.
func (s *ContractStore) CreateContract(contract domain.Contract) (string, error) {
	var trimmedModel, trimmedItemKey string
	var err error
	// Model is optional - only validate if provided
	if contract.Model != "" {
		trimmedModel, err = repository.ValidateName(contract.Model, 255)
		if err != nil {
			return "", err
		}
	}
	// ItemKey is optional - only validate if provided
	if contract.ItemKey != "" {
		trimmedItemKey, err = repository.ValidateName(contract.ItemKey, 255)
		if err != nil {
			return "", err
		}
	}

	// Start and End dates are now optional - only validate if both are provided
	if contract.StartDate != nil && contract.EndDate != nil {
		if contract.EndDate.Before(*contract.StartDate) || contract.EndDate.Equal(*contract.StartDate) {
			return "", errors.New("end date must be after start date")
		}
	}

	if contract.SubcategoryID == "" {
		return "", errors.New("line ID cannot be empty")
	}
	if contract.ClientID == "" {
		return "", errors.New("client ID cannot be empty")
	}

	// Check if line exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM subcategories WHERE id = $1", contract.SubcategoryID).Scan(&count)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", errors.New("line does not exist")
	}

	// Check if client exists e não está arquivado
	err = s.db.QueryRow("SELECT COUNT(*) FROM clients WHERE id = $1 AND archived_at IS NULL", contract.ClientID).Scan(&count)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", errors.New("client does not exist or is archived")
	}

	// If affiliateID is provided, check if affiliate exists and belongs to the client
	if contract.AffiliateID != nil && *contract.AffiliateID != "" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM affiliates WHERE id = $1 AND client_id = $2", *contract.AffiliateID, contract.ClientID).Scan(&count)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return "", errors.New("affiliate does not exist or does not belong to the specified client")
		}
	}

	// Check for duplicate product key only if provided
	if contract.ItemKey != "" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM contracts WHERE item_key = $1", contract.ItemKey).Scan(&count)
		if err != nil {
			return "", err
		}
		if count > 0 {
			return "", errors.New("product key already exists")
		}
	}

	// NOVA REGRA: Verificar sobreposição temporal apenas se ambas as datas forem fornecidas
	if contract.StartDate != nil && contract.EndDate != nil {
		var overlapCount int
		if contract.AffiliateID != nil && *contract.AffiliateID != "" {
			err = s.db.QueryRow(`
				SELECT COUNT(*) FROM contracts
				WHERE subcategory_id = $1 AND client_id = $2 AND affiliate_id = $3
				AND start_date IS NOT NULL AND end_date IS NOT NULL
				AND (
					(start_date <= $4 AND end_date >= $5) OR
					(start_date <= $6 AND end_date >= $7)
				)
			`, contract.SubcategoryID, contract.ClientID, *contract.AffiliateID,
				*contract.EndDate, *contract.EndDate,
				*contract.StartDate, *contract.StartDate,
			).Scan(&overlapCount)
		} else {
			err = s.db.QueryRow(`
				SELECT COUNT(*) FROM contracts
				WHERE subcategory_id = $1 AND client_id = $2 AND affiliate_id IS NULL
				AND start_date IS NOT NULL AND end_date IS NOT NULL
				AND (
					(start_date <= $3 AND end_date >= $4) OR
					(start_date <= $5 AND end_date >= $6)
				)
			`, contract.SubcategoryID, contract.ClientID,
				*contract.EndDate, *contract.EndDate,
				*contract.StartDate, *contract.StartDate,
			).Scan(&overlapCount)
		}
		if err != nil {
			return "", err
		}
		if overlapCount > 0 {
			return "", errors.New("contract period overlaps with another contract of the same type for this client/affiliate")
		}
	}

	newID := uuid.New().String()
	sqlStatement := `
		INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.db.Exec(sqlStatement,
		newID,
		trimmedModel,
		trimmedItemKey,
		nullTimeFromTime(contract.StartDate),
		nullTimeFromTime(contract.EndDate),
		contract.SubcategoryID,
		contract.ClientID,
		contract.AffiliateID,
	)
	if err != nil {
		return "", err
	}
	return newID, nil
}

// Helper function to convert *time.Time to sql.NullTime
func nullTimeFromTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// GetContractsByClientID fetches all contracts for a specific client.
func (s *ContractStore) GetContractsByClientID(clientID string) (contracts []domain.Contract, err error) {
	if clientID == "" {
		return nil, errors.New("client ID is required")
	}
	// Check if client exists and is not archived
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM clients WHERE id = $1 AND archived_at IS NULL", clientID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("client not found or archived")
	}

	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id FROM contracts WHERE client_id = $1`
	rows, err := s.db.Query(sqlStatement, clientID)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetContractsExpiringSoon busca contratos que expiram nos próximos 'days' dias.
func (s *ContractStore) GetContractsExpiringSoon(days int) (contracts []domain.Contract, err error) {
	now := time.Now()
	limitDate := now.AddDate(0, 0, days)

	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id
	                 FROM contracts
	                 WHERE end_date BETWEEN $1 AND $2 AND archived_at IS NULL`

	rows, err := s.db.Query(sqlStatement, now, limitDate)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetContractsNotStarted busca contratos que ainda não iniciaram (start_date no futuro)
func (s *ContractStore) GetContractsNotStarted() (contracts []domain.Contract, err error) {
	now := time.Now()

	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id
	                 FROM contracts
	                 WHERE start_date > $1 AND archived_at IS NULL`

	rows, err := s.db.Query(sqlStatement, now)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

func (s *ContractStore) UpdateContract(contract domain.Contract) error {
	if contract.ID == "" {
		return errors.New("contract ID cannot be empty")
	}

	// Check if contract exists and fetch current values for fields that might not be updated
	var currentModel, currentItemKey string
	var currentStartDate, currentEndDate sql.NullTime
	err := s.db.QueryRow("SELECT model, item_key, start_date, end_date FROM contracts WHERE id = $1", contract.ID).Scan(&currentModel, &currentItemKey, &currentStartDate, &currentEndDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("contract does not exist")
		}
		return err
	}

	var trimmedModel, trimmedItemKey string

	// Model is optional - if provided and not empty, validate it; otherwise keep current
	if contract.Model != "" {
		trimmedModel, err = repository.ValidateName(contract.Model, 255)
		if err != nil {
			return err
		}
	} else {
		trimmedModel = currentModel
	}

	// ItemKey is optional - if provided and not empty, validate it; otherwise keep current
	if contract.ItemKey != "" {
		trimmedItemKey, err = repository.ValidateName(contract.ItemKey, 255)
		if err != nil {
			return err
		}
	} else {
		trimmedItemKey = currentItemKey
	}

	// Handle dates: if not provided, use current values
	var startDate, endDate sql.NullTime
	if contract.StartDate != nil {
		startDate = nullTimeFromTime(contract.StartDate)
	} else {
		startDate = currentStartDate
	}

	if contract.EndDate != nil {
		endDate = nullTimeFromTime(contract.EndDate)
	} else {
		endDate = currentEndDate
	}

	// Validate date logic: if both are provided/valid, end_date must be after start_date
	if startDate.Valid && endDate.Valid {
		if endDate.Time.Before(startDate.Time) || endDate.Time.Equal(startDate.Time) {
			return errors.New("end date must be after start date")
		}
	}

	// If affiliateID is provided, check if affiliate exists and belongs to the client
	if contract.AffiliateID != nil && *contract.AffiliateID != "" {
		var depCount int
		err = s.db.QueryRow("SELECT COUNT(*) FROM affiliates WHERE id = $1 AND client_id = $2", *contract.AffiliateID, contract.ClientID).Scan(&depCount)
		if err != nil {
			return err
		}
		if depCount == 0 {
			return errors.New("affiliate does not exist or does not belong to the specified client")
		}
	}

	sqlStatement := `UPDATE contracts SET model = $1, item_key = $2, start_date = $3, end_date = $4 WHERE id = $5`
	result, err := s.db.Exec(sqlStatement, trimmedModel, trimmedItemKey, startDate, endDate, contract.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no contract updated")
	}
	return nil
}

// GetContractStatsByClientID retorna estatísticas de contratos de um cliente específico
func (s *ContractStore) GetContractStatsByClientID(clientID string) (*ContractStats, error) {
	if clientID == "" {
		return nil, errors.New("client ID cannot be empty")
	}

	stats := &ContractStats{
		ClientID: clientID,
	}

	now := time.Now()

	// Count active contracts (not archived and not expired)
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM contracts
		WHERE client_id = $1
		AND archived_at IS NULL
		AND end_date > $2
	`, clientID, now).Scan(&stats.ActiveContracts)
	if err != nil {
		return nil, err
	}

	// Count expired contracts (not archived but expired)
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM contracts
		WHERE client_id = $1
		AND archived_at IS NULL
		AND end_date <= $2
	`, clientID, now).Scan(&stats.ExpiredContracts)
	if err != nil {
		return nil, err
	}

	// Count archived contracts
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM contracts
		WHERE client_id = $1
		AND archived_at IS NOT NULL
	`, clientID).Scan(&stats.ArchivedContracts)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetContractStatsForAllClients retorna estatísticas de contratos para todos os clientes
func (s *ContractStore) GetContractStatsForAllClients() (map[string]*ContractStats, error) {
	now := time.Now()

	rows, err := s.db.Query(`
		SELECT
			client_id,
			COUNT(CASE WHEN archived_at IS NULL AND (end_date IS NULL OR end_date > $1) THEN 1 END) as active,
			COUNT(CASE WHEN archived_at IS NULL AND end_date IS NOT NULL AND end_date <= $1 THEN 1 END) as expired,
			COUNT(CASE WHEN archived_at IS NOT NULL THEN 1 END) as archived
		FROM contracts
		GROUP BY client_id
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statsMap := make(map[string]*ContractStats)
	for rows.Next() {
		var stats ContractStats
		if err := rows.Scan(&stats.ClientID, &stats.ActiveContracts, &stats.ExpiredContracts, &stats.ArchivedContracts); err != nil {
			return nil, err
		}
		statsMap[stats.ClientID] = &stats
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statsMap, nil
}

// GetContractStatsForClientIDs retorna estatísticas de contratos para uma lista de clientes
func (s *ContractStore) GetContractStatsForClientIDs(clientIDs []string) (map[string]*ContractStats, error) {
	statsMap := make(map[string]*ContractStats)
	if len(clientIDs) == 0 {
		return statsMap, nil
	}

	placeholders := make([]string, 0, len(clientIDs))
	args := make([]interface{}, 0, len(clientIDs)+1)
	for i, id := range clientIDs {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
		args = append(args, id)
	}
	now := time.Now()
	args = append(args, now)
	nowPlaceholder := "$" + strconv.Itoa(len(args))

	query := `
		SELECT
			client_id,
			COUNT(CASE WHEN archived_at IS NULL AND (end_date IS NULL OR end_date > ` + nowPlaceholder + `) THEN 1 END) as active,
			COUNT(CASE WHEN archived_at IS NULL AND end_date IS NOT NULL AND end_date <= ` + nowPlaceholder + ` THEN 1 END) as expired,
			COUNT(CASE WHEN archived_at IS NOT NULL THEN 1 END) as archived
		FROM contracts
		WHERE client_id IN (` + strings.Join(placeholders, ", ") + `)
		GROUP BY client_id
	`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stats ContractStats
		if err := rows.Scan(&stats.ClientID, &stats.ActiveContracts, &stats.ExpiredContracts, &stats.ArchivedContracts); err != nil {
			return nil, err
		}
		statsMap[stats.ClientID] = &stats
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statsMap, nil
}

// GetContractByID busca um contrato específico pelo seu ID
func (s *ContractStore) GetContractByID(id string) (*domain.Contract, error) {
	if id == "" {
		return nil, errors.New("contract ID cannot be empty")
	}
	sqlStatement := `
		SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id, archived_at
		FROM contracts
		WHERE id = $1`

	var contract domain.Contract
	var startDate, endDate, archivedAt sql.NullTime
	err := s.db.QueryRow(sqlStatement, id).Scan(
		&contract.ID,
		&contract.Model,
		&contract.ItemKey,
		&startDate,
		&endDate,
		&contract.SubcategoryID,
		&contract.ClientID,
		&contract.AffiliateID,
		&archivedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if startDate.Valid {
		t := startDate.Time
		contract.StartDate = &t
	}
	if endDate.Valid {
		t := endDate.Time
		contract.EndDate = &t
	}
	if archivedAt.Valid {
		contract.ArchivedAt = &archivedAt.Time
	}

	return &contract, nil
}

// GetContractStatus returns explicit status for a contract (ativo, expirando, expirado)
// Uses time.Now() internally, so timing can matter in tests
// EndDate nil = infinito superior (nunca expira)
// StartDate nil = infinito inferior (começou sempre)
func GetContractStatus(contract domain.Contract) string {
	now := time.Now()

	// EndDate nil = nunca expira, sempre ativo
	if contract.EndDate == nil {
		return "ativo"
	}

	if contract.EndDate.Before(now) {
		return "expirado"
	}

	// Contract is expiring if it expires within 30 days
	daysUntilExpiry := contract.EndDate.Sub(now).Hours() / 24
	if daysUntilExpiry > 0 && daysUntilExpiry <= 30 {
		return "expirando"
	}

	return "ativo"
}

func (s *ContractStore) DeleteContract(id string) error {
	if id == "" {
		return errors.New("contract ID cannot be empty")
	}
	// Check if contract exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM contracts WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("contract does not exist")
	}
	sqlStatement := `DELETE FROM contracts WHERE id = $1`
	result, err := s.db.Exec(sqlStatement, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no contract deleted")
	}
	return nil
}

// ArchiveContract arquiva um contrato definindo archived_at = NOW().
func (s *ContractStore) ArchiveContract(id string) error {
	if id == "" {
		return errors.New("contract ID cannot be empty")
	}
	// Check if contract exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM contracts WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("contract not found")
	}
	// Check if already archived
	var archivedAt sql.NullTime
	err = s.db.QueryRow("SELECT archived_at FROM contracts WHERE id = $1", id).Scan(&archivedAt)
	if err != nil {
		return err
	}
	if archivedAt.Valid {
		return errors.New("contract is already archived")
	}
	sqlStatement := `UPDATE contracts SET archived_at = $1 WHERE id = $2`
	result, err := s.db.Exec(sqlStatement, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no contract archived")
	}
	return nil
}

// UnarchiveContract desarquiva um contrato definindo archived_at = NULL.
func (s *ContractStore) UnarchiveContract(id string) error {
	if id == "" {
		return errors.New("contract ID cannot be empty")
	}
	// Check if contract exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM contracts WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("contract not found")
	}
	// Check if actually archived
	var archivedAt sql.NullTime
	err = s.db.QueryRow("SELECT archived_at FROM contracts WHERE id = $1", id).Scan(&archivedAt)
	if err != nil {
		return err
	}
	if !archivedAt.Valid {
		return errors.New("contract is not archived")
	}
	sqlStatement := `UPDATE contracts SET archived_at = NULL WHERE id = $1`
	result, err := s.db.Exec(sqlStatement, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no contract unarchived")
	}
	return nil
}

// GetAllContracts fetches all contracts in the system (excluding archived)
func (s *ContractStore) GetAllContracts() (contracts []domain.Contract, err error) {
	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id
	                 FROM contracts WHERE archived_at IS NULL`
	rows, err := s.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetAllContractsIncludingArchived fetches all contracts in the system (including archived and cancelled)
func (s *ContractStore) GetAllContractsIncludingArchived() (contracts []domain.Contract, err error) {
	sqlStatement := `
		SELECT
			c.id, c.model, c.item_key, c.start_date, c.end_date,
			c.subcategory_id, c.client_id, c.affiliate_id, c.archived_at,
			COALESCE(cl.name, '') AS client_name,
			cl.nickname AS client_nickname,
			COALESCE(cat.name, '') AS category_name,
			COALESCE(s.name, '') AS subcategory_name
		FROM contracts c
		LEFT JOIN clients cl ON cl.id = c.client_id
		LEFT JOIN subcategories s ON s.id = c.subcategory_id
		LEFT JOIN categories cat ON cat.id = s.category_id
	`
	rows, err := s.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate, archivedAt sql.NullTime
		var clientNickname sql.NullString
		if err = rows.Scan(
			&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate,
			&c.SubcategoryID, &c.ClientID, &c.AffiliateID, &archivedAt,
			&c.ClientName, &clientNickname, &c.CategoryName, &c.SubcategoryName,
		); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		if archivedAt.Valid {
			c.ArchivedAt = &archivedAt.Time
		}
		if clientNickname.Valid {
			c.ClientNickname = &clientNickname.String
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetContractsByName fetches contracts by partial name/model (case-insensitive)
func (s *ContractStore) GetContractsByName(name string) ([]domain.Contract, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	// Busca case-insensitive já funcionará com CITEXT
	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id
	                 FROM contracts WHERE model LIKE $1`
	likePattern := "%" + name + "%"
	rows, err := s.db.Query(sqlStatement, likePattern)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()
	var contracts []domain.Contract
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetContractsBySubcategoryID fetches all contracts for a specific line
func (s *ContractStore) GetContractsBySubcategoryID(subcategoryID string) (contracts []domain.Contract, err error) {
	if subcategoryID == "" {
		return nil, errors.New("line ID is required")
	}
	// Check if line exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM subcategories WHERE id = $1", subcategoryID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("line not found")
	}

	sqlStatement := `SELECT id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id
	                 FROM contracts WHERE subcategory_id = $1`
	rows, err := s.db.Query(sqlStatement, subcategoryID)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}

// GetContractsByCategoryID fetches all contracts for a specific category
func (s *ContractStore) GetContractsByCategoryID(categoryID string) (contracts []domain.Contract, err error) {
	if categoryID == "" {
		return nil, errors.New("category ID is required")
	}
	// Check if category exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", categoryID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("category not found")
	}

	sqlStatement := `
		SELECT c.id, c.model, c.item_key, c.start_date, c.end_date, c.subcategory_id, c.client_id, c.affiliate_id
		FROM contracts c
		INNER JOIN subcategories ln ON c.subcategory_id = ln.id
		WHERE ln.category_id = $1`
	rows, err := s.db.Query(sqlStatement, categoryID)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := rows.Close()
		if err == nil {
			err = closeErr
		}
	}()

	contracts = []domain.Contract{}
	for rows.Next() {
		var c domain.Contract
		var startDate, endDate sql.NullTime
		if err = rows.Scan(&c.ID, &c.Model, &c.ItemKey, &startDate, &endDate, &c.SubcategoryID, &c.ClientID, &c.AffiliateID); err != nil {
			return nil, err
		}
		if startDate.Valid {
			t := startDate.Time
			c.StartDate = &t
		}
		if endDate.Valid {
			t := endDate.Time
			c.EndDate = &t
		}
		contracts = append(contracts, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return contracts, nil
}
