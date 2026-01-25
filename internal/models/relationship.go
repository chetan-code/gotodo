package models

import "time"

type Relationship struct {
	ID           int       `json:"id"`
	ManagerEmail string    `json:"manager_email"`
	WorkerEmail  string    `json:"worker_email"`
	Status       string    `json:"status"` //"pending" , "accepted"
	CreatedAt    time.Time `json:"created_at"`
}
