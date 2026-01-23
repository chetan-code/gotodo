package models

import "time"

type Task struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Title     string    `json:"title"`
	IsDone    bool      `json:"is_done"`
	CreatedAt time.Time `json:"created_at"`
}

// User represent the authenticated person
type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
