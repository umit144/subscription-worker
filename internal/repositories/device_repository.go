package repositories

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/blockloop/scan/v2"
	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/models"
)

type DeviceRepository struct {
	db *database.Database
}

func NewDeviceRepository(db *database.Database) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) FetchByIds(ids []uint64) ([]models.Device, error) {
	query := sq.Select("id", "uid", "platform", "language", "created_at", "updated_at").
		From("devices").
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

	var devices []models.Device
	if err := scan.Rows(&devices, rows); err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return devices, nil
}
