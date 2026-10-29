package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"food-kiosk/database"
)

// Credential is a row of the additive credentials table.
type Credential struct {
	ID           int64
	Principal    string
	PasswordHash string
	Role         Role
	UserID       sql.NullInt64
	CreatedAt    string
}

// normalizePrincipal canonicalises a login name the same way users.roll_no is
// canonicalised, so a customer registers and signs in with any casing.
func normalizePrincipal(principal string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(principal))
	if normalized == "" {
		return "", fmt.Errorf("%w: principal is empty", database.ErrInvalidInput)
	}
	return normalized, nil
}

// CreateCredential stores a hashed secret for a principal.
func CreateCredential(ctx context.Context, db database.DB, principal, passwordHash string, role Role, userID *int64) (int64, error) {
	normalized, err := normalizePrincipal(principal)
	if err != nil {
		return 0, err
	}

	var uid any
	if userID != nil {
		uid = *userID
	}

	result, err := db.ExecContext(ctx,
		`INSERT INTO credentials (principal, password_hash, role, user_id) VALUES (?, ?, ?, ?)`,
		normalized, passwordHash, string(role), uid)
	if err != nil {
		return 0, fmt.Errorf("insert credential: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get inserted credential ID: %w", err)
	}
	return id, nil
}

// GetCredentialByPrincipal looks a login up by its canonical principal.
func GetCredentialByPrincipal(ctx context.Context, db database.DB, principal string) (*Credential, error) {
	normalized, err := normalizePrincipal(principal)
	if err != nil {
		return nil, err
	}

	var (
		c    Credential
		role string
	)
	err = db.QueryRowContext(ctx,
		`SELECT id, principal, password_hash, role, user_id, created_at FROM credentials WHERE principal = ?`,
		normalized,
	).Scan(&c.ID, &c.Principal, &c.PasswordHash, &role, &c.UserID, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: credentials for %q", database.ErrNotFound, normalized)
	}
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	c.Role = Role(role)
	return &c, nil
}

// UpsertAdminCredential creates or refreshes an admin login. Idempotent, so it
// is safe to run on every startup: an existing principal has its password
// rotated rather than failing on the UNIQUE constraint.
func UpsertAdminCredential(ctx context.Context, db database.DB, principal, passwordHash string) error {
	normalized, err := normalizePrincipal(principal)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO credentials (principal, password_hash, role, user_id) VALUES (?, ?, 'admin', NULL)
		 ON CONFLICT(principal) DO UPDATE SET password_hash = excluded.password_hash, role = 'admin'`,
		normalized, passwordHash)
	if err != nil {
		return fmt.Errorf("upsert admin credential: %w", err)
	}
	return nil
}

// BootstrapAdmin creates or refreshes the admin credential named in the config.
func BootstrapAdmin(ctx context.Context, db database.DB, principal, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return UpsertAdminCredential(ctx, db, principal, hash)
}
