package models

import "time"

type Device struct {
	ID        uint64    `db:"id"`
	UID       string    `db:"uid"`
	Platform  string    `db:"platform"`
	Language  string    `db:"language"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
