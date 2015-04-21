package models

import (
	"encoding/json"
)

type nullString struct {
	str   string
	empty bool
}

func NullString(s string) nullString {
	return nullString{s, false}
}

func (s nullString) MarshalJSON() ([]byte, error) {
	var value interface{}

	if s.str != "" {
		value = s.str
	}

	return json.Marshal(value)
}

func (s *nullString) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &s.str); err != nil {
		return err
	}

	s.empty = s.str == ""
	return nil
}

func (s *nullString) String() string {
	return s.str
}
