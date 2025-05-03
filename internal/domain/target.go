package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Target represents an individual recipient in the phishing simulation.
type Target struct {
	UUID      uuid.UUID  `db:"uuid"`
	FullName  string     `db:"full_name"`
	Email     string     `db:"email"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	SentAt    *time.Time `db:"sent_at"`    // Pointer to handle NULL timestamps easily
	ClickedAt *time.Time `db:"clicked_at"` // Pointer to handle NULL timestamps easily
}

// NewTarget creates a new Target instance with a generated UUID and timestamps.
func NewTarget(fullName, email string) *Target {
	return &Target{
		UUID:      uuid.New(),
		FullName:  fullName,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		SentAt:    nil, // Explicitly nil
		ClickedAt: nil, // Explicitly nil
	}
}

// --- Add UUID parsing helper ---
// In domain/target.go or a new domain/uuid.go

// ParseUUID safely parses a string into a UUID.
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format '%s': %w", s, err)
	}
	return id, nil
}
