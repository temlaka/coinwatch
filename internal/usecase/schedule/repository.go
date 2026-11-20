package schedule

import (
	"github.com/temlaka/coinwatch/internal/domain"
	"time"
)

type PriceUpdater interface {
	UpdateCrypto(symbol string) (domain.Crypto, error)
	GetListCrypto() ([]domain.Crypto, error)
}
type ScheduleRepository interface {
	Get() (domain.Schedule, error)
	Edit(enabled bool, interval int, lastUpdated time.Time, nextUpdate time.Time) error
	Update() error
}
