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
	"Open-Generic-Hub/backend/domain"
	"Open-Generic-Hub/backend/repository"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CategoryStore é a nossa "caixa de ferramentas" para operações com a tabela categories.
type CategoryStore struct {
	db repository.DBInterface
}

// NewCategoryStore cria uma nova instância de CategoryStore.
func NewCategoryStore(db repository.DBInterface) *CategoryStore {
	return &CategoryStore{
		db: db,
	}
}

// CreateCategory insere uma nova categoria no banco de dados.
func (s *CategoryStore) CreateCategory(category domain.Category) (string, error) {
	trimmedName, err := repository.ValidateName(category.Name, 255)
	if err != nil {
		return "", err
	}

	newID := uuid.New().String()
	sqlStatement := `INSERT INTO categories (id, name) VALUES ($1, $2)`

	_, err = s.db.Exec(sqlStatement, newID, trimmedName)
	if err != nil {
		return "", err
	}

	return newID, nil
}

// GetAllCategories busca TODAS as categorias do banco de dados.
// Isto será útil para preencher listas de seleção no frontend.
func (s *CategoryStore) GetAllCategories() (categories []domain.Category, err error) {
	sqlStatement := `SELECT id, name, status, archived_at FROM categories WHERE archived_at IS NULL`
	rows, err := s.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []domain.Category{}, nil
	}

	defer func() {
		if rows != nil {
			closeErr := rows.Close()
			if err == nil {
				err = closeErr
			}
		}
	}()

	categories = []domain.Category{}
	for rows.Next() {
		var category domain.Category
		if err = rows.Scan(&category.ID, &category.Name, &category.Status, &category.ArchivedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetAllCategoriesIncludingArchived busca todas as categorias, incluindo arquivadas
func (s *CategoryStore) GetAllCategoriesIncludingArchived() (categories []domain.Category, err error) {
	sqlStatement := `SELECT id, name, status, archived_at FROM categories`
	rows, err := s.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []domain.Category{}, nil
	}

	defer func() {
		if rows != nil {
			closeErr := rows.Close()
			if err == nil {
				err = closeErr
			}
		}
	}()

	categories = []domain.Category{}
	for rows.Next() {
		var category domain.Category
		if err = rows.Scan(&category.ID, &category.Name, &category.Status, &category.ArchivedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// SearchCategories busca categorias por nome, com suporte a limites e filtro de inativos
func (s *CategoryStore) SearchCategories(name string, includeArchived bool, limit int, offset int) ([]domain.Category, error) {
	name = strings.TrimSpace(name)
	var sqlStatement string
	var args []interface{}
	argsCount := 1

	sqlStatement = `SELECT id, name, status, archived_at FROM categories WHERE 1=1`

	if name != "" {
		sqlStatement += fmt.Sprintf(` AND name ILIKE $%d`, argsCount)
		args = append(args, "%"+name+"%")
		argsCount++
	}

	if !includeArchived {
		sqlStatement += ` AND archived_at IS NULL`
	}
	sqlStatement += ` ORDER BY name ASC`

	if limit > 0 {
		sqlStatement += fmt.Sprintf(" LIMIT $%d", argsCount)
		args = append(args, limit)
		argsCount++
	}

	if offset > 0 {
		sqlStatement += fmt.Sprintf(" OFFSET $%d", argsCount)
		args = append(args, offset)
		argsCount++
	}

	rows, err := s.db.Query(sqlStatement, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			closeErr := rows.Close()
			if err == nil {
				err = closeErr
			}
		}
	}()

	var categories []domain.Category
	for rows.Next() {
		var category domain.Category
		if err = rows.Scan(&category.ID, &category.Name, &category.Status, &category.ArchivedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// GetCategoryByID busca uma categoria específica pelo seu ID
func (s *CategoryStore) GetCategoryByID(id string) (*domain.Category, error) {
	if id == "" {
		return nil, errors.New("category ID cannot be empty")
	}
	sqlStatement := `SELECT id, name, status, archived_at FROM categories WHERE id = $1`

	var category domain.Category
	err := s.db.QueryRow(sqlStatement, id).Scan(
		&category.ID,
		&category.Name,
		&category.Status,
		&category.ArchivedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (s *CategoryStore) UpdateCategory(category domain.Category) error {
	if category.ID == "" {
		return errors.New("category ID cannot be empty")
	}
	trimmedName, err := repository.ValidateName(category.Name, 255)
	if err != nil {
		return err
	}
	// Check if category exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", category.ID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("category does not exist")
	}
	sqlStatement := `UPDATE categories SET name = $1 WHERE id = $2`
	result, err := s.db.Exec(sqlStatement, trimmedName, category.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no category updated")
	}
	return nil
}

func (s *CategoryStore) DeleteCategory(id string) error {
	if id == "" {
		return errors.New("category ID cannot be empty")
	}
	// Check if category exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("category does not exist")
	}

	// Check if any subcategories are used in active contracts
	var contractsCount int
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT c.id)
		FROM contracts c
		INNER JOIN subcategories l ON c.subcategory_id = l.id
		WHERE l.category_id = $1
	`, id).Scan(&contractsCount)
	if err != nil {
		return err
	}
	if contractsCount > 0 {
		return errors.New("cannot delete category: it has subcategories associated with active contracts")
	}

	// Delete all subcategories associated with this category first
	_, err = s.db.Exec("DELETE FROM subcategories WHERE category_id = $1", id)
	if err != nil {
		return err
	}

	// Now delete the category
	sqlStatement := `DELETE FROM categories WHERE id = $1`
	result, err := s.db.Exec(sqlStatement, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no category deleted")
	}
	return nil
}

// ArchiveCategory arquiva uma categoria
func (s *CategoryStore) ArchiveCategory(id string) error {
	if id == "" {
		return errors.New("category ID cannot be empty")
	}

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("category does not exist")
	}

	sqlStatement := `UPDATE categories SET archived_at = $1 WHERE id = $2`
	result, err := s.db.Exec(sqlStatement, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no category archived")
	}

	return nil
}

// UnarchiveCategory desarquiva uma categoria
func (s *CategoryStore) UnarchiveCategory(id string) error {
	if id == "" {
		return errors.New("category ID cannot be empty")
	}

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM categories WHERE id = $1", id).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("category does not exist")
	}

	sqlStatement := `UPDATE categories SET archived_at = NULL WHERE id = $1`
	result, err := s.db.Exec(sqlStatement, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no category unarchived")
	}

	return nil
}

// UpdateCategoryStatus atualiza o status da categoria baseado no uso em contratos
// Ignora categorias arquivadas
func (s *CategoryStore) UpdateCategoryStatus(categoryID string) error {
	if categoryID == "" {
		return errors.New("category ID cannot be empty")
	}

	// Check if category is archived - don't update status for archived categories
	var archivedAt *time.Time
	err := s.db.QueryRow(`SELECT archived_at FROM categories WHERE id = $1`, categoryID).Scan(&archivedAt)
	if err != nil {
		return err
	}
	if archivedAt != nil {
		return nil // Don't update status for archived categories
	}

	// Check if category has any subcategories associated with active contracts
	// Active = not archived and (no end_date or end_date in future)
	var activeContracts int
	now := time.Now()
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT c.id)
		FROM contracts c
		INNER JOIN subcategories l ON c.subcategory_id = l.id
		WHERE l.category_id = $1
		AND c.archived_at IS NULL
		AND (c.end_date IS NULL OR c.end_date > $2)
	`, categoryID, now).Scan(&activeContracts)
	if err != nil {
		return err
	}

	// Set status based on usage in active contracts
	var newStatus string
	if activeContracts > 0 {
		newStatus = "ativo"
	} else {
		newStatus = "inativo"
	}

	_, err = s.db.Exec(`UPDATE categories SET status = $1 WHERE id = $2`, newStatus, categoryID)
	if err != nil {
		return err
	}

	return nil
}
