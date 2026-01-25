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
	query := "SELECT id, email, title, is_done, worker_email FROM todos WHERE email = $1"
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
		err := row.Scan(&t.ID, &t.Email, &t.Title, &t.IsDone, &t.WorkerEmail)
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

func (r *TodoRepo) AddTask(email string, title string, workerEmail string) error {
	/*In Go, we use placeholders ($1, $2) instead of string formatting (like fmt.Sprintf). This tells
	the database driver to "sanitize" the input, which prevents SQL Injection attacks
	(where a user might try to type a command into your task box to delete your whole database).*/
	addTaskQuery := "INSERT INTO todos (email, title, worker_email) VALUES ($1, $2, $3)"
	_, err := r.db.Exec(addTaskQuery, email, title, workerEmail)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "insert into table",
			"error", err,
			"email", email,
			"title", title,
			"worker_email", workerEmail,
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
	query := `UPDATE todos SET is_done = NOT is_done 
			WHERE id = $1 AND (email = $2 OR worker_email = $2)`
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

func (r *TodoRepo) SendInvite(managerEmail string, workerEmail string) error {
	if managerEmail == workerEmail {
		return fmt.Errorf("cannot invite self")
	}

	//insert new only if not exists
	query := `
			INSERT INTO relationships (manager_email, worker_email, status)
			VALUES ($1, $2, 'pending')
			ON CONFLICT (manager_email, worker_email) DO NOTHING`
	_, err := r.db.Exec(query, managerEmail, workerEmail)
	if err != nil {
		slog.Error("database_query_failed",
			"op", "insert into relationship where manager and worker",
			"error", err,
			"worker_email", workerEmail,
			"manager_email", managerEmail,
		)
		return err
	}
	slog.Debug("database_query_success",
		"op", "invite sent to worker",
		"worker_email", workerEmail)
	return nil
}

func (r *TodoRepo) FetchPendingInvites(workerEmail string) ([]models.Relationship, error) {
	query := `
			SELECT id, manager_email, created_at FROM relationships
			WHERE worker_email = $1 AND status = 'pending'
			ORDER BY created_at DESC`
	rows, err := r.db.Query(query, workerEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []models.Relationship
	for rows.Next() {
		var rel models.Relationship
		err = rows.Scan(&rel.ID, &rel.ManagerEmail, &rel.CreatedAt)
		if err == nil {
			invites = append(invites, rel)
		}
	}
	slog.Debug("database_query_success",
		"op", "select pending request for worker",
		"worker_email", workerEmail)
	return invites, nil
}

// accep to reject req
func (r *TodoRepo) RespondToInvite(id int, status string) error {
	query := `
			UPDATE relationships SET status = $1
			WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	slog.Debug("database_query_success",
		"op", "reponded to invite id with status",
		"id", id,
		"status", status)
	return err
}

// workers that have accepted my invite
func (r *TodoRepo) FetchMyWorkers(managerEmail string) ([]string, error) {
	query := `
			SELECT worker_email FROM relationships
			WHERE manager_email = $1 AND status = 'accepted'`
	rows, err := r.db.Query(query, managerEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []string
	for rows.Next() {
		var w string
		err := rows.Scan(&w)
		if err != nil {
			slog.Error("database_scan_failed",
				"op", "fetch worker scan",
				"error", err)
			continue
		}
		workers = append(workers, w)
	}
	slog.Debug("database_query_success",
		"op", "fetch workers for manager",
		"manager_email", managerEmail)
	return workers, nil
}

func (r *TodoRepo) FetchAssignedToMe(myEmail string) ([]models.Task, error) {
	// Tasks where I am the worker, but someone else is the creator
	query := `
        SELECT id, email, title, is_done 
        FROM todos 
        WHERE worker_email = $1 AND email != $1
        ORDER BY created_at DESC`

	rows, err := r.db.Query(query, myEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Email, &t.Title, &t.IsDone); err == nil {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}
