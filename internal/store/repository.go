package store

import (
	"context"
	"time"

	"github.com/SarathLUN/go-email-phishing-tools/internal/domain" // Make sure the module path is correct
	"github.com/google/uuid"
)

// TargetRepository defines the operations for persisting and retrieving Target data.
type TargetRepository interface {
	// Create inserts a single new target into the database.
	Create(ctx context.Context, target *domain.Target) error
	// BulkCreate inserts multiple targets efficiently, often using a transaction.
	BulkCreate(ctx context.Context, targets []*domain.Target) (int64, error) // Returns count of successfully inserted rows
	// FindByEmail checks if a target with the given email exists.
	FindByEmail(ctx context.Context, email string) (*domain.Target, error)
	// Add methods for Stage 2 later (e.g., FindNonSent, MarkAsSent)

	// --- new methods for stage 2 ---
	// FindNonSend retrieves all targets that have not yet been sent and email (sent_at IS NULL)
	FindNonSent(ctx context.Context) ([]*domain.Target, error)

	// MarkAsSent updates the sent_at timestamp for a given target UUID.
	MarkAsSent(ctx context.Context, uuid uuid.UUID, sentTime time.Time) error

	// --- New method for Stage 3 ---
	// MarkAsClicked updates the clicked_at timestamp for a given target UUID,
	// only if clicked_at is currently NULL. Returns true if the row was updated.
	MarkAsClicked(ctx context.Context, uuid uuid.UUID, clickedTime time.Time) (bool, error)
}
