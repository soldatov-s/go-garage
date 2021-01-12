package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// NullMeta represents a map[string]interface{} that may be null.
// NullMeta implements the Scanner interface so
// it can be used as a scan destination
type NullMeta struct {
	Map   map[string]interface{}
	Valid bool // Valid is true if Map is not NULL
}

// Make the NullMeta struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (x NullMeta) Value() (driver.Value, error) {
	if !x.Valid {
		return nil, nil
	}

	return json.Marshal(x.Map)
}

// Make the NullMeta struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (x *NullMeta) Scan(value interface{}) error {
	if value == nil {
		x.Map, x.Valid = make(map[string]interface{}), false
		return nil
	}

	b, ok := value.([]byte)

	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	x.Valid = true

	return json.Unmarshal(b, &x.Map)
}

// MarshalJSON method is called by json.Marshal,
// whenever it is of type NullMeta
func (x *NullMeta) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(x.Map)
}

// UnmarshalJSON method is called by json.Unmarshal,
// whenever it is of type NullMeta
func (x *NullMeta) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &x.Map); err != nil {
		return err
	}

	x.Valid = true

	return nil
}
