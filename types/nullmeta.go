package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

var ErrTypeAssertion = errors.New("type assertion to []byte failed")

// NullMeta represents a map[string]interface{} that may be null.
// NullMeta implements the Scanner interface so
// it can be used as a scan destination
type NullMeta struct {
	Map   map[string]interface{}
	Valid bool // Valid is true if Map is not NULL
}

// Init initilze/reinitilize NullMeta
func (x *NullMeta) Init() {
	x.Map = make(map[string]interface{})
	x.Valid = true
}

// Add adds new item to NullMeta
func (x *NullMeta) Add(key string, value interface{}) {
	if !x.Valid {
		x.Init()
	}
	x.Map[key] = value
}

// Delete deletes item to NullMeta
func (x *NullMeta) Delete(key string) {
	delete(x.Map, key)
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
		return ErrTypeAssertion
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
