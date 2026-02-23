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
	"Open-Generic-Hub/backend/domain"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func insertTestDependencies(db *sql.DB) (string, string, string, string, error) {
	uniqueID := uuid.New().String()[:8]
	clientID, err := InsertTestClient(db, "Test Client "+uniqueID, generateUniqueCNPJ())
	if err != nil {
		return "", "", "", "", err
	}
	// Helper para inserir entidade no banco
	subClientID := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO affiliates (id, name, client_id) VALUES ($1, $2, $3)",
		subClientID,
		"Test Affiliate "+uniqueID,
		clientID,
	)
	if err != nil {
		return "", "", "", "", err
	}
	categoryID, err := InsertTestCategory(db, "Test Category "+uniqueID)
	if err != nil {
		return "", "", "", "", err
	}
	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory "+uniqueID, categoryID)
	if err != nil {
		return "", "", "", "", err
	}
	return clientID, subClientID, categoryID, subcategoryID, nil
}

// Teste: Não pode mover linha entre categorias
func TestCannotMoveLineBetweenCategories(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	categoryID1, err := InsertTestCategory(db, "Categoria 1")
	if err != nil {
		t.Fatalf("Failed to insert test category 1: %v", err)
	}
	categoryID2, err := InsertTestCategory(db, "Categoria 2")
	if err != nil {
		t.Fatalf("Failed to insert test category 2: %v", err)
	}

	subcategoryStore := NewSubcategoryStore(db)
	line := domain.Subcategory{
		Name:       "Linha Teste",
		CategoryID: categoryID1,
	}
	subcategoryID, err := subcategoryStore.CreateSubcategory(line)
	if err != nil {
		t.Fatalf("Failed to create line: %v", err)
	}

	// Tentar mover linha para outra categoria
	lineUpdate := domain.Subcategory{
		ID:         subcategoryID,
		Name:       "Linha Teste",
		CategoryID: categoryID2,
	}
	err = subcategoryStore.UpdateSubcategory(lineUpdate)
	if err == nil {
		t.Error("Expected error when moving line between categories, got none")
	}
}

// Teste: Não pode deletar linha com contratos associadas
func TestDeleteSubcategoryWithContractsAssociated(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}
	categoryID, err := InsertTestCategory(db, "Categoria Teste")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}
	subcategoryStore := NewSubcategoryStore(db)
	line := domain.Subcategory{
		Name:       "Linha Teste",
		CategoryID: categoryID,
	}
	subcategoryID, err := subcategoryStore.CreateSubcategory(line)
	if err != nil {
		t.Fatalf("Failed to create line: %v", err)
	}

	contractStore := NewContractStore(db)
	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	contract := domain.Contract{
		Model:         "Licença Teste",
		ItemKey:       "LINE-DEL-KEY-001",
		StartDate:     timePtr(startDate),
		EndDate:       timePtr(endDate),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}
	_, err = contractStore.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	err = subcategoryStore.DeleteSubcategory(subcategoryID)
	if err == nil {
		t.Error("Expected error when deleting line with contracts associated, got none")
	}
}

func TestCreateContract(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := timePtr(time.Now())
	endDate := timePtr(time.Now().AddDate(1, 0, 0))

	tests := []struct {
		name        string
		contract    domain.Contract
		expectError bool
	}{
		{
			name: "sucesso - criação normal com unidade",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-125",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
				AffiliateID:   &subClientID,
			},
			expectError: false,
		},
		{
			name: "sucesso - criação sem unidade",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-124",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: false,
		},
		{
			name: "erro - nome vazio",
			contract: domain.Contract{
				ItemKey:       "TEST-KEY-125",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "erro - chave do produto vazia",
			contract: domain.Contract{
				Model:         "Test Contract",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "erro - data final antes da inicial",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-126",
				StartDate:     endDate,
				EndDate:       startDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "erro - sobreposição de datas para mesma empresa/unidade/tipo",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-OVERLAP",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
				AffiliateID:   &subClientID,
			},
			expectError: true,
		},
		{
			name: "erro - atualização com StartDate após EndDate",
			contract: domain.Contract{
				Model:         "Test Contract Update",
				ItemKey:       "TEST-KEY-UPDATE",
				StartDate:     timePtr(endDate.AddDate(2, 0, 0)),
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "erro - tipo inválido",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-127",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: "invalid-line",
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "erro - empresa inválida",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-128",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      "invalid-client",
			},
			expectError: true,
		},
		{
			name: "erro - empresa arquivada",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-ARCHIVED",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID, // Usar clientID válido
			},
			expectError: true,
		},
		{
			name: "erro - unidade inválida",
			contract: domain.Contract{
				Model:         "Test Contract",
				ItemKey:       "TEST-KEY-129",
				StartDate:     startDate,
				EndDate:       endDate,
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
				AffiliateID:   func() *string { s := "invalid-affiliate"; return &s }(),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contractStore := NewContractStore(db)
			id, err := contractStore.CreateContract(tt.contract)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && id == "" {
				t.Error("Expected ID but got empty string")
			}
		})
	}
}

// Função de utilitário para status de licença deve estar fora do TestCreateContract!
func TestContractStatusUtil(t *testing.T) {
	now := time.Now()
	contractActive := domain.Contract{
		EndDate: timePtr(now.AddDate(0, 2, 0)), // expira em 2 meses (fora da janela de "expirando")
	}
	contractExpiring := domain.Contract{
		EndDate: timePtr(now.AddDate(0, 0, 10)), // expira em 10 dias
	}
	contractExpired := domain.Contract{
		EndDate: timePtr(now.AddDate(0, 0, -1)), // já expirou
	}

	statusActive := GetContractStatus(contractActive)
	statusExpiring := GetContractStatus(contractExpiring)
	statusExpired := GetContractStatus(contractExpired)

	if statusActive != "ativo" {
		t.Errorf("Expected status 'ativo', got '%s'", statusActive)
	}
	if statusExpiring != "expirando" {
		t.Errorf("Expected status 'expirando', got '%s'", statusExpiring)
	}
	if statusExpired != "expirado" {
		t.Errorf("Expected status 'expirado', got '%s'", statusExpired)
	}
}

func TestUpdateContractEdgeCases(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	contractStore := NewContractStore(db)
	contract := domain.Contract{
		Model:         "Edge Contract",
		ItemKey:       "EDGE-KEY-1",
		StartDate:     timePtr(startDate),
		EndDate:       timePtr(endDate),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
		AffiliateID:   &subClientID,
	}
	contractID, err := contractStore.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract for update edge case: %v", err)
	}

	// Atualizar com datas invertidas
	contract.ID = contractID
	contract.StartDate = timePtr(endDate.AddDate(2, 0, 0))
	contract.EndDate = timePtr(endDate)
	err = contractStore.UpdateContract(contract)
	if err == nil {
		t.Error("Expected error when updating contract with StartDate after EndDate, got none")
	}

	// Atualizar licença inexistente
	contract.ID = uuid.New().String()
	contract.StartDate = timePtr(startDate)
	contract.EndDate = timePtr(endDate)
	contractStore = NewContractStore(db)
	err = contractStore.UpdateContract(contract)
	if err == nil {
		t.Error("Expected error when updating non-existent contract, got none")
	}
}

func TestDeleteContractEdgeCases(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	contractStore := NewContractStore(db)
	// Tentar deletar licença inexistente
	err = contractStore.DeleteContract(uuid.New().String())
	if err == nil {
		t.Error("Expected error when deleting non-existent contract, got none")
	}
}

func TestDeleteClientEdgeCases(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	clientStore := NewClientStore(db)
	// Tentar deletar empresa inexistente
	err = clientStore.DeleteClientPermanently(uuid.New().String())
	if err == nil {
		t.Error("Expected error when deleting non-existent client, got none")
	}
}

