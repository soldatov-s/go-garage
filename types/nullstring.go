package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// NullString is a wrapper around sql.NullString
type NullString struct {
	sql.NullString
}

// MarshalJSON method is called by json.Marshal,
// whenever it is of type NullString
func (x *NullString) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(x.String)
}

// UnmarshalJSON method is called by json.Unmarshal,
// whenever it is of type NullString
func (x *NullString) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &x.String); err != nil {
		return err
	}

	x.Valid = true

	return nil
}

// Scan implements the Scanner interface.
func (x *NullString) Scan(value interface{}) error {
	return x.NullString.Scan(value)
}

// Value implements the driver Valuer interface.
func (x NullString) Value() (driver.Value, error) {
	return x.NullString.Value()
}
