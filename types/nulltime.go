package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// NullString is a wrapper around sql.NullString
type NullTime struct {
	sql.NullTime
}

// SetTime sets time to NullTime
func (x *NullTime) SetTime(t time.Time) {
	x.Valid = true
	x.Time = t
}

// SetNow sets current UTC time to NullTime structure
func (x *NullTime) SetNow() {
	x.SetTime(time.Now().UTC())
}

// MarshalJSON method is called by json.Marshal,
// whenever it is of type NullString
func (x *NullTime) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(x.Time)
}

// UnmarshalJSON method is called by json.Unmarshal,
// whenever it is of type NullTime
func (x *NullTime) UnmarshalJSON(data []byte) error {
	var v time.Time

	if err := json.Unmarshal(data, &v); err != nil {
		return errors.Wrap(err, "unmarshal")
	}

	if v.Year() == 1 {
		return nil
	}

	x.Time = v
	x.Valid = true

	return nil
}

// Scan implements the Scanner interface.
func (x *NullTime) Scan(value interface{}) error {
	return x.NullTime.Scan(value)
}

// Value implements the driver Valuer interface.
func (x NullTime) Value() (driver.Value, error) {
	return x.NullTime.Value()
}

// Timestamp create timestamp and fill NullTime struct
func (x *NullTime) Timestamp() {
	timeNow := time.Now()
	x.Time = timeNow
	x.Valid = true
}

// Timestamp create UTC timestamp and fill NullTime struct
func (x *NullTime) TimestampUTC() {
	timeNow := time.Now().UTC()
	x.Time = timeNow
	x.Valid = true
}
