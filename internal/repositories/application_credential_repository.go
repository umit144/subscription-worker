package repositories

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/blockloop/scan/v2"
	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/models"
)

type ApplicationCredentialsRepository struct {
	db *database.Database
}

func NewApplicationCredentialsRepository(db *database.Database) *ApplicationCredentialsRepository {
	return &ApplicationCredentialsRepository{db: db}
}

func (r *ApplicationCredentialsRepository) FetchByIds(ids []uint64) ([]models.ApplicationCredentials, error) {
	query := sq.Select("id", "application_id", "platform", "username", "password", "created_at", "updated_at").
		From("application_credentials").
		Where(sq.Eq{"id": ids}).
		PlaceholderFormat(sq.Question)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("query build failed: %w", err)
	}

	rows, err := r.db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var credentials []models.ApplicationCredentials
	if err := scan.Rows(&credentials, rows); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return credentials, nil
}
