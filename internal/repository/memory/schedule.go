package memory

import (
	"github.com/temlaka/coinwatch/internal/domain"
	"sync"
	"time"
)

type ScheduleRepository struct {
	storage domain.Schedule
	mu      sync.RWMutex
}

func NewScheduleRepo() *ScheduleRepository {

	return &ScheduleRepository{
		storage: domain.Schedule{
			Enabled:         true,
			IntervalSeconds: 30,
			LastUpdate:      time.Now().UTC(),
			NextUpdate:      time.Now().Add(30 * time.Second).UTC(),
		},
	}
}

func (sr *ScheduleRepository) Get() (domain.Schedule, error) {
	sr.mu.RLock()
	res := sr.storage
	sr.mu.RUnlock()
	return res, nil
}

func (sr *ScheduleRepository) Edit(enabled bool, interval int, lastUpdated time.Time, nextUpdate time.Time) error {
	sr.mu.Lock()
	x := domain.Schedule{
		Enabled:         enabled,
		IntervalSeconds: interval,
		LastUpdate:      lastUpdated,
		NextUpdate:      nextUpdate,
	}
	sr.storage = x
	sr.mu.Unlock()
	return nil
}

func (sr *ScheduleRepository) Update() error {
	sr.mu.Lock()
	sr.storage.LastUpdate = time.Now().UTC()
	sr.storage.NextUpdate = time.Now().Add(time.Duration(sr.storage.IntervalSeconds) * time.Second).UTC()
	sr.mu.Unlock()
	return nil
}