func TestGetContractByID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	contractID := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		contractID,
		"Test Contract",
		"TEST-KEY-123",
		startDate,
		endDate,
		subcategoryID,
		clientID,
		subClientID,
	)
	if err != nil {
		t.Fatalf("Failed to insert test contract: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		expectError bool
		expectFound bool
	}{
		{
			name:        "sucesso - licença encontrada",
			id:          contractID,
			expectError: false,
			expectFound: true,
		},
		{
			name:        "erro - id vazio",
			id:          "",
			expectError: true,
			expectFound: false,
		},
		{
			name:        "não encontrado - id inexistente",
			id:          uuid.New().String(),
			expectError: false,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contractStore := NewContractStore(db)
			contract, err := contractStore.GetContractByID(tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && contract == nil {
				t.Error("Expected contract but got nil")
			}
			if !tt.expectFound && contract != nil {
				t.Error("Expected no contract but got one")
			}
		})
	}
}

func TestGetContractsByClientID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)

	for i := 0; i < 3; i++ {
		contractID := uuid.New().String()
		_, err := db.Exec(
			"INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			contractID,
			"Test Contract "+string(rune('A'+i)),
			"TEST-KEY-"+uuid.New().String()[:8],
			startDate,
			endDate,
			subcategoryID,
			clientID,
			subClientID,
		)
		if err != nil {
			t.Fatalf("Failed to insert test contract: %v", err)
		}
	}

	tests := []struct {
		name        string
		clientID    string
		expectError bool
		expectCount int
	}{
		{
			name:        "sucesso - contratos encontradas",
			clientID:    clientID,
			expectError: false,
			expectCount: 3,
		},
		{
			name:        "erro - empresa vazia",
			clientID:    "",
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "erro - empresa não existe",
			clientID:    "non-existent-client",
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contractStore := NewContractStore(db)
			contracts, err := contractStore.GetContractsByClientID(tt.clientID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(contracts) != tt.expectCount {
				t.Errorf("Expected %d contracts but got %d", tt.expectCount, len(contracts))
			}
		})
	}
}

func TestUpdateContract(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	contractID := uuid.New().String()
	contractIDForEmptyName := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		contractID,
		"Test Contract",
		"TEST-KEY-UPDATE",
		startDate,
		endDate,
		subcategoryID,
		clientID,
		subClientID,
	)
	if err != nil {
		t.Fatalf("Failed to insert test contract: %v", err)
	}
	// Create a separate contract for the empty name test
	_, err = db.Exec(
		"INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		contractIDForEmptyName,
		"Original Contract Name",
		"TEST-KEY-EMPTY",
		startDate,
		endDate,
		subcategoryID,
		clientID,
		subClientID,
	)
	if err != nil {
		t.Fatalf("Failed to insert test contract for empty name: %v", err)
	}

	tests := []struct {
		name        string
		contract    domain.Contract
		expectError bool
	}{
		{
			name: "sucesso - atualização normal",
			contract: domain.Contract{
				ID:            contractID,
				Model:         "Updated Contract",
				ItemKey:       "TEST-KEY-123",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate.AddDate(1, 0, 0)),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
				AffiliateID:   &subClientID,
			},
			expectError: false,
		},
		{
			name: "erro - id vazio",
			contract: domain.Contract{
				Model:         "Updated Contract",
				ItemKey:       "TEST-KEY-123",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
		{
			name: "sucesso - nome vazio mantém valor atual",
			contract: domain.Contract{
				ID:            contractIDForEmptyName,
				ItemKey:       "TEST-KEY-EMPTY-KEEP",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: false,
		},
		{
			name: "erro - data final antes da inicial",
			contract: domain.Contract{
				ID:            contractID,
				Model:         "Updated Contract",
				ItemKey:       "TEST-KEY-FINAL",
				StartDate:     timePtr(endDate),
				EndDate:       timePtr(startDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contractStore := NewContractStore(db)
			err := contractStore.UpdateContract(tt.contract)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				var model string
				err = db.QueryRow("SELECT model FROM contracts WHERE id = $1", tt.contract.ID).Scan(&model)
				if err != nil {
					t.Errorf("Failed to query updated contract: %v", err)
				}
				// If Model is empty, it should keep the original value
				expectedModel := tt.contract.Model
				if expectedModel == "" {
					expectedModel = "Original Contract Name"
				}
				if model != expectedModel {
					t.Errorf("Expected model %q but got %q", expectedModel, model)
				}
			}
		})
	}
}

func TestDeleteContract(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)
	contractID := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO contracts (id, model, item_key, start_date, end_date, subcategory_id, client_id, affiliate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		contractID,
		"Test Contract",
		"TEST-KEY-DELETE",
		startDate,
		endDate,
		subcategoryID,
		clientID,
		subClientID,
	)
	if err != nil {
		t.Fatalf("Failed to insert test contract: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		expectError bool
	}{
		{
			name:        "sucesso - deleção normal",
			id:          contractID,
			expectError: false,
		},
		{
			name:        "erro - id vazio",
			id:          "",
			expectError: true,
		},
		{
			name:        "erro - id inexistente",
			id:          uuid.New().String(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contractStore := NewContractStore(db)
			err := contractStore.DeleteContract(tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM contracts WHERE id = $1", tt.id).Scan(&count)
				if err != nil {
					t.Errorf("Failed to query deleted contract: %v", err)
				}
				if count != 0 {
					t.Error("Expected contract to be deleted, but it still exists")
				}
			}
		})
	}
}

// Helper function to create a string pointer

// ============================================================================
// CRITICAL TESTS
// ============================================================================

func setupContractTestDB(t *testing.T) *sql.DB {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}
	return db
}

// TestGetContractsExpiringSoon tests retrieving contracts expiring within a specified number of days
func TestGetContractsExpiringSoon(t *testing.T) {
	db := setupContractTestDB(t)
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Insert test data
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line: %v", err)
	}

	now := time.Now()

	// Insert contracts with different expiration dates
	// Contract 1: expires in 10 days (should be included in 30-day search)
	_, err = InsertTestContract(db, "Contract 1", "KEY-1", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 10)), subcategoryID, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 1: %v", err)
	}

	// Contract 2: expires in 60 days (should NOT be included in 30-day search)
	_, err = InsertTestContract(db, "Contract 2", "KEY-2", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 60)), subcategoryID, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 2: %v", err)
	}

	// Contract 3: already expired (should NOT be included)
	_, err = InsertTestContract(db, "Contract 3", "KEY-3", timePtr(now.AddDate(0, 0, -30)), timePtr(now.AddDate(0, 0, -5)), subcategoryID, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 3: %v", err)
	}

	// Contract 4: expires in 25 days (should be included in 30-day search)
	_, err = InsertTestContract(db, "Contract 4", "KEY-4", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 25)), subcategoryID, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 4: %v", err)
	}

	// Get contracts expiring in 30 days
	contracts, err := contractStore.GetContractsExpiringSoon(30)
	if err != nil {
		t.Fatalf("Failed to get expiring contracts: %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("Expected 2 expiring contracts, got %d", len(contracts))
	}

	// Verify correct contracts are returned
	expiredKeys := make(map[string]bool)
	for _, l := range contracts {
		expiredKeys[l.ItemKey] = true
	}

	if !expiredKeys["KEY-1"] || !expiredKeys["KEY-4"] {
		t.Error("Expected contracts KEY-1 and KEY-4 in results")
	}
}

