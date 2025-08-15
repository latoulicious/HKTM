package utilities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSONB is a reusable type for PostgreSQL jsonb columns
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONB: %w", err)
	}
	return bytes, nil
}

func (j *JSONB) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}

	bytes, ok := src.([]byte)
	if !ok {
		return errors.New("JSONB scan: type assertion to []byte failed")
	}

	if err := json.Unmarshal(bytes, j); err != nil {
		return fmt.Errorf("JSONB scan: failed to unmarshal: %w", err)
	}
	return nil
}
