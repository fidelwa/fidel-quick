package featureflags

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/theluisbolivar/fidel-quick/internal/apperror"
)

// Repository is the persistence port for feature flags.
type Repository interface {
	List(ctx context.Context) ([]Flag, error)
	Get(ctx context.Context, key string) (*Flag, error)
	Upsert(ctx context.Context, key string, in UpdateInput) (*Flag, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const flagColumns = `key, enabled_globally, customer_overrides, default_value, description, created_at, updated_at`

func scanFlag(row interface{ Scan(...any) error }) (*Flag, error) {
	var f Flag
	var overrides []byte
	var desc sql.NullString
	if err := row.Scan(
		&f.Key, &f.EnabledGlobally, &overrides, &f.DefaultValue, &desc, &f.CreatedAt, &f.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(overrides) > 0 {
		if err := json.Unmarshal(overrides, &f.CustomerOverrides); err != nil {
			return nil, fmt.Errorf("unmarshal customer_overrides: %w", err)
		}
	}
	if f.CustomerOverrides == nil {
		f.CustomerOverrides = map[string]bool{}
	}
	f.Description = desc.String
	return &f, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Flag, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+flagColumns+` FROM feature_flags ORDER BY key`)
	if err != nil {
		return nil, apperror.Internal("failed to list feature flags", err)
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		f, err := scanFlag(rows)
		if err != nil {
			return nil, apperror.Internal("failed to scan feature flag", err)
		}
		flags = append(flags, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal("failed to iterate feature flags", err)
	}
	return flags, nil
}

func (r *PostgresRepository) Get(ctx context.Context, key string) (*Flag, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+flagColumns+` FROM feature_flags WHERE key = $1`, key)
	f, err := scanFlag(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NotFound("feature flag not found", nil)
	}
	if err != nil {
		return nil, apperror.Internal("failed to query feature flag", err)
	}
	return f, nil
}

// Upsert creates or updates a flag. It reads the current row (if any), applies
// the non-nil fields of the input, and writes it back — so a partial update
// only touches the fields the caller provided.
func (r *PostgresRepository) Upsert(ctx context.Context, key string, in UpdateInput) (*Flag, error) {
	current, err := r.Get(ctx, key)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == "not_found" {
			current = &Flag{Key: key, CustomerOverrides: map[string]bool{}}
		} else {
			return nil, err
		}
	}

	if in.EnabledGlobally != nil {
		current.EnabledGlobally = *in.EnabledGlobally
	}
	if in.DefaultValue != nil {
		current.DefaultValue = *in.DefaultValue
	}
	if in.Description != nil {
		current.Description = *in.Description
	}
	if in.CustomerOverrides != nil {
		current.CustomerOverrides = in.CustomerOverrides
	}
	if current.CustomerOverrides == nil {
		current.CustomerOverrides = map[string]bool{}
	}

	overrides, err := json.Marshal(current.CustomerOverrides)
	if err != nil {
		return nil, apperror.Internal("failed to marshal customer_overrides", err)
	}

	var descArg interface{}
	if current.Description != "" {
		descArg = current.Description
	}

	row := r.db.QueryRowContext(ctx,
		`INSERT INTO feature_flags (key, enabled_globally, customer_overrides, default_value, description)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (key) DO UPDATE SET
			enabled_globally = EXCLUDED.enabled_globally,
			customer_overrides = EXCLUDED.customer_overrides,
			default_value = EXCLUDED.default_value,
			description = EXCLUDED.description,
			updated_at = NOW()
		 RETURNING `+flagColumns,
		key, current.EnabledGlobally, overrides, current.DefaultValue, descArg,
	)
	f, err := scanFlag(row)
	if err != nil {
		return nil, apperror.Internal("failed to upsert feature flag", err)
	}
	return f, nil
}
