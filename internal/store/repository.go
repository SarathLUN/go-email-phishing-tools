package store

import (
	"context"
	"github.com/SarathLUN/go-email-phishing-tools/internal/domain" // Make sure the module path is correct
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
}

// Add other repository interfaces here if needed (e.g., CampaignRepository)