// TestGetContractsBySubcategoryIDCritical tests retrieving contracts by line ID
func TestGetContractsBySubcategoryIDCritical(t *testing.T) {
	db := setupContractTestDB(t)
	defer CloseDB(db)

	contractStore := NewContractStore(db)

	// Insert test data
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID1, err := InsertTestSubcategory(db, "Subcategory 1", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line 1: %v", err)
	}

	subcategoryID2, err := InsertTestSubcategory(db, "Subcategory 2", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line 2: %v", err)
	}

	now := time.Now()

	// Insert contracts for different subcategories
	_, err = InsertTestContract(db, "Contract 1", "KEY-1", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID1, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 1: %v", err)
	}

	_, err = InsertTestContract(db, "Contract 2", "KEY-2", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID1, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 2: %v", err)
	}

	_, err = InsertTestContract(db, "Contract 3", "KEY-3", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID2, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 3: %v", err)
	}

	// Get contracts for subcategoryID1
	contracts, err := contractStore.GetContractsBySubcategoryID(subcategoryID1)
	if err != nil {
		t.Fatalf("Failed to get contracts by line: %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("Expected 2 contracts for line 1, got %d", len(contracts))
	}

	// Verify all returned contracts belong to subcategoryID1
	for _, l := range contracts {
		if l.SubcategoryID != subcategoryID1 {
			t.Errorf("Expected line ID '%s', got '%s'", subcategoryID1, l.SubcategoryID)
		}
	}
}

// TestGetContractsByCategoryIDCritical tests retrieving contracts by category ID
func TestGetContractsByCategoryIDCritical(t *testing.T) {
	db := setupContractTestDB(t)
	defer CloseDB(db)

	contractStore := NewContractStore(db)

	// Insert test data
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID1, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category 1: %v", err)
	}

	categoryID2, err := InsertTestCategory(db, "Hardware")
	if err != nil {
		t.Fatalf("Failed to insert test category 2: %v", err)
	}

	subcategoryID1, err := InsertTestSubcategory(db, "Subcategory 1", categoryID1)
	if err != nil {
		t.Fatalf("Failed to insert test line 1: %v", err)
	}

	subcategoryID2, err := InsertTestSubcategory(db, "Subcategory 2", categoryID2)
	if err != nil {
		t.Fatalf("Failed to insert test line 2: %v", err)
	}

	now := time.Now()

	// Insert contracts for different categories
	_, err = InsertTestContract(db, "Contract 1", "KEY-1", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID1, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 1: %v", err)
	}

	_, err = InsertTestContract(db, "Contract 2", "KEY-2", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID1, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 2: %v", err)
	}

	_, err = InsertTestContract(db, "Contract 3", "KEY-3", timePtr(now.AddDate(0, 0, -10)), timePtr(now.AddDate(0, 0, 30)), subcategoryID2, clientID, nil)
	if err != nil {
		t.Fatalf("Failed to insert contract 3: %v", err)
	}

	// Get contracts for categoryID1
	contracts, err := contractStore.GetContractsByCategoryID(categoryID1)
	if err != nil {
		t.Fatalf("Failed to get contracts by category: %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("Expected 2 contracts for category 1, got %d", len(contracts))
	}

	// Verify all returned contracts belong to categoryID1
	for _, l := range contracts {
		if l.SubcategoryID != subcategoryID1 {
			t.Errorf("Expected line ID '%s', got '%s'", subcategoryID1, l.SubcategoryID)
		}
	}
}

// TestCreateContractWithOverlap tests contract creation fails when there's a time overlap
func TestCreateContractWithOverlap(t *testing.T) {
	db := setupContractTestDB(t)
	defer CloseDB(db)

	contractStore := NewContractStore(db)

	// Insert test data
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line: %v", err)
	}

	now := time.Now()

	// Create first contract
	firstContract := domain.Contract{
		Model:         "First Contract",
		ItemKey:       "KEY-1",
		StartDate:     timePtr(now.AddDate(0, 0, -10)),
		EndDate:       timePtr(now.AddDate(0, 0, 30)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := contractStore.CreateContract(firstContract)
	if err != nil {
		t.Fatalf("Failed to create first contract: %v", err)
	}
	if id == "" {
		t.Error("Expected contract ID")
	}

	// Try to create overlapping contract
	overlappingContract := domain.Contract{
		Model:         "Overlapping Contract",
		ItemKey:       "KEY-2",
		StartDate:     timePtr(now.AddDate(0, 0, 5)),  // Within first contract period
		EndDate:       timePtr(now.AddDate(0, 0, 20)), // Within first contract period
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	_, err = contractStore.CreateContract(overlappingContract)
	if err == nil {
		t.Error("Expected error creating overlapping contract")
	}
}

// TestCreateContractNonOverlappingValid tests that non-overlapping contracts can be created
func TestCreateContractNonOverlappingValid(t *testing.T) {
	db := setupContractTestDB(t)
	defer CloseDB(db)

	contractStore := NewContractStore(db)

	// Insert test data
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line: %v", err)
	}

	now := time.Now()

	// Create first contract
	firstContract := domain.Contract{
		Model:         "First Contract",
		ItemKey:       "KEY-1",
		StartDate:     timePtr(now.AddDate(0, 0, -30)),
		EndDate:       timePtr(now.AddDate(0, 0, -5)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := contractStore.CreateContract(firstContract)
	if err != nil {
		t.Fatalf("Failed to create first contract: %v", err)
	}
	if id == "" {
		t.Error("Expected contract ID")
	}

	// Create non-overlapping contract (starts after first ends)
	secondContract := domain.Contract{
		Model:         "Second Contract",
		ItemKey:       "KEY-2",
		StartDate:     timePtr(now.AddDate(0, 0, -4)),
		EndDate:       timePtr(now.AddDate(0, 0, 30)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id2, err := contractStore.CreateContract(secondContract)
	if err != nil {
		t.Fatalf("Failed to create second contract: %v", err)
	}
	if id2 == "" {
		t.Error("Expected contract ID for second contract")
	}
}

// ============================================================================
// VALIDATION & EDGE CASES
// ============================================================================

// TestCreateContractWithInvalidNames tests various invalid contract model names
func TestCreateContractWithInvalidNames(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	contractStore := NewContractStore(db)
	startDate := time.Now()
	var clientID, categoryID, subcategoryID string

	// Generate long names
	longName := strings.Repeat("a", 256)
	veryLongName := strings.Repeat("a", 1000)

	tests := []struct {
		name        string
		model       string
		expectError bool
		description string
	}{
		{
			name:        "invalid - model name too long (256 chars)",
			model:       longName,
			expectError: true,
			description: "Contract model with 256 characters should be rejected",
		},
		{
			name:        "invalid - model name way too long (1000 chars)",
			model:       veryLongName,
			expectError: true,
			description: "Contract model with 1000 characters should be rejected",
		},
		{
			name:        "invalid - model name with only spaces",
			model:       "     ",
			expectError: true,
			description: "Contract model with only whitespace should be rejected",
		},
		{
			name:        "valid - model name at max length (255 chars)",
			model:       strings.Repeat("a", 255),
			expectError: false,
			description: "Contract model with exactly 255 characters should be allowed",
		},
		{
			name:        "valid - model name with special characters",
			model:       "License Pro - Enterprise Edition (2025)",
			expectError: false,
			description: "Contract model with special characters should be allowed",
		},
		{
			name:        "valid - model name with accents",
			model:       "Licença Profissional",
			expectError: false,
			description: "Contract model with Portuguese accents should be allowed",
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each test to avoid conflicts
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			// Re-insert dependencies after clearing tables
			var errInsert error
			clientID, errInsert = InsertTestClient(db, "Test Client", generateUniqueCNPJ())
			if errInsert != nil {
				t.Fatalf("Failed to insert test client: %v", errInsert)
			}

			categoryID, errInsert = InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
			if errInsert != nil {
				t.Fatalf("Failed to insert test category: %v", errInsert)
			}

			subcategoryID, errInsert = InsertTestSubcategory(db, "Test Subcategory", categoryID)
			if errInsert != nil {
				t.Fatalf("Failed to insert test line: %v", errInsert)
			}

			// Use different time periods for each contract to avoid overlap detection
			contractStart := startDate.AddDate(0, 0, idx*30)
			contractEnd := contractStart.AddDate(1, 0, 0)

			contract := domain.Contract{
				Model:         tt.model,
				ItemKey:       "TEST-KEY-" + string(rune('A'+idx)),
				StartDate:     timePtr(contractStart),
				EndDate:       timePtr(contractEnd),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			}

			_, errCreate := contractStore.CreateContract(contract)

			if tt.expectError && errCreate == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && errCreate != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, errCreate)
			}
		})
	}
}

// TestCreateContractWithInvalidItemKeys tests various invalid product key formats
func TestCreateContractWithInvalidItemKeys(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	contractStore := NewContractStore(db)
	startDate := time.Now()
	var clientID, categoryID, subcategoryID string

	longKey := strings.Repeat("A", 256)
	veryLongKey := strings.Repeat("A", 1000)

	tests := []struct {
		name        string
		itemKey     string
		expectError bool
		description string
	}{
		{
			name:        "invalid - product key too long (256 chars)",
			itemKey:     longKey,
			expectError: true,
			description: "Product key with 256 characters should be rejected",
		},
		{
			name:        "invalid - product key way too long (1000 chars)",
			itemKey:     veryLongKey,
			expectError: true,
			description: "Product key with 1000 characters should be rejected",
		},
		{
			name:        "invalid - product key with only spaces",
			itemKey:     "     ",
			expectError: true,
			description: "Product key with only whitespace should be rejected",
		},
		{
			name:        "valid - product key at max length (255 chars)",
			itemKey:     strings.Repeat("K", 255),
			expectError: false,
			description: "Product key with exactly 255 characters should be allowed",
		},
		{
			name:        "valid - product key with special characters",
			itemKey:     "ABCD-1234-EFGH-5678",
			expectError: false,
			description: "Product key with dashes should be allowed",
		},
		{
			name:        "valid - product key with mixed case",
			itemKey:     "AbCd-1234-EfGh-5678",
			expectError: false,
			description: "Product key with mixed case should be allowed",
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each test to avoid conflicts
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			// Re-insert dependencies after clearing tables
			var errInsert error
			clientID, errInsert = InsertTestClient(db, "Test Client", generateUniqueCNPJ())
			if errInsert != nil {
				t.Fatalf("Failed to insert test client: %v", errInsert)
			}

			categoryID, errInsert = InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
			if errInsert != nil {
				t.Fatalf("Failed to insert test category: %v", errInsert)
			}

			subcategoryID, errInsert = InsertTestSubcategory(db, "Test Subcategory", categoryID)
			if errInsert != nil {
				t.Fatalf("Failed to insert test line: %v", errInsert)
			}

			// Use different time periods for each contract to avoid overlap detection
			contractStart := startDate.AddDate(0, 0, idx*30)
			contractEnd := contractStart.AddDate(1, 0, 0)

			contract := domain.Contract{
				Model:         "Test Model " + string(rune('A'+idx)),
				ItemKey:       tt.itemKey,
				StartDate:     timePtr(contractStart),
				EndDate:       timePtr(contractEnd),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			}

			_, errCreate := contractStore.CreateContract(contract)

			if tt.expectError && errCreate == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && errCreate != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, errCreate)
			}
		})
	}
}

