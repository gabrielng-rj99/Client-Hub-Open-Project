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

package categorystore

import (
	"testing"
	"time"

	"Open-Generic-Hub/backend/domain"
)

func TestCreateCategory(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	tests := []struct {
		name        string
		category    domain.Category
		expectError bool
	}{
		{
			name: "sucesso - criação normal",
			category: domain.Category{
				Name: "Test Category",
			},
			expectError: false,
		},
		{
			name: "erro - nome vazio",
			category: domain.Category{
				Name: "",
			},
			expectError: true,
		},
		{
			name: "erro - nome duplicado",
			category: domain.Category{
				Name: "Test Category",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test to avoid leftover data
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			if tt.name == "erro - nome duplicado" {
				_, err := InsertTestCategory(db, "Test Category")
				if err != nil {
					t.Fatalf("Failed to insert category for duplicate test: %v", err)
				}
			}

			categoryStore := NewCategoryStore(db)
			id, err := categoryStore.CreateCategory(tt.category)

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

func TestGetCategoryByID(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	// Clean up before test
	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	// Insert test category
	categoryID, err := InsertTestCategory(db, "Test Category")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		expectError bool
		expectFound bool
	}{
		{
			name:        "sucesso - categoria encontrada",
			id:          categoryID,
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
			id:          "550e8400-e29b-41d4-a716-446655440000",
			expectError: false,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryStore := NewCategoryStore(db)
			category, err := categoryStore.GetCategoryByID(tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectFound && category == nil {
				t.Error("Expected category but got nil")
			}
			if !tt.expectFound && category != nil {
				t.Error("Expected no category but got one")
			}
		})
	}
}

func TestDeleteCategoryWithLinesAssociated(t *testing.T) {
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

	categoryID, err := InsertTestCategory(db, "Categoria Com Linha")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryStore := NewSubcategoryStore(db)
	line := domain.Subcategory{
		Name:       "Linha Teste",
		CategoryID: categoryID,
	}
	lineID, err := subcategoryStore.CreateSubcategory(line)
	if err != nil {
		t.Fatalf("Failed to create line: %v", err)
	}

	// Insert test client for contract
	clientID, err := InsertTestClient(db, "Test Client Protection", "12.345.678/0001-90")
	if err != nil {
		t.Fatalf("Failed to insert test client: %v", err)
	}

	// Add an contract to check protection
	contractStore := NewContractStore(db)
	_, err = contractStore.CreateContract(domain.Contract{
		Model:         "Protection Test",
		SubcategoryID: lineID,
		ClientID:      clientID,
	})
	// Need to fix code above to capture ID.

}

func TestLineNameUniquePerCategory(t *testing.T) {
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

	categoryID, err := InsertTestCategory(db, "Categoria Unica")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	subcategoryStore := NewSubcategoryStore(db)
	line := domain.Subcategory{
		Name:       "Linha Unica",
		CategoryID: categoryID,
	}
	_, err = subcategoryStore.CreateSubcategory(line)
	if err != nil {
		t.Fatalf("Failed to create first line: %v", err)
	}

	// Tentar criar segunda linha com mesmo nome na mesma categoria
	_, err = subcategoryStore.CreateSubcategory(line)
	if err == nil {
		t.Error("Expected error when creating duplicate line name in same category, got none")
	}
}

func TestGetAllCategories(t *testing.T) {
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

	// Insert test categories
	testCategories := []string{"Category 1", "Category 2", "Category 3"}
	for _, name := range testCategories {
		_, err := InsertTestCategory(db, name)
		if err != nil {
			t.Fatalf("Failed to insert test category %q: %v", name, err)
		}
	}

	categoryStore := NewCategoryStore(db)
	categories, err := categoryStore.GetAllCategories()

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if len(categories) != len(testCategories) {
		t.Errorf("Expected %d categories but got %d", len(testCategories), len(categories))
	}
	// Clean up after test
	if err := ClearTables(db); err != nil {
		t.Errorf("Failed to clear tables after test: %v", err)
	}
}

func TestUpdateCategory(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	// Clean up before test
	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	// Insert test category
	categoryID, err := InsertTestCategory(db, "Test Category")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	tests := []struct {
		name        string
		category    domain.Category
		expectError bool
	}{
		{
			name: "sucesso - atualização normal",
			category: domain.Category{
				ID:   categoryID,
				Name: "Updated Category",
			},
			expectError: false,
		},
		{
			name: "erro - id vazio",
			category: domain.Category{
				Name: "Updated Category",
			},
			expectError: true,
		},
		{
			name: "erro - nome vazio",
			category: domain.Category{
				ID: categoryID,
			},
			expectError: true,
		},
		{
			name: "erro - categoria não existe",
			category: domain.Category{
				ID:   "non-existent-id",
				Name: "Updated Category",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each subtest to avoid data conflicts
			if err := ClearTables(db); err != nil {
				t.Fatalf("Failed to clear tables: %v", err)
			}

			// For each subtest, insert fresh category if needed
			var testCategoryID string
			switch tt.category.ID {
			case categoryID:
				id, err := InsertTestCategory(db, "Test Category")
				if err != nil {
					t.Fatalf("Failed to insert test category: %v", err)
				}
				testCategoryID = id
				tt.category.ID = testCategoryID
			case "non-existent-id":
				tt.category.ID = "550e8400-e29b-41d4-a716-446655440000"
			}

			categoryStore := NewCategoryStore(db)
			err := categoryStore.UpdateCategory(tt.category)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify update
				var name string
				err = db.QueryRow("SELECT name FROM categories WHERE id = $1", tt.category.ID).Scan(&name)
				if err != nil {
					t.Errorf("Failed to query updated category: %v", err)
				}
				if name != tt.category.Name {
					t.Errorf("Expected name %q but got %q", tt.category.Name, name)
				}
			}
		})
	}
}

func TestDeleteCategory(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer func() {
		if err := CloseDB(db); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	}()

	// Clean up before test
	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	// Insert test category
	categoryID, err := InsertTestCategory(db, "Test Category")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	tests := []struct {
		name        string
		id          string
		expectError bool
	}{
		{
			name:        "sucesso - deleção normal",
			id:          categoryID,
			expectError: false,
		},
		{
			name:        "erro - id vazio",
			id:          "",
			expectError: true,
		},
		{
			name:        "erro - categoria não existe",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryStore := NewCategoryStore(db)
			err := categoryStore.DeleteCategory(tt.id)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Verify deletion
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", tt.id).Scan(&count)
				if err != nil {
					t.Fatalf("Failed to query deleted category: %v", err)
				}
				if count != 0 {
					t.Errorf("Expected category to be deleted but found %d", count)
				}
			}
		})
	}
}

