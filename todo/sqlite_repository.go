package todo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set journal mode: %w", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS lists (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS todos (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			list_id     INTEGER NOT NULL REFERENCES lists(id),
			title       TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			completed   BOOLEAN NOT NULL DEFAULT 0,
			priority    TEXT NOT NULL DEFAULT 'medium' CHECK(priority IN ('low','medium','high')),
			created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,
		`INSERT OR IGNORE INTO lists (id, name) VALUES (1, 'My Todos')`,
	}
	for _, stmt := range stmts {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) CreateList(name string) (*List, error) {
	result, err := r.db.Exec("INSERT INTO lists (name) VALUES (?)", name)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &List{ID: id, Name: name, CreatedAt: time.Now()}, nil
}

func (r *SQLiteRepository) GetDefaultList() (*List, error) {
	var l List
	err := r.db.QueryRow("SELECT id, name, created_at FROM lists WHERE id = 1").
		Scan(&l.ID, &l.Name, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *SQLiteRepository) Create(listID int64, title, description string, priority Priority) (*Todo, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO todos (list_id, title, description, priority, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		listID, title, description, string(priority), now, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Todo{
		ID:          id,
		ListID:      listID,
		Title:       title,
		Description: description,
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *SQLiteRepository) List(filter Filter) ([]Todo, error) {
	query := "SELECT id, list_id, title, description, completed, priority, created_at, updated_at FROM todos WHERE list_id = ?"
	args := []any{filter.ListID}

	if filter.Status != nil {
		query += " AND completed = ?"
		if *filter.Status {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	if filter.Priority != nil {
		query += " AND priority = ?"
		args = append(args, string(*filter.Priority))
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		var priority string
		if err := rows.Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Completed, &priority, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Priority = Priority(strings.ToLower(priority))
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

func (r *SQLiteRepository) ToggleCompleted(id int64) (*Todo, error) {
	_, err := r.db.Exec("UPDATE todos SET completed = NOT completed, updated_at = datetime('now') WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	var t Todo
	var priority string
	err = r.db.QueryRow(
		"SELECT id, list_id, title, description, completed, priority, created_at, updated_at FROM todos WHERE id = ?", id,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Completed, &priority, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Priority = Priority(strings.ToLower(priority))
	return &t, nil
}

func (r *SQLiteRepository) Delete(id int64) error {
	result, err := r.db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("todo %d not found", id)
	}
	return nil
}