// TestCreateContractWithDuplicateItemKey tests duplicate product key detection
func TestCreateContractWithDuplicateItemKey(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	contractStore := NewContractStore(db)
	startDate := time.Now()

	tests := []struct {
		name        string
		contract    domain.Contract
		expectError bool
		description string
	}{
		{
			name: "duplicate - same product key for same client/affiliate",
			contract: domain.Contract{
				Model:     "Duplicate Contract",
				ItemKey:   "DUPLICATE-KEY-TEST",
				StartDate: timePtr(startDate),
				EndDate:   timePtr(startDate.AddDate(1, 0, 0)),
			},
			expectError: true,
			description: "Same product key for same client/affiliate should be rejected",
		},
		{
			name: "valid - same product key for different client",
			contract: domain.Contract{
				Model:     "Different Client Contract",
				ItemKey:   "DUPLICATE-KEY-TEST",
				StartDate: timePtr(startDate.AddDate(0, 0, 60)),
				EndDate:   timePtr(startDate.AddDate(1, 0, 60)),
			},
			expectError: false,
			description: "Same product key for different client should be allowed",
		},
		{
			name: "valid - same product key for same client but different affiliate",
			contract: domain.Contract{
				Model:     "Different Affiliate Contract",
				ItemKey:   "DUPLICATE-KEY-TEST",
				StartDate: timePtr(startDate.AddDate(0, 0, 120)),
				EndDate:   timePtr(startDate.AddDate(1, 0, 120)),
			},
			expectError: true,
			description: "Same product key with mismatched client/affiliate should be rejected",
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each test
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			// Setup dependencies for each test
			client1ID, errSetup := InsertTestClient(db, "Client 1", generateUniqueCNPJ())
			if errSetup != nil {
				t.Fatalf("Failed to insert client 1: %v", errSetup)
			}

			client2ID, errSetup := InsertTestClient(db, "Client 2", "11.222.333/0001-81")
			if errSetup != nil {
				t.Fatalf("Failed to insert client 2: %v", errSetup)
			}

			categoryID, errSetup := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
			if errSetup != nil {
				t.Fatalf("Failed to insert test category: %v", errSetup)
			}

			subcategoryID, errSetup := InsertTestSubcategory(db, "Test Subcategory", categoryID)
			if errSetup != nil {
				t.Fatalf("Failed to insert test line: %v", errSetup)
			}

			// Create affiliate for client 1
			affiliate1ID := uuid.New().String()
			_, errSetup = db.Exec("INSERT INTO affiliates (id, name, client_id) VALUES ($1, $2, $3)",
				affiliate1ID, "Affiliate 1", client1ID)
			if errSetup != nil {
				t.Fatalf("Failed to insert affiliate 1: %v", errSetup)
			}

			// Create affiliate for client 2
			affiliate2ID := uuid.New().String()
			_, errSetup = db.Exec("INSERT INTO affiliates (id, name, client_id) VALUES ($1, $2, $3)",
				affiliate2ID, "Affiliate 2", client2ID)
			if errSetup != nil {
				t.Fatalf("Failed to insert affiliate 2: %v", errSetup)
			}

			// For test 0 (duplicate), create first contract and set up the contract to test
			if idx == 0 {
				_, errSetup = contractStore.CreateContract(domain.Contract{
					Model:         "First Contract",
					ItemKey:       "DUPLICATE-KEY-TEST",
					StartDate:     tt.contract.StartDate,
					EndDate:       tt.contract.EndDate,
					SubcategoryID: subcategoryID,
					ClientID:      client1ID,
					AffiliateID:   &affiliate1ID,
				})
				if errSetup != nil {
					t.Fatalf("Failed to create first contract: %v", errSetup)
				}
			}

			// For test 1 (different client), use client2
			contract := tt.contract
			if idx == 1 {
				contract.SubcategoryID = subcategoryID
				contract.ClientID = client2ID
			} else {
				// For tests 0 and 2, use client1
				contract.SubcategoryID = subcategoryID
				contract.ClientID = client1ID
				if idx == 2 {
					// Test 2 uses a affiliate from client2 with client1 (should fail)
					contract.AffiliateID = &affiliate2ID
				} else {
					// Test 0 uses affiliate1
					contract.AffiliateID = &affiliate1ID
				}
			}

			_, err := contractStore.CreateContract(contract)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
		})
	}
}

