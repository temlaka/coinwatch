package schedule_test

import (
	"errors"
	"testing"
	"time"

	"github.com/temlaka/coinwatch/internal/domain"
	"github.com/temlaka/coinwatch/internal/repository/memory"
	"github.com/temlaka/coinwatch/internal/usecase/schedule"
)

var errUpdate = errors.New("coingecko is down")

type fakeUpdater struct {
	list    []domain.Crypto
	listErr error
	fail    map[string]bool
	updated []string
}

var _ schedule.PriceUpdater = (*fakeUpdater)(nil)

func (f *fakeUpdater) GetListCrypto() ([]domain.Crypto, error) {
	return f.list, f.listErr
}

func (f *fakeUpdater) UpdateCrypto(symbol string) (domain.Crypto, error) {
	if f.fail[symbol] {
		return domain.Crypto{}, errUpdate
	}
	f.updated = append(f.updated, symbol)
	return domain.Crypto{Symbol: symbol}, nil
}

func cryptos(symbols ...string) []domain.Crypto {
	list := make([]domain.Crypto, 0, len(symbols))
	for _, s := range symbols {
		list = append(list, domain.Crypto{Symbol: s})
	}
	return list
}

func TestEditSettings_Interval(t *testing.T) {
	cases := []struct {
		name     string
		interval int
		wantErr  bool
	}{
		{"below min", 9, true},
		{"min", 10, false},
		{"max", 3600, false},
		{"above max", 3601, true},
		{"negative", -1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := memory.NewScheduleRepo()
			svc := schedule.NewService(repo, &fakeUpdater{})

			err := svc.EditSettings(false, tc.interval)
			if tc.wantErr {
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("err = %v, want ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("EditSettings: %v", err)
			}

			got, _ := repo.Get()
			if got.Enabled || got.IntervalSeconds != tc.interval {
				t.Errorf("saved enabled=%v interval=%d, want false/%d", got.Enabled, got.IntervalSeconds, tc.interval)
			}
			wantNext := time.Now().Add(time.Duration(tc.interval) * time.Second)
			if d := got.NextUpdate.Sub(wantNext); d < -time.Second || d > time.Second {
				t.Errorf("NextUpdate = %v, want about %v", got.NextUpdate, wantNext)
			}
		})
	}
}

func TestEditSettings_NotifiesScheduler(t *testing.T) {
	svc := schedule.NewService(memory.NewScheduleRepo(), &fakeUpdater{})

	for range 3 {
		if err := svc.EditSettings(true, 60); err != nil {
			t.Fatalf("EditSettings: %v", err)
		}
	}

	select {
	case <-svc.Changes():
	default:
		t.Fatal("no change notification")
	}
	select {
	case <-svc.Changes():
		t.Fatal("notifications must be coalesced into one")
	default:
	}
}

func TestTrigger_PartialFailure(t *testing.T) {
	repo := memory.NewScheduleRepo()
	past := time.Now().Add(-time.Hour).UTC()
	_ = repo.Edit(true, 30, past, past)

	upd := &fakeUpdater{list: cryptos("BTC", "ETH", "SOL"), fail: map[string]bool{"ETH": true}}
	svc := schedule.NewService(repo, upd)

	count, ts, err := svc.Trigger()
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if ts.IsZero() {
		t.Error("timestamp is zero")
	}
	if len(upd.updated) != 2 || upd.updated[0] != "BTC" || upd.updated[1] != "SOL" {
		t.Errorf("updated = %v, want [BTC SOL]", upd.updated)
	}

	got, _ := repo.Get()
	if !got.LastUpdate.After(past) {
		t.Errorf("LastUpdate was not moved forward: %v", got.LastUpdate)
	}
}

func TestTrigger_AllFailed(t *testing.T) {
	repo := memory.NewScheduleRepo()
	past := time.Now().Add(-time.Hour).UTC()
	_ = repo.Edit(true, 30, past, past)

	upd := &fakeUpdater{list: cryptos("BTC", "ETH"), fail: map[string]bool{"BTC": true, "ETH": true}}
	svc := schedule.NewService(repo, upd)

	count, _, err := svc.Trigger()
	if !errors.Is(err, errUpdate) {
		t.Fatalf("err = %v, want errUpdate", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	got, _ := repo.Get()
	if !got.LastUpdate.Equal(past) {
		t.Errorf("LastUpdate changed on failure: %v", got.LastUpdate)
	}
	if !got.NextUpdate.After(time.Now()) {
		t.Errorf("NextUpdate was not postponed: %v", got.NextUpdate)
	}
}

func TestTrigger_Empty(t *testing.T) {
	svc := schedule.NewService(memory.NewScheduleRepo(), &fakeUpdater{})

	count, _, err := svc.Trigger()
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}
