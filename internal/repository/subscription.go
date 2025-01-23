package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/models"
)

type SubscriptionRepository struct {
	db *database.Database
}

func NewSubscriptionRepository(db *database.Database) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) GetExpiredSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	query := `
        SELECT 
            id,
            device_id,
            application_id,
            receipt,
            status,
            expire_date
        FROM subscriptions 
        WHERE expire_date < NOW() 
            AND status = 1
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var subscriptions []models.Subscription
	for rows.Next() {
		var s models.Subscription
		var expireDate []uint8
		err := rows.Scan(
			&s.ID,
			&s.DeviceID,
			&s.ApplicationID,
			&s.Receipt,
			&s.Status,
			&expireDate,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		s.ExpireDate, err = time.Parse("2006-01-02 15:04:05", string(expireDate))
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		subscriptions = append(subscriptions, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return subscriptions, nil
}