// TestCreateContractWithInvalidDates tests various date validation scenarios
func TestCreateContractWithInvalidDates(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	contractStore := NewContractStore(db)
	now := time.Now()
	var clientID, categoryID, subcategoryID string

	tests := []struct {
		name        string
		startDate   time.Time
		endDate     time.Time
		expectError bool
		description string
	}{
		{
			name:        "invalid - end date before start date",
			startDate:   now,
			endDate:     now.AddDate(0, 0, -1),
			expectError: true,
			description: "End date before start date should be rejected",
		},
		{
			name:        "invalid - end date equals start date",
			startDate:   now,
			endDate:     now,
			expectError: true,
			description: "End date equal to start date should be rejected",
		},
		{
			name:        "valid - start date in the past",
			startDate:   now.AddDate(0, 0, -30),
			endDate:     now.AddDate(0, 0, 30),
			expectError: false,
			description: "Contract with start date in the past is allowed",
		},
		{
			name:        "valid - very short contract (1 day)",
			startDate:   now,
			endDate:     now.AddDate(0, 0, 1),
			expectError: false,
			description: "Contract with 1 day duration should be allowed",
		},
		{
			name:        "valid - very long contract (10 years)",
			startDate:   now,
			endDate:     now.AddDate(10, 0, 0),
			expectError: false,
			description: "Contract with 10 years duration should be allowed",
		},
		{
			name:        "valid - permanent contract (year 9999)",
			startDate:   now,
			endDate:     time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC),
			expectError: false,
			description: "Permanent contract with far future date should be allowed",
		},
		{
			name:        "invalid - negative duration (1 year backwards)",
			startDate:   now,
			endDate:     now.AddDate(-1, 0, 0),
			expectError: true,
			description: "Negative duration should be rejected",
		},
		{
			name:        "valid - start and end on same day but different times",
			startDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:     time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC),
			expectError: false,
			description: "Valid contract spanning more than a year",
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each test to avoid conflicts
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			// Re-insert dependencies after clearing tables
			var errInsert error
			clientID, errInsert = InsertTestClient(db, "Test Client", generateUniqueCNPJ())
			if errInsert != nil {
				t.Fatalf("Failed to insert test client: %v", errInsert)
			}

			categoryID, errInsert = InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
			if errInsert != nil {
				t.Fatalf("Failed to insert test category: %v", errInsert)
			}

			subcategoryID, errInsert = InsertTestSubcategory(db, "Test Subcategory", categoryID)
			if errInsert != nil {
				t.Fatalf("Failed to insert test line: %v", errInsert)
			}

			// Use different time periods for each contract to avoid overlap detection
			contractStart := tt.startDate.AddDate(0, 0, idx*60)
			contractEnd := tt.endDate.AddDate(0, 0, idx*60)

			contract := domain.Contract{
				Model:         "Test Contract " + string(rune('A'+idx)),
				ItemKey:       "KEY-" + string(rune('A'+idx)) + "-TEST",
				StartDate:     timePtr(contractStart),
				EndDate:       timePtr(contractEnd),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			}

			_, err := contractStore.CreateContract(contract)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
		})
	}
}

// TestCreateContractWithArchivedClient tests that contracts cannot be created for archived clients
func TestCreateContractWithArchivedClient(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	// Setup dependencies
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line: %v", err)
	}

	// Archive the client
	_, err = db.Exec("UPDATE clients SET archived_at = $1 WHERE id = $2", time.Now(), clientID)
	if err != nil {
		t.Fatalf("Failed to archive client: %v", err)
	}

	contractStore := NewContractStore(db)
	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)

	contract := domain.Contract{
		Model:         "Test Contract",
		ItemKey:       "ARCHIVED-CLIENT-KEY",
		StartDate:     timePtr(startDate),
		EndDate:       timePtr(endDate),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	_, err = contractStore.CreateContract(contract)
	if err == nil {
		t.Error("Expected error when creating contract for archived client, but got none")
	}
}

// TestUpdateContractWithInvalidData tests update operations with invalid data
func TestUpdateContractWithInvalidData(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer CloseDB(db)

	// Setup dependencies
	clientID, err := InsertTestClient(db, "Test Client", generateUniqueCNPJ())
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	categoryID, err := InsertTestCategory(db, "Software-"+uuid.New().String()[:8])
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryID, err := InsertTestSubcategory(db, "Test Subcategory", categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test line: %v", err)
	}

	contractStore := NewContractStore(db)
	startDate := time.Now()
	endDate := startDate.AddDate(1, 0, 0)

	// Create initial contract
	contractID, err := contractStore.CreateContract(domain.Contract{
		Model:         "Original Contract",
		ItemKey:       "ORIGINAL-KEY",
		StartDate:     timePtr(startDate),
		EndDate:       timePtr(endDate),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	})
	if err != nil {
		t.Fatalf("Failed to create initial contract: %v", err)
	}

	tests := []struct {
		name        string
		contract    domain.Contract
		expectError bool
		description string
	}{
		{
			name: "invalid - update with name too long",
			contract: domain.Contract{
				ID:            contractID,
				Model:         strings.Repeat("a", 256),
				ItemKey:       "ORIGINAL-KEY",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
			description: "Update with name too long should fail",
		},
		{
			name: "valid - update with empty product key keeps current",
			contract: domain.Contract{
				ID:            contractID,
				Model:         "Updated Contract",
				ItemKey:       "",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: false,
			description: "Update with empty product key should keep current value",
		},
		{
			name: "invalid - update with end date before start date",
			contract: domain.Contract{
				ID:            contractID,
				Model:         "Updated Contract",
				ItemKey:       "ORIGINAL-KEY",
				StartDate:     timePtr(endDate),
				EndDate:       timePtr(startDate),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: true,
			description: "Update with invalid dates should fail",
		},
		{
			name: "valid - update with valid data",
			contract: domain.Contract{
				ID:            contractID,
				Model:         "Updated Contract Name",
				ItemKey:       "ORIGINAL-KEY",
				StartDate:     timePtr(startDate),
				EndDate:       timePtr(endDate.AddDate(0, 1, 0)),
				SubcategoryID: subcategoryID,
				ClientID:      clientID,
			},
			expectError: false,
			description: "Update with valid data should succeed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := contractStore.UpdateContract(tt.contract)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.description, err)
			}
		})
	}
}

// ============================================================================
// NEW METHODS & ADDITIONAL OPERATIONS
// ============================================================================

// TestGetAllContracts tests the GetAllContracts method
func TestGetAllContracts(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Setup dependencies
	clientID, subClientID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	tests := []struct {
		name           string
		contractsToAdd int
		expectedCount  int
	}{
		{
			name:           "empty database",
			contractsToAdd: 0,
			expectedCount:  0,
		},
		{
			name:           "single contract",
			contractsToAdd: 1,
			expectedCount:  1,
		},
		{
			name:           "multiple contracts",
			contractsToAdd: 5,
			expectedCount:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear contracts for each test
			if _, err := db.Exec("DELETE FROM contracts"); err != nil {
				t.Fatalf("Failed to clear contracts: %v", err)
			}

			// Add contracts
			for i := 0; i < tt.contractsToAdd; i++ {
				contract := domain.Contract{
					Model:         "Test Contract",
					ItemKey:       "KEY-" + string(rune(48+i)),
					StartDate:     timePtr(time.Now().AddDate(0, 0, i*30)),
					EndDate:       timePtr(time.Now().AddDate(0, 0, (i+1)*30)),
					SubcategoryID: subcategoryID,
					ClientID:      clientID,
					AffiliateID:   &subClientID,
				}
				_, err := contractStore.CreateContract(contract)
				if err != nil {
					t.Fatalf("Failed to create contract: %v", err)
				}
			}

			// Test GetAllContracts
			contracts, err := contractStore.GetAllContracts()
			if err != nil {
				t.Errorf("GetAllContracts failed: %v", err)
			}

			if len(contracts) != tt.expectedCount {
				t.Errorf("Expected %d contracts, got %d", tt.expectedCount, len(contracts))
			}

			// Verify all contracts have required fields
			for _, contract := range contracts {
				if contract.ID == "" {
					t.Error("Contract ID is empty")
				}
				if contract.Model == "" {
					t.Error("Contract Model is empty")
				}
				if contract.ItemKey == "" {
					t.Error("Contract ItemKey is empty")
				}
			}
		})
	}
}

