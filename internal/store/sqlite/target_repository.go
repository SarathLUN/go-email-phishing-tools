package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SarathLUN/go-email-phishing-tools/internal/domain"
	"github.com/SarathLUN/go-email-phishing-tools/internal/store"
	"log"
	"strings"

	"github.com/mattn/go-sqlite3"
)

// sqliteTargetRepository implements the store.TargetRepository interface for SQLite.
type sqliteTargetRepository struct {
	db *sql.DB
}

// NewSQLiteTargetRepository creates a new repository instance.
func NewSQLiteTargetRepository(db *sql.DB) store.TargetRepository {
	return &sqliteTargetRepository{db: db}
}

// Create inserts a single new target.
func (r *sqliteTargetRepository) Create(ctx context.Context, target *domain.Target) error {
	query := `INSERT INTO targets (uuid, full_name, email, created_at, updated_at, sent_at, clicked_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		target.UUID.String(), // Store UUID as string
		target.FullName,
		target.Email,
		target.CreatedAt,
		target.UpdatedAt,
		target.SentAt,    // Will be NULL if pointer is nil
		target.ClickedAt, // Will be NULL if pointer is nil
	)

	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			// Check for UNIQUE constraint violation (code 19 constraint 1555)
			// See https://www.sqlite.org/rescode.html
			if sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				// Check if it's the email constraint
				if strings.Contains(sqliteErr.Error(), "targets.email") {
					return fmt.Errorf("%w: email '%s'", store.ErrDuplicateEmail, target.Email)
				}
				// Could be the UUID, though highly unlikely
				if strings.Contains(sqliteErr.Error(), "targets.uuid") {
					return fmt.Errorf("%w: uuid '%s'", store.ErrDuplicateUUID, target.UUID.String())
				}
				// Some other unique constraint violation
				return fmt.Errorf("database constraint violation: %w", err)
			}
		}
		return fmt.Errorf("failed to insert target: %w", err)
	}
	return nil
}

// BulkCreate inserts multiple targets using a transaction for efficiency.
// It skips targets with duplicate emails and returns the count of newly inserted targets.
func (r *sqliteTargetRepository) BulkCreate(ctx context.Context, targets []*domain.Target) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if anything goes wrong before commit

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO targets (uuid, full_name, email, created_at, updated_at, sent_at, clicked_at)
	                                    VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	var insertedCount int64 = 0
	var skippedEmails []string

	for _, target := range targets {
		_, err := stmt.ExecContext(ctx,
			target.UUID.String(),
			target.FullName,
			target.Email,
			target.CreatedAt,
			target.UpdatedAt,
			target.SentAt,
			target.ClickedAt,
		)
		if err != nil {
			var sqliteErr sqlite3.Error
			if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique && strings.Contains(sqliteErr.Error(), "targets.email") {
				// Skip duplicate email, log it
				skippedEmails = append(skippedEmails, target.Email)
				continue // Move to the next target
			}
			// For other errors, rollback the whole transaction
			return 0, fmt.Errorf("failed to execute insert for email '%s': %w", target.Email, err)
		}
		insertedCount++
	}

	if len(skippedEmails) > 0 {
		log.Printf("Skipped %d targets due to duplicate emails: %v", len(skippedEmails), skippedEmails)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return insertedCount, nil
}

// FindByEmail retrieves a target by its email address. Returns nil, nil if not found.
func (r *sqliteTargetRepository) FindByEmail(ctx context.Context, email string) (*domain.Target, error) {
	query := `SELECT uuid, full_name, email, created_at, updated_at, sent_at, clicked_at
	          FROM targets WHERE email = ?`
	row := r.db.QueryRowContext(ctx, query, email)

	var target domain.Target
	var uuidStr string // Read UUID as string first
	err := row.Scan(
		&uuidStr,
		&target.FullName,
		&target.Email,
		&target.CreatedAt,
		&target.UpdatedAt,
		&target.SentAt,
		&target.ClickedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Standard way to indicate not found
		}
		return nil, fmt.Errorf("failed to query target by email '%s': %w", email, err)
	}

	// Parse UUID string
	parsedUUID, parseErr := domain.ParseUUID(uuidStr)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse UUID '%s' from database for email '%s': %w", uuidStr, email, parseErr)
	}
	target.UUID = parsedUUID

	return &target, nil
}
