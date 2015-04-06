package models

// Error is a envelope for error messages.
type Error struct {
	// Details error description.
	Text string `json:"error"`
}
