package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice stores a JSON array (e.g. ["gost","iptables"]) in the database and
// marshals to/from JSON as []string.
type StringSlice []string

func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}

	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unsupported Scan type for StringSlice: %T", value)
	}

	if raw == "" {
		*s = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		// Fallback: treat it as a single value.
		*s = StringSlice{raw}
		return nil
	}

	*s = StringSlice(arr)
	return nil
}

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}
