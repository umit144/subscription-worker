package models

import "time"

type Subscription struct {
	ID            uint64    `db:"id"`
	DeviceID      uint64    `db:"device_id"`
	ApplicationID uint64    `db:"application_id"`
	Receipt       string    `db:"receipt"`
	Status        int8      `db:"status"`
	ExpireDate    time.Time `db:"expire_date"`
}