func TestArchiveCategory(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	store := NewCategoryStore(db)

	// Create category
	cat := domain.Category{Name: "To Archive"}
	id, err := store.CreateCategory(cat)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Archive
	err = store.ArchiveCategory(id)
	if err != nil {
		t.Fatalf("Failed to archive category: %v", err)
	}

	// Verify archived (Check ArchivedAt != nil)
	archivedCat, err := store.GetCategoryByID(id)
	if err != nil {
		t.Fatalf("Failed to get archived category: %v", err)
	}
	if archivedCat.ArchivedAt == nil {
		t.Error("Category should be archived (ArchivedAt should not be nil)")
	}

	// Unarchive
	err = store.UnarchiveCategory(id)
	if err != nil {
		t.Fatalf("Failed to unarchive category: %v", err)
	}

	// Verify unarchived
	unarchivedCat, err := store.GetCategoryByID(id)
	if err != nil {
		t.Fatalf("Failed to get unarchived category: %v", err)
	}
	if unarchivedCat.ArchivedAt != nil {
		t.Error("Category should not be archived")
	}
}

func TestUpdateCategoryStatus(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	categoryStore := NewCategoryStore(db)

	// Create category
	categoryID, err := InsertTestCategory(db, "Status Test Category")
	if err != nil {
		t.Fatalf("Failed to insert category: %v", err)
	}

	// Update status (should be inactive as no contracts)
	err = categoryStore.UpdateCategoryStatus(categoryID)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM categories WHERE id = $1", categoryID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query category status: %v", err)
	}
	if status != "inativo" {
		t.Errorf("Expected status 'inativo' for category with no contracts, got '%s'", status)
	}

	// Add an active contract
	subcategoryID, _ := InsertTestSubcategory(db, "Line 1", categoryID)
	clientID, _ := InsertTestClient(db, "Client 1", "55.723.174/0001-10")
	contractStore := NewContractStore(db)
	startDate := timePtr(time.Now().AddDate(0, 0, -10))
	endDate := timePtr(time.Now().AddDate(0, 0, 10))
	_, err = contractStore.CreateContract(domain.Contract{
		Model:         "Contract 1",
		ItemKey:       "KEY-1",
		StartDate:     startDate,
		EndDate:       endDate,
		SubcategoryID: subcategoryID,
		ClientID:      clientID,
	})
	if err != nil {
		t.Fatalf("Failed to create contract: %v", err)
	}

	// Update status (should be active)
	err = categoryStore.UpdateCategoryStatus(categoryID)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	err = db.QueryRow("SELECT status FROM categories WHERE id = $1", categoryID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query category status: %v", err)
	}
	if status != "ativo" {
		t.Errorf("Expected status 'ativo' for category with active contracts, got '%s'", status)
	}
}

func TestSearchCategories(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer CloseDB(db)

	if err := ClearTables(db); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	store := NewCategoryStore(db)

	// Create categories
	cat1 := domain.Category{Name: "SearchAlpha"}
	cat2 := domain.Category{Name: "SearchBeta"}
	store.CreateCategory(cat1)
	store.CreateCategory(cat2)

	// Test Exact Match
	cats, err := store.SearchCategories("SearchAlpha", true, 0, 0)
	if err != nil {
		t.Fatalf("Failed to search categories: %v", err)
	}
	if len(cats) == 0 {
		t.Error("Expected to find category")
	}
	if cats[0].Name != "SearchAlpha" {
		t.Errorf("Expected SearchAlpha, got %s", cats[0].Name)
	}
}
