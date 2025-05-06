package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/SarathLUN/go-email-phishing-tools/internal/domain"
	"github.com/SarathLUN/go-email-phishing-tools/internal/store"
	"github.com/google/uuid"

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

// FindNonSent retrieves all targets where sent_at is NULL.
func (r *sqliteTargetRepository) FindNonSent(ctx context.Context) ([]*domain.Target, error) {
	query := `
		SELECT uuid, full_name, email, created_at, updated_at, sent_at, clicked_at
		FROM targets
		WHERE sent_at IS NULL 
		ORDER BY created_at ASC 
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query non-sent targets: %w", err)
	}
	defer rows.Close()

	targets := []*domain.Target{} // initialize empty slice
	for rows.Next() {
		var target domain.Target
		var uuidStr string
		// need to scan all columns returned by the query.
		err := rows.Scan(
			&uuidStr,
			&target.FullName,
			&target.Email,
			&target.CreatedAt,
			&target.UpdatedAt,
			&target.SentAt,    // will scan as null if the DB value is null
			&target.ClickedAt, // will scan as null if the DB value is null
		)
		if err != nil {
			// Log error for the specific row and continue if possible, or return accumulated error
			log.Printf("Error scanning target row: %v", err)
			continue // Skip this row on scan error
		}
		// parse UUID string
		parseUUID, parseErr := domain.ParseUUID(uuidStr)
		if parseErr != nil {
			log.Printf("Error parsing UUID '%s' from database for non-sent target: %v", uuidStr, parseErr)
			continue // Skip row with invalid UUID
		}
		target.UUID = parseUUID
		targets = append(targets, &target)
	}
	// check for errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating non-sent target rows: %w", err)
	}

	return targets, nil
}

// MarkAsSent updates the sent_at timestamp for the target with the given UUID.
// It relies on the database trigger to update 'updated_at'.
func (r *sqliteTargetRepository) MarkAsSent(ctx context.Context, uuid uuid.UUID, sentTime time.Time) error {
	query := `UPDATE targets SET sent_at = ? WHERE uuid = ?`
	result, err := r.db.ExecContext(ctx, query, sentTime, uuid.String())
	if err != nil {
		return fmt.Errorf("failed to update sent_at for target UUID %s: %w", uuid.String(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Log this error but don't necessarily fail the operation if update succeeded
		log.Printf("Warning: Could not get rows affected after marking target %s as sent: %v", uuid.String(), err)
	} else if rowsAffected == 0 {
		// This means the UUID didn't exist, which is unexpected here
		// Return ErrNotFound or a specific error
		log.Printf("Warning: Attempted to mark non-existent target UUID %s as sent.", uuid.String())
		return fmt.Errorf("target UUID %s not found: %w", uuid.String(), store.ErrNotFound)
	} else if rowsAffected > 1 {
		// Should not happen with UUID as primary key
		log.Printf("Warning: Expected 1 row affected but got %d for UUID %s", rowsAffected, uuid.String())
	}

	return nil
}

// MarkAsClicked updates the clicked_at timestamp for the target with the given UUID,
// only if clicked_at is currently NULL. It relies on the database trigger to update 'updated_at'.
// Returns true if the clicked_at field was updated, false otherwise (e.g., already clicked or not found).
func (r *sqliteTargetRepository) MarkAsClicked(ctx context.Context, uuid uuid.UUID, clickedTime time.Time) (bool, error) {
	query := `UPDATE targets SET clicked_at = ? WHERE uuid = ? AND clicked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, clickedTime, uuid.String())
	if err != nil {
		return false, fmt.Errorf("failed to update clicked_at for target UUID %s: %w", uuid.String(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// This is an error in fetching RowsAffected, not necessarily in the update itself if it happened.
		// Log it, but base success on rowsAffected if available.
		log.Printf("Warning: Could not get rows affected after marking target %s as clicked: %v", uuid.String(), err)
		// Consider returning the error if critical, or false if rowsAffected might still be zero.
		// For simplicity, if we can't get RowsAffected, assume update might not have occurred as expected.
		return false, fmt.Errorf("failed to get rows affected for clicked_at update (UUID: %s): %w", uuid.String(), err)
	}

	if rowsAffected == 0 {
		// This could mean the UUID doesn't exist OR clicked_at was already set.
		// We can't distinguish without another query, but for this function's contract,
		// it means clicked_at was not newly updated.
		log.Printf("Target UUID %s not updated (either not found or already clicked).", uuid.String())
		return false, nil // Not an error per se, just no update occurred.
	}
	if rowsAffected > 1 {
		// Should not happen with UUID as primary key
		log.Printf("CRITICAL: Expected 0 or 1 row affected for click tracking but got %d for UUID %s", rowsAffected, uuid.String())
		// This is a more serious issue.
		return true, fmt.Errorf("unexpected number of rows affected (%d) for click tracking (UUID: %s)", rowsAffected, uuid.String())
	}

	return true, nil // Update occurred
}