// TestGetContractsBySubcategoryID tests filtering contracts by line ID
func TestGetContractsBySubcategoryID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Setup dependencies
	clientID, subClientID, categoryID, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	// Create a second line in the same category
	subcategoryID2, err := InsertTestSubcategory(db, "Test Subcategory 2", categoryID)
	if err != nil {
		t.Fatalf("Failed to create second line: %v", err)
	}

	// Create contracts for both subcategories
	contract1 := domain.Contract{
		Model:         "Contract Subcategory 1",
		ItemKey:       "KEY-LINE1-001",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
		AffiliateID:   &subClientID,
	}
	_, err = contractStore.CreateContract(contract1)
	if err != nil {
		t.Fatalf("Failed to create contract 1: %v", err)
	}

	contract2 := domain.Contract{
		Model:         "Contract Subcategory 2",
		ItemKey:       "KEY-LINE2-001",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID2,
		ClientID:      clientID,
		AffiliateID:   &subClientID,
	}
	_, err = contractStore.CreateContract(contract2)
	if err != nil {
		t.Fatalf("Failed to create contract 2: %v", err)
	}

	tests := []struct {
		name               string
		querySubcategoryID string
		expectedCount      int
		shouldError        bool
	}{
		{
			name:               "valid line with contracts",
			querySubcategoryID: subcategoryID,
			expectedCount:      1,
			shouldError:        false,
		},
		{
			name:               "valid line with different contracts",
			querySubcategoryID: subcategoryID2,
			expectedCount:      1,
			shouldError:        false,
		},
		{
			name:               "empty line ID",
			querySubcategoryID: "",
			expectedCount:      0,
			shouldError:        true,
		},
		{
			name:               "non-existent line ID",
			querySubcategoryID: uuid.New().String(),
			expectedCount:      0,
			shouldError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contracts, err := contractStore.GetContractsBySubcategoryID(tt.querySubcategoryID)

			if (err != nil) != tt.shouldError {
				t.Errorf("Expected error: %v, got: %v", tt.shouldError, err)
			}

			if len(contracts) != tt.expectedCount {
				t.Errorf("Expected %d contracts, got %d", tt.expectedCount, len(contracts))
			}

			// Verify contracts belong to the queried line
			for _, contract := range contracts {
				if contract.SubcategoryID != tt.querySubcategoryID {
					t.Errorf("Contract SubcategoryID %s doesn't match query SubcategoryID %s", contract.SubcategoryID, tt.querySubcategoryID)
				}
			}
		})
	}
}

// TestGetContractsByCategoryID tests filtering contracts by category ID
func TestGetContractsByCategoryID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Setup dependencies
	clientID, subClientID, categoryID, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	// Create a second category with its own line
	categoryID2, err := InsertTestCategory(db, "Test Category 2")
	if err != nil {
		t.Fatalf("Failed to create second category: %v", err)
	}

	subcategoryID2, err := InsertTestSubcategory(db, "Test Subcategory in Category 2", categoryID2)
	if err != nil {
		t.Fatalf("Failed to create line in second category: %v", err)
	}

	// Create contracts in both categories
	contract1 := domain.Contract{
		Model:         "Contract Category 1",
		ItemKey:       "KEY-CAT1-001",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
		AffiliateID:   &subClientID,
	}
	_, err = contractStore.CreateContract(contract1)
	if err != nil {
		t.Fatalf("Failed to create contract in category 1: %v", err)
	}

	contract2 := domain.Contract{
		Model:         "Contract Category 2",
		ItemKey:       "KEY-CAT2-001",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID2,
		ClientID:      clientID,
		AffiliateID:   &subClientID,
	}
	_, err = contractStore.CreateContract(contract2)
	if err != nil {
		t.Fatalf("Failed to create contract in category 2: %v", err)
	}

	tests := []struct {
		name            string
		queryCategoryID string
		expectedCount   int
		shouldError     bool
	}{
		{
			name:            "valid category with contracts",
			queryCategoryID: categoryID,
			expectedCount:   1,
			shouldError:     false,
		},
		{
			name:            "valid category with different contracts",
			queryCategoryID: categoryID2,
			expectedCount:   1,
			shouldError:     false,
		},
		{
			name:            "empty category ID",
			queryCategoryID: "",
			expectedCount:   0,
			shouldError:     true,
		},
		{
			name:            "non-existent category ID",
			queryCategoryID: uuid.New().String(),
			expectedCount:   0,
			shouldError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contracts, err := contractStore.GetContractsByCategoryID(tt.queryCategoryID)

			if (err != nil) != tt.shouldError {
				t.Errorf("Expected error: %v, got: %v", tt.shouldError, err)
			}

			if len(contracts) != tt.expectedCount {
				t.Errorf("Expected %d contracts, got %d", tt.expectedCount, len(contracts))
			}

			// Verify contracts belong to subcategories in the queried category
			for _, contract := range contracts {
				var lineCategory string
				err := db.QueryRow("SELECT category_id FROM subcategories WHERE id = $1", contract.SubcategoryID).Scan(&lineCategory)
				if err != nil {
					t.Errorf("Failed to verify contract's line category: %v", err)
				}
				if lineCategory != tt.queryCategoryID {
					t.Errorf("Contract's line category %s doesn't match query category %s", lineCategory, tt.queryCategoryID)
				}
			}
		})
	}
}

