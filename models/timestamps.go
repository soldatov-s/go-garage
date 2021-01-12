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
	t.CreatedAt.Time = timeNow
	t.CreatedAt.Valid = true
	t.UpdatedAt.Time = timeNow
	t.UpdatedAt.Valid = true
}

func (t *Timestamp) UpdateTimestamp() {
	timeNow := time.Now()
	t.UpdatedAt.Time = timeNow
	t.UpdatedAt.Valid = true
}

func (t *Timestamp) DeleteTimestamp() {
	timeNow := time.Now()
	t.DeletedAt.Time = timeNow
	t.DeletedAt.Valid = true
}
