package models

import "time"

type ApplicationCredentials struct {
	ID            uint64    `db:"id"`
	ApplicationID uint64    `db:"application_id"`
	Platform      string    `db:"platform"`
	Username      string    `db:"username"`
	Password      string    `db:"password"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