// TestGetAllContractsWithMultipleClients tests GetAllContracts with contracts from different clients
func TestGetAllContractsWithMultipleClients(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Create two clients with their dependencies
	client1ID, affiliate1ID, _, line1ID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert dependencies for client 1: %v", err)
	}

	client2ID, err := InsertTestClient(db, "Test Client 2", "12.345.678/0001-99")
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}
	affiliate2ID, err := InsertTestAffiliate(db, "Affiliate 2", client2ID)
	if err != nil {
		t.Fatalf("Failed to create affiliate 2: %v", err)
	}
	category2ID, err := InsertTestCategory(db, "Category 2")
	if err != nil {
		t.Fatalf("Failed to create category 2: %v", err)
	}
	line2ID, err := InsertTestSubcategory(db, "Subcategory 2", category2ID)
	if err != nil {
		t.Fatalf("Failed to create line 2: %v", err)
	}

	// Create contracts for both clients
	for i := 0; i < 3; i++ {
		contract := domain.Contract{
			Model:         "Contract Client 1",
			ItemKey:       "KEY-C1-" + string(rune(48+i)),
			StartDate:     timePtr(time.Now().AddDate(0, 0, i*30)),
			EndDate:       timePtr(time.Now().AddDate(0, 0, (i+1)*30)),
			SubcategoryID: line1ID,
			ClientID:      client1ID,
			AffiliateID:   &affiliate1ID,
		}
		_, err := contractStore.CreateContract(contract)
		if err != nil {
			t.Fatalf("Failed to create contract for client 1: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		contract := domain.Contract{
			Model:         "Contract Client 2",
			ItemKey:       "KEY-C2-" + string(rune(48+i)),
			StartDate:     timePtr(time.Now().AddDate(0, 1, i*30)),
			EndDate:       timePtr(time.Now().AddDate(0, 1, (i+1)*30)),
			SubcategoryID: line2ID,
			ClientID:      client2ID,
			AffiliateID:   &affiliate2ID,
		}
		_, err := contractStore.CreateContract(contract)
		if err != nil {
			t.Fatalf("Failed to create contract for client 2: %v", err)
		}
	}

	// Test GetAllContracts returns all contracts
	contracts, err := contractStore.GetAllContracts()
	if err != nil {
		t.Errorf("GetAllContracts failed: %v", err)
	}

	expectedTotal := 5
	if len(contracts) != expectedTotal {
		t.Errorf("Expected %d total contracts, got %d", expectedTotal, len(contracts))
	}

	// Verify we have contracts from both clients
	client1Count := 0
	client2Count := 0
	for _, contract := range contracts {
		switch contract.ClientID {
		case client1ID:
			client1Count++
		case client2ID:
			client2Count++
		}
	}

	if client1Count != 3 {
		t.Errorf("Expected 3 contracts for client 1, got %d", client1Count)
	}
	if client2Count != 2 {
		t.Errorf("Expected 2 contracts for client 2, got %d", client2Count)
	}
}

