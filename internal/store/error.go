package store

import "errors"

// Custom error types for the store package.
// These allow callers to check for specific database-related issues.
var (
	// ErrDuplicateEmail indicates an attempt to insert a target
	// with an email address that already exists in the database.
	ErrDuplicateEmail = errors.New("email already exists")

	// ErrDuplicateUUID indicates an attempt to insert a record
	// with a UUID that already exists (should be extremely rare).
	ErrDuplicateUUID = errors.New("uuid already exists")

	// ErrNotFound indicates that a query expected to return a record
	// found no matching record. Useful for abstracting sql.ErrNoRows.
	ErrNotFound = errors.New("record not found")
)

// You can add more store-specific errors here as needed.
// For example:
// var ErrOptimisticLock = errors.New("record update conflict")
