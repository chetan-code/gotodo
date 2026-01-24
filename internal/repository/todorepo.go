package repository

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/chetan-code/gotodo/internal/models"
)

type TodoRepo struct {
	db *sql.DB
}

type TodoStats struct {
	Pending   int
	Completed int
	Total     int
}

func NewTodoRepo(db *sql.DB) (*TodoRepo, error) {
	repo := &TodoRepo{db: db}
	return repo, nil
}

func (r *TodoRepo) FetchTasks(email string, search string, status string) ([]models.Task, error) {
	//base query
	query := "SELECT id, email, title, is_done FROM todos WHERE email = $1"
	args := []any{email}
	argCount := 2 //we were using $1, $2 etc but we can have multiple such fmt tags

	//dynamically append filters
	if search != "" {
		query += fmt.Sprintf(" AND title ILIKE $%d", argCount) //$2
		args = append(args, "%"+search+"%")
		argCount++
	}

	switch status {
	case "completed":
		query += fmt.Sprintf(" AND is_done = $%d", argCount)
		args = append(args, true)
		argCount++
	case "pending":
		query += fmt.Sprintf(" AND is_done = $%d", argCount)
		args = append(args, false)
		argCount++
	}

	query += " ORDER BY is_done ASC, id DESC"

	row, err := r.db.Query(query, args...)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "select table row for email",
			"error", err,
			"email", email,
		)
		return nil, err
	}
	defer row.Close() //close the connect in the end
	var tasks []models.Task
	for row.Next() {
		var t models.Task
		err := row.Scan(&t.ID, &t.Email, &t.Title, &t.IsDone)
		if err != nil {
			slog.Error("database_scan_failed",
				"op", "scan row failed",
				"error", err,
				"email", email,
			)
			continue
		}
		tasks = append(tasks, t)
	}
	slog.Debug("database_query_success",
		"op", "select table row for email",
		"email", email,
	)
	return tasks, nil
}

func (r *TodoRepo) AddTask(email string, title string) error {
	/*In Go, we use placeholders ($1, $2) instead of string formatting (like fmt.Sprintf). This tells
	the database driver to "sanitize" the input, which prevents SQL Injection attacks
	(where a user might try to type a command into your task box to delete your whole database).*/
	addTaskQuery := "INSERT INTO todos (email, title) VALUES ($1, $2)"
	_, err := r.db.Exec(addTaskQuery, email, title)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "insert into table",
			"error", err,
			"email", email,
			"title", title,
		)
	}
	slog.Debug("database_query_success",
		"op", "insert into table",
		"email", email,
	)
	return err
}

func (r *TodoRepo) DeleteTask(id int, email string) error {
	query := "DELETE FROM todos WHERE id = $1 AND email = $2"
	_, err := r.db.Exec(query, id, email)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "delete table row for email and id",
			"error", err,
			"email", email,
			"id", id,
		)
	}
	slog.Debug("database_query_success",
		"op", "delete table row for email and id",
		"email", email,
		"id", id,
	)
	return err
}

func (r *TodoRepo) ToggleTask(id int, email string) error {
	query := "UPDATE todos SET is_done = NOT is_done WHERE id = $1 AND email = $2"
	_, err := r.db.Exec(query, id, email)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "update table row for id and email",
			"error", err,
			"email", email,
			"id", id,
		)
	}
	slog.Debug("database_query_success",
		"op", "update table row for id and email",
		"email", email,
		"id", id,
	)
	return err
}

func (r *TodoRepo) RemoveAllTask(email string) error {
	query := "DELETE FROM todos WHERE email = $1"
	_, err := r.db.Exec(query, email)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "delete table rows for email",
			"error", err,
			"email", email,
		)
	}
	slog.Debug("database_query_success",
		"op", "delete table rows for email",
		"email", email,
	)
	return err
}

func (r *TodoRepo) GetStats(email string) (TodoStats, error) {
	query := `
			SELECT
				COUNT(*) FILTER (WHERE is_done = false) as pending,
				COUNT(*) FILTER (WHERE is_done = true) as completed,
				COUNT(*) as total
			FROM todos
			WHERE email = $1`
	var stats TodoStats
	err := r.db.QueryRow(query, email).Scan(&stats.Pending, &stats.Completed, &stats.Total)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "stats from table rows",
			"error", err,
			"email", email)
		return stats, err
	}

	return stats, nil
}