// TestGetContractsBySubcategoryIDWithMultipleClients tests that GetContractsBySubcategoryID returns contracts from all clients
func TestGetContractsBySubcategoryIDWithMultipleClients(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	contractStore := NewContractStore(db)

	// Create two clients
	client1ID, affiliate1ID, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert dependencies for client 1: %v", err)
	}

	client2ID, err := InsertTestClient(db, "Test Client 2", "98.765.432/0001-11")
	if err != nil {
		t.Fatalf("Failed to create second client: %v", err)
	}
	affiliate2ID, err := InsertTestAffiliate(db, "Affiliate 2", client2ID)
	if err != nil {
		t.Fatalf("Failed to create affiliate 2: %v", err)
	}

	// Create contracts for the same line but different clients
	contract1 := domain.Contract{
		Model:         "Contract Client 1",
		ItemKey:       "KEY-SAME-001",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      client1ID,
		AffiliateID:   &affiliate1ID,
	}
	_, err = contractStore.CreateContract(contract1)
	if err != nil {
		t.Fatalf("Failed to create contract for client 1: %v", err)
	}

	contract2 := domain.Contract{
		Model:         "Contract Client 2",
		ItemKey:       "KEY-SAME-002",
		StartDate:     timePtr(time.Now()),
		EndDate:       timePtr(time.Now().AddDate(1, 0, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      client2ID,
		AffiliateID:   &affiliate2ID,
	}
	_, err = contractStore.CreateContract(contract2)
	if err != nil {
		t.Fatalf("Failed to create contract for client 2: %v", err)
	}

	// Test GetContractsBySubcategoryID returns contracts from both clients
	contracts, err := contractStore.GetContractsBySubcategoryID(subcategoryID)
	if err != nil {
		t.Errorf("GetContractsBySubcategoryID failed: %v", err)
	}

	if len(contracts) != 2 {
		t.Errorf("Expected 2 contracts for line, got %d", len(contracts))
	}

	// Verify we have contracts from both clients
	client1Found := false
	client2Found := false
	for _, contract := range contracts {
		if contract.ClientID == client1ID {
			client1Found = true
		}
		if contract.ClientID == client2ID {
			client2Found = true
		}
	}

	if !client1Found {
		t.Error("Contract from client 1 not found")
	}
	if !client2Found {
		t.Error("Contract from client 2 not found")
	}
}

func TestGetContractsNotStarted(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert dependencies: %v", err)
	}

	contractStore := NewContractStore(db)

	// Create future contract
	futureStart := timePtr(time.Now().AddDate(0, 1, 0))
	contract := domain.Contract{
		Model:         "Future Contract",
		ItemKey:       "FUTURE-1",
		StartDate:     futureStart,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}
	_, err = contractStore.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Create started contract
	storeContract := domain.Contract{
		Model:         "Started Contract",
		ItemKey:       "STARTED-1",
		StartDate:     timePtr(time.Now().AddDate(0, -1, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}
	_, err = contractStore.CreateContract(storeContract)
	if err != nil {
		t.Fatalf("Failed to create contract 2: %v", err)
	}

	contracts, err := contractStore.GetContractsNotStarted()
	if err != nil {
		t.Fatalf("Failed to get contracts not started: %v", err)
	}
	if len(contracts) != 1 {
		t.Errorf("Expected 1 contract not started, got %d", len(contracts))
	}
}

func TestGetContractStatsByClientID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert dependencies: %v", err)
	}
	contractStore := NewContractStore(db)

	// Active
	_, err = contractStore.CreateContract(domain.Contract{
		Model:         "Active",
		ItemKey:       "ACTIVE-1",
		StartDate:     timePtr(time.Now().AddDate(0, -1, 0)),
		EndDate:       timePtr(time.Now().AddDate(0, 1, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	})
	if err != nil {
		t.Fatalf("Failed to create active contract: %v", err)
	}

	// Expired
	_, err = contractStore.CreateContract(domain.Contract{
		Model:         "Expired",
		ItemKey:       "EXPIRED-1",
		StartDate:     timePtr(time.Now().AddDate(0, -3, 0)),
		EndDate:       timePtr(time.Now().AddDate(0, -2, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	})
	if err != nil {
		t.Fatalf("Failed to create expired contract: %v", err)
	}

	stats, err := contractStore.GetContractStatsByClientID(clientID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.ActiveContracts != 1 {
		t.Errorf("Expected 1 active contract, got %d", stats.ActiveContracts)
	}
	if stats.ExpiredContracts != 1 {
		t.Errorf("Expected 1 expired contract, got %d", stats.ExpiredContracts)
	}
}

func TestGetContractStatsForAllClients(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert dependencies: %v", err)
	}
	contractStore := NewContractStore(db)

	// Active
	contractStore.CreateContract(domain.Contract{
		Model:         "Active Global",
		ItemKey:       "ACTIVE-GLOBAL-1",
		StartDate:     timePtr(time.Now().AddDate(0, -1, 0)),
		EndDate:       timePtr(time.Now().AddDate(0, 1, 0)),
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	})

	statsList, err := contractStore.GetContractStatsForAllClients()
	if err != nil {
		t.Fatalf("Failed to get global stats: %v", err)
	}
	if len(statsList) == 0 {
		t.Error("Expected stats, got empty list")
	}

	found := false
	for _, s := range statsList {
		if s.ClientID == clientID {
			found = true
			if s.ActiveContracts != 1 {
				t.Errorf("Expected 1 active contract in global stats, got %d", s.ActiveContracts)
			}
		}
	}
	if !found {
		t.Error("Client stats not found in global list")
	}
}

func TestArchiveContract(t *testing.T) {
	db := setupContractTestDB(t)
	defer db.Close()

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	store := NewContractStore(db)

	// Create a contract to archive
	now := time.Now()
	endDate := now.AddDate(1, 0, 0)
	contract := domain.Contract{
		Model:         "Archive Test Contract",
		ItemKey:       "archive-test-key-" + uuid.New().String()[:8],
		StartDate:     &now,
		EndDate:       &endDate,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Verify contract is not archived initially
	created, err := store.GetContractByID(id)
	if err != nil {
		t.Fatalf("Failed to get contract: %v", err)
	}
	if created.ArchivedAt != nil {
		t.Error("Contract should not be archived initially")
	}

	// Archive the contract
	err = store.ArchiveContract(id)
	if err != nil {
		t.Fatalf("Failed to archive contract: %v", err)
	}

	// Verify contract is excluded from GetAllContracts (non-archived only)
	allContracts, err := store.GetAllContracts()
	if err != nil {
		t.Fatalf("Failed to get all contracts: %v", err)
	}
	for _, c := range allContracts {
		if c.ID == id {
			t.Error("Archived contract should NOT appear in GetAllContracts")
		}
	}

	// Verify contract IS included in GetAllContractsIncludingArchived
	allIncluding, err := store.GetAllContractsIncludingArchived()
	if err != nil {
		t.Fatalf("Failed to get all contracts including archived: %v", err)
	}
	found := false
	for _, c := range allIncluding {
		if c.ID == id {
			found = true
			if c.ArchivedAt == nil {
				t.Error("Archived contract should have ArchivedAt set")
			}
		}
	}
	if !found {
		t.Error("Archived contract should appear in GetAllContractsIncludingArchived")
	}

	// Archiving an already archived contract should fail
	err = store.ArchiveContract(id)
	if err == nil {
		t.Error("Expected error when archiving an already archived contract")
	}
	if err != nil && !strings.Contains(err.Error(), "already archived") {
		t.Errorf("Expected 'already archived' error, got: %v", err)
	}

	// Archiving a non-existent contract should fail
	err = store.ArchiveContract(uuid.New().String())
	if err == nil {
		t.Error("Expected error when archiving non-existent contract")
	}

	// Archiving with empty ID should fail
	err = store.ArchiveContract("")
	if err == nil {
		t.Error("Expected error when archiving with empty ID")
	}
}

func TestUnarchiveContract(t *testing.T) {
	db := setupContractTestDB(t)
	defer db.Close()

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	store := NewContractStore(db)

	// Create and archive a contract
	now := time.Now()
	endDate := now.AddDate(1, 0, 0)
	contract := domain.Contract{
		Model:         "Unarchive Test Contract",
		ItemKey:       "unarchive-test-key-" + uuid.New().String()[:8],
		StartDate:     &now,
		EndDate:       &endDate,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	err = store.ArchiveContract(id)
	if err != nil {
		t.Fatalf("Failed to archive contract: %v", err)
	}

	// Verify it is archived
	allContracts, err := store.GetAllContracts()
	if err != nil {
		t.Fatalf("Failed to get all contracts: %v", err)
	}
	for _, c := range allContracts {
		if c.ID == id {
			t.Error("Archived contract should NOT appear in GetAllContracts before unarchive")
		}
	}

	// Unarchive the contract
	err = store.UnarchiveContract(id)
	if err != nil {
		t.Fatalf("Failed to unarchive contract: %v", err)
	}

	// Verify contract reappears in GetAllContracts
	allContracts, err = store.GetAllContracts()
	if err != nil {
		t.Fatalf("Failed to get all contracts after unarchive: %v", err)
	}
	found := false
	for _, c := range allContracts {
		if c.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("Unarchived contract should appear in GetAllContracts")
	}

	// Unarchiving a non-archived contract should fail
	err = store.UnarchiveContract(id)
	if err == nil {
		t.Error("Expected error when unarchiving a contract that is not archived")
	}
	if err != nil && !strings.Contains(err.Error(), "not archived") {
		t.Errorf("Expected 'not archived' error, got: %v", err)
	}

	// Unarchiving a non-existent contract should fail
	err = store.UnarchiveContract(uuid.New().String())
	if err == nil {
		t.Error("Expected error when unarchiving non-existent contract")
	}

	// Unarchiving with empty ID should fail
	err = store.UnarchiveContract("")
	if err == nil {
		t.Error("Expected error when unarchiving with empty ID")
	}
}

func TestArchiveUnarchiveContractRoundTrip(t *testing.T) {
	db := setupContractTestDB(t)
	defer db.Close()

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	store := NewContractStore(db)

	now := time.Now()
	endDate := now.AddDate(1, 0, 0)
	contract := domain.Contract{
		Model:         "Roundtrip Test Contract",
		ItemKey:       "roundtrip-test-key-" + uuid.New().String()[:8],
		StartDate:     &now,
		EndDate:       &endDate,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Archive → Unarchive → Archive → Unarchive cycle
	for i := 0; i < 3; i++ {
		err = store.ArchiveContract(id)
		if err != nil {
			t.Fatalf("Round %d: Failed to archive: %v", i, err)
		}

		// Confirm excluded from active list
		active, _ := store.GetAllContracts()
		for _, c := range active {
			if c.ID == id {
				t.Fatalf("Round %d: Archived contract should not be in active list", i)
			}
		}

		err = store.UnarchiveContract(id)
		if err != nil {
			t.Fatalf("Round %d: Failed to unarchive: %v", i, err)
		}

		// Confirm back in active list
		active, _ = store.GetAllContracts()
		found := false
		for _, c := range active {
			if c.ID == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("Round %d: Unarchived contract should be in active list", i)
		}
	}
}

func TestArchiveContractAffectsStats(t *testing.T) {
	db := setupContractTestDB(t)
	defer db.Close()

	clientID, _, _, subcategoryID, err := insertTestDependencies(db)
	if err != nil {
		t.Fatalf("Failed to insert test dependencies: %v", err)
	}

	store := NewContractStore(db)

	now := time.Now()
	endDate := now.AddDate(1, 0, 0)
	contract := domain.Contract{
		Model:         "Stats Test Contract",
		ItemKey:       "stats-test-key-" + uuid.New().String()[:8],
		StartDate:     &now,
		EndDate:       &endDate,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	}

	id, err := store.CreateContract(contract)
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Check stats before archiving
	stats, err := store.GetContractStatsByClientID(clientID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.ActiveContracts != 1 {
		t.Errorf("Expected 1 active contract before archive, got %d", stats.ActiveContracts)
	}
	if stats.ArchivedContracts != 0 {
		t.Errorf("Expected 0 archived contracts before archive, got %d", stats.ArchivedContracts)
	}

	// Archive
	err = store.ArchiveContract(id)
	if err != nil {
		t.Fatalf("Failed to archive contract: %v", err)
	}

	// Check stats after archiving
	stats, err = store.GetContractStatsByClientID(clientID)
	if err != nil {
		t.Fatalf("Failed to get stats after archive: %v", err)
	}
	if stats.ActiveContracts != 0 {
		t.Errorf("Expected 0 active contracts after archive, got %d", stats.ActiveContracts)
	}
	if stats.ArchivedContracts != 1 {
		t.Errorf("Expected 1 archived contract after archive, got %d", stats.ArchivedContracts)
	}

	// Unarchive
	err = store.UnarchiveContract(id)
	if err != nil {
		t.Fatalf("Failed to unarchive contract: %v", err)
	}

	// Check stats after unarchiving
	stats, err = store.GetContractStatsByClientID(clientID)
	if err != nil {
		t.Fatalf("Failed to get stats after unarchive: %v", err)
	}
	if stats.ActiveContracts != 1 {
		t.Errorf("Expected 1 active contract after unarchive, got %d", stats.ActiveContracts)
	}
	if stats.ArchivedContracts != 0 {
		t.Errorf("Expected 0 archived contracts after unarchive, got %d", stats.ArchivedContracts)
	}
}
