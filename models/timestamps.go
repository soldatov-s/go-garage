package models

import (
	"time"

	"github.com/soldatov-s/go-garage/types"
)

type Timestamp struct {
	CreatedAt types.NullTime `json:"created_at" db:"created_at"`
	UpdatedAt types.NullTime `json:"updated_at" db:"updated_at"`
	DeletedAt types.NullTime `json:"deleted_at" db:"deleted_at"`
}

func (t *Timestamp) CreateTimestamp() {
	timeNow := time.Now()
	t.CreatedAt.SetTime(timeNow)
	t.UpdatedAt.SetTime(timeNow)
}

func (t *Timestamp) UpdateTimestamp() {
	t.UpdatedAt.SetNow()
}

func (t *Timestamp) DeleteTimestamp() {
	timeNow := time.Now()
	t.UpdatedAt.SetTime(timeNow)
	t.DeletedAt.SetTime(timeNow)
}
