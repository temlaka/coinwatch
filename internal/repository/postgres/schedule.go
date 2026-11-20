package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/temlaka/coinwatch/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleRepository struct {
	pool *pgxpool.Pool
}

func NewScheduleRepo(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{
		pool: pool,
	}
}

func (sr *ScheduleRepository) Get() (domain.Schedule, error) {
	var s domain.Schedule

	err := sr.pool.QueryRow(context.Background(), "SELECT enabled, interval_seconds, last_update, next_update FROM schedule WHERE id = 1").Scan(&s.Enabled, &s.IntervalSeconds, &s.LastUpdate, &s.NextUpdate)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Schedule{}, fmt.Errorf("get schedule:no row: %w", err)
	}
	if err != nil {
		return domain.Schedule{}, fmt.Errorf("get schedule: %w", err)
	}
	s.LastUpdate = s.LastUpdate.UTC()
	s.NextUpdate = s.NextUpdate.UTC()
	return s, nil

}

func (sr *ScheduleRepository) Edit(enabled bool, interval int, lastUpdated time.Time, nextUpdate time.Time) error {

	r, err := sr.pool.Exec(context.Background(), "UPDATE schedule SET enabled = $1, interval_seconds = $2, last_update = $3, next_update = $4 WHERE id = 1", enabled, interval, lastUpdated, nextUpdate)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("update schedule not found: %w", domain.ErrRowsNotAffected)
	}
	return nil
}

func (sr *ScheduleRepository) Update() error {

	r, err := sr.pool.Exec(context.Background(), "UPDATE schedule SET last_update = now(), next_update = now() + interval '1 second' * interval_seconds WHERE id = 1")
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("update schedule not affected: %w", domain.ErrRowsNotAffected)
	}
	return nil
}
