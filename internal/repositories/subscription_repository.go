package repositories

import (
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/blockloop/scan/v2"
	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/models"
)

type SubscriptionRepository struct {
	db *database.Database
}

func NewSubscriptionRepository(db *database.Database) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) GetExpiredSubscriptions() ([]models.Subscription, error) {
	query := sq.Select("id", "device_id", "application_id", "receipt", "status", "expire_date").
		From("subscriptions").
		Where("expire_date < NOW()").
		Where(sq.Eq{"status": 1}).
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

	var subscriptions []models.Subscription
	if err := scan.Rows(&subscriptions, rows); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return subscriptions, nil
}

func (r *SubscriptionRepository) UpdateExpireDate(id uint64, expireDate time.Time) error {
	query := sq.Update("subscriptions").
		Set("expire_date", expireDate).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Question)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("query build failed: %w", err)
	}

	_, err = r.db.Exec(sql, args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) UpdateStatus(id uint64, status bool) error {
	query := sq.Update("subscriptions").
		Set("status", status).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Question)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("query build failed: %w", err)
	}

	_, err = r.db.Exec(sql, args...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}
