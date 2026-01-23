package repository

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/chetan-code/gotodo/internal/models"
)

type TodoRepo struct {
	db *sql.DB
}

func NewTodoRepo(db *sql.DB) (*TodoRepo, error) {
	repo := &TodoRepo{db: db}

	err := repo.CreateTable()
	if err != nil {
		return nil, fmt.Errorf("could not initialize table: %w", err)
	}

	return repo, nil
}

func (r *TodoRepo) FetchTask(email string) ([]models.Task, error) {
	query := "SELECT id, email, title, is_done FROM todos WHERE email = $1 ORDER BY is_done ASC"
	row, err := r.db.Query(query, email)
	if err != nil {
		log.Println("Error fetching task ", err)
		return nil, err
	}
	defer row.Close() //close the connect in the end
	var tasks []models.Task
	for row.Next() {
		var t models.Task
		err := row.Scan(&t.ID, &t.Email, &t.Title, &t.IsDone)
		if err != nil {
			log.Printf("SCAN ERROR: %v", err) // This would say "sql: expected 1 destination arguments in Scan, not 3"
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *TodoRepo) CreateTable() error {
	createTableQuery := `CREATE TABLE IF NOT EXISTS todos(
		id SERIAL PRIMARY KEY,
		email TEXT NOT NULL,
		task TEXT NOT NULL
	);`
	_, err := r.db.Exec(createTableQuery)
	return err
}

func (r *TodoRepo) AddTaskDB(email string, title string) error {
	/*In Go, we use placeholders ($1, $2) instead of string formatting (like fmt.Sprintf). This tells
	the database driver to "sanitize" the input, which prevents SQL Injection attacks
	(where a user might try to type a command into your task box to delete your whole database).*/
	addTaskQuery := "INSERT INTO todos (email, title) VALUES ($1, $2)"
	_, err := r.db.Exec(addTaskQuery, email, title)
	return err
}

func (r *TodoRepo) DeleteTask(id int, email string) error {
	query := "DELETE FROM todos WHERE id = $1 AND email = $2"
	_, err := r.db.Exec(query, id, email)
	return err
}

func (r *TodoRepo) ToggleTask(id int, email string) error {
	query := "UPDATE todos SET is_done = NOT is_done WHERE id = $1 AND email = $2"
	_, err := r.db.Exec(query, id, email)
	if err != nil {
		log.Printf("Repo : Update toggle failed")
	} else {
		fmt.Println("[Update status success is done for id :]")
	}
	return err
}

func (r *TodoRepo) RemoveAllTask(email string) error {
	query := "DELETE FROM todos WHERE email = $1"
	_, err := r.db.Exec(query, email)
	if err != nil {
		log.Printf("Error deleting task : %s \n", err)
	}
	return err
}
