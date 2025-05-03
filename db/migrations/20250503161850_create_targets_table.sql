-- +goose Up
-- +goose StatementBegin
CREATE TABLE targets (
    uuid TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME NULL,
    clicked_at DATETIME NULL
);

-- Create a trigger to automatically update updated_at on row changes
CREATE TRIGGER update_targets_updated_at
AFTER UPDATE ON targets
FOR EACH ROW
BEGIN
    UPDATE targets SET updated_at = CURRENT_TIMESTAMP WHERE uuid = OLD.uuid;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_targets_updated_at;
DROP TABLE IF EXISTS targets;
-- +goose StatementEnd
