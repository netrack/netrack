package models

type Mechanism struct {
	// Mechanism name
	Name string `json:"name"`

	// Mechanism description
	Description string `json:"description"`

	// Mechanism state
	State string `json:"state"`
}
