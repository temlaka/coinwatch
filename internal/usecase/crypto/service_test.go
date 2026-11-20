package crypto_test

import (
	"errors"
	"testing"
	"time"

	"github.com/temlaka/coinwatch/internal/domain"
	"github.com/temlaka/coinwatch/internal/repository/memory"
	"github.com/temlaka/coinwatch/internal/usecase/crypto"
)

type fakeProvider struct {
	name  string
	price float64
	err   error
	calls int
}

var _ crypto.PriceProvider = (*fakeProvider)(nil)

func (f *fakeProvider) GetPrice(symbol string) (string, float64, error) {
	f.calls++
	return f.name, f.price, f.err
}

func newService(p *fakeProvider) (*crypto.Service, *memory.CryptoRepo) {
	repo := memory.NewCryptoRepo(100)
	return crypto.NewService(repo, p), repo
}

func TestAddCrypto(t *testing.T) {
	p := &fakeProvider{name: "Bitcoin", price: 100}
	svc, repo := newService(p)

	got, err := svc.AddCrypto("btc")
	if err != nil {
		t.Fatalf("AddCrypto: %v", err)
	}
	if got.Symbol != "BTC" || got.Name != "Bitcoin" || got.CurrentPrice != 100 {
		t.Errorf("got %+v, want BTC/Bitcoin/100", got)
	}
	if got.LastUpdated.IsZero() {
		t.Error("LastUpdated is zero")
	}

	stored, err := repo.Get("BTC")
	if err != nil {
		t.Fatalf("repo.Get: %v", err)
	}
	if stored != got {
		t.Errorf("stored %+v, want %+v", stored, got)
	}
}

func TestAddCrypto_Duplicate(t *testing.T) {
	p := &fakeProvider{name: "Bitcoin", price: 100}
	svc, _ := newService(p)

	if _, err := svc.AddCrypto("BTC"); err != nil {
		t.Fatalf("first AddCrypto: %v", err)
	}
	_, err := svc.AddCrypto("btc")
	if !errors.Is(err, domain.ErrCryptoExists) {
		t.Fatalf("err = %v, want ErrCryptoExists", err)
	}
	if p.calls != 1 {
		t.Errorf("provider calls = %d, want 1", p.calls)
	}
}

func TestAddCrypto_ProviderError(t *testing.T) {
	p := &fakeProvider{err: domain.ErrUnknownSymbol}
	svc, repo := newService(p)

	_, err := svc.AddCrypto("NOPE")
	if !errors.Is(err, domain.ErrUnknownSymbol) {
		t.Fatalf("err = %v, want ErrUnknownSymbol", err)
	}
	if _, err := repo.Get("NOPE"); !errors.Is(err, domain.ErrCryptoNotFound) {
		t.Errorf("crypto was saved after provider error: %v", err)
	}
}

func TestUpdateCrypto(t *testing.T) {
	p := &fakeProvider{name: "Bitcoin", price: 100}
	svc, repo := newService(p)
	if _, err := svc.AddCrypto("BTC"); err != nil {
		t.Fatalf("AddCrypto: %v", err)
	}

	p.price = 120
	got, err := svc.UpdateCrypto("btc")
	if err != nil {
		t.Fatalf("UpdateCrypto: %v", err)
	}
	if got.CurrentPrice != 120 {
		t.Errorf("CurrentPrice = %v, want 120", got.CurrentPrice)
	}

	history, err := repo.History("BTC")
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 2 || history[0].Price != 100 || history[1].Price != 120 {
		t.Errorf("history = %+v, want prices [100 120]", history)
	}
}

func TestUpdateCrypto_NotFound(t *testing.T) {
	p := &fakeProvider{name: "Bitcoin", price: 100}
	svc, _ := newService(p)

	_, err := svc.UpdateCrypto("BTC")
	if !errors.Is(err, domain.ErrCryptoNotFound) {
		t.Fatalf("err = %v, want ErrCryptoNotFound", err)
	}
	if p.calls != 0 {
		t.Errorf("provider calls = %d, want 0", p.calls)
	}
}

func TestGetStats(t *testing.T) {
	cases := []struct {
		name   string
		prices []float64
		want   domain.Stats
	}{
		{
			name:   "single price",
			prices: []float64{100},
			want:   domain.Stats{MinPrice: 100, MaxPrice: 100, AvgPrice: 100, RecordsCount: 1},
		},
		{
			name:   "growth",
			prices: []float64{100, 50, 150},
			want: domain.Stats{
				MinPrice: 50, MaxPrice: 150, AvgPrice: 100,
				PriceChange: 50, PriceChangePercent: 50, RecordsCount: 3,
			},
		},
		{
			name:   "drop",
			prices: []float64{200, 100},
			want: domain.Stats{
				MinPrice: 100, MaxPrice: 200, AvgPrice: 150,
				PriceChange: -100, PriceChangePercent: -50, RecordsCount: 2,
			},
		},
		{
			name:   "first price zero",
			prices: []float64{0, 10},
			want: domain.Stats{
				MinPrice: 0, MaxPrice: 10, AvgPrice: 5,
				PriceChange: 10, PriceChangePercent: 0, RecordsCount: 2,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newService(&fakeProvider{})
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

			err := repo.Create(domain.Crypto{Symbol: "BTC", Name: "Bitcoin", CurrentPrice: tc.prices[0], LastUpdated: start})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			for i, price := range tc.prices[1:] {
				point := domain.PricePoint{Price: price, Timestamp: start.Add(time.Duration(i+1) * time.Minute)}
				if err := repo.UpdatePrice("BTC", point); err != nil {
					t.Fatalf("UpdatePrice: %v", err)
				}
			}

			current, stats, err := svc.GetStats("btc")
			if err != nil {
				t.Fatalf("GetStats: %v", err)
			}
			if want := tc.prices[len(tc.prices)-1]; current != want {
				t.Errorf("current = %v, want %v", current, want)
			}
			if stats != tc.want {
				t.Errorf("stats = %+v\nwant    %+v", stats, tc.want)
			}
		})
	}
}

func TestGetStats_NotFound(t *testing.T) {
	svc, _ := newService(&fakeProvider{})

	_, _, err := svc.GetStats("BTC")
	if !errors.Is(err, domain.ErrCryptoNotFound) {
		t.Fatalf("err = %v, want ErrCryptoNotFound", err)
	}
}
