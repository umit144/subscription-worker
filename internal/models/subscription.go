package models

import "time"

type Subscription struct {
	ID            uint64
	DeviceID      uint64
	ApplicationID uint64
	Receipt       string
	Status        int8
	ExpireDate    time.Time
	Credentials   *ApplicationCredential
}

type ApplicationCredential struct {
	ID            uint64
	ApplicationID uint64
	Platform      string
	Username      string
	Password      string
}
