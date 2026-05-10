package internal

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(dsn string) (*TaskRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &TaskRepository{db: db}, nil
}

func (r *TaskRepository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			title       TEXT NOT NULL,
			description TEXT,
			done        BOOLEAN DEFAULT false,
			created_at  TIMESTAMP DEFAULT now()
		)
	`)
	return err
}

func (r *TaskRepository) GetAll(ctx context.Context) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, done FROM tasks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// SearchUnsafe — УЯЗВИМЫЙ поиск (только для демонстрации!)
func (r *TaskRepository) SearchUnsafe(ctx context.Context, title string) ([]Task, error) {
	query := "SELECT id, title, description, done FROM tasks WHERE title = '" + title + "'"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// SearchSafe — безопасный поиск с параметризованным запросом
func (r *TaskRepository) SearchSafe(ctx context.Context, title string) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, done FROM tasks WHERE title = $1
	`, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Done); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// Create — создаёт новую задачу и возвращает её
func (r *TaskRepository) Create(ctx context.Context, title, description string) (Task, error) {
	var t Task
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tasks (title, description)
		VALUES ($1, $2)
		RETURNING id, title, description, done
	`, title, description).Scan(&t.ID, &t.Title, &t.Description, &t.Done)
	return t, err
}
