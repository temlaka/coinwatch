package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/temlaka/coinwatch/internal/domain"
	"github.com/temlaka/coinwatch/internal/usecase/crypto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ crypto.CryptoRepository = (*CryptoRepo)(nil)

type CryptoRepo struct {
	pool         *pgxpool.Pool
	historyLimit int
}

func NewCryptoRepo(pool *pgxpool.Pool, limit int) *CryptoRepo {
	return &CryptoRepo{
		pool:         pool,
		historyLimit: limit,
	}
}

func (cr *CryptoRepo) Create(c domain.Crypto) error {
	tx, err := cr.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(context.Background(), "INSERT INTO cryptos (symbol, name, current_price, last_updated) VALUES ($1, $2, $3, $4)", c.Symbol, c.Name, c.CurrentPrice, c.LastUpdated); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("create %s: %w", c.Symbol, domain.ErrCryptoExists)
		}
		return fmt.Errorf("inesrt crypto: %w", err)
	}

	if _, err := tx.Exec(context.Background(), "INSERT INTO price_history (symbol, price, recorded_at) VALUES ($1, $2, $3)", c.Symbol, c.CurrentPrice, c.LastUpdated); err != nil {
		return fmt.Errorf("insert price history: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (cr *CryptoRepo) List() ([]domain.Crypto, error) {
	r, err := cr.pool.Query(context.Background(), "SELECT symbol,name,current_price,last_updated FROM cryptos ORDER BY symbol")

	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer r.Close()

	list := make([]domain.Crypto, 0)

	for r.Next() {
		var c domain.Crypto
		if err := r.Scan(&c.Symbol, &c.Name, &c.CurrentPrice, &c.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan crypto: %w", err)
		}
		c.LastUpdated = c.LastUpdated.UTC()
		list = append(list, c)
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return list, nil
}

func (cr *CryptoRepo) Get(symbol string) (domain.Crypto, error) {
	var getcrypto domain.Crypto
	err := cr.pool.QueryRow(context.Background(), "SELECT symbol, name, current_price, last_updated FROM cryptos WHERE symbol = $1", symbol).Scan(&getcrypto.Symbol, &getcrypto.Name, &getcrypto.CurrentPrice, &getcrypto.LastUpdated)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Crypto{}, fmt.Errorf("get %s: %w", symbol, domain.ErrCryptoNotFound)
	}
	if err != nil {
		return domain.Crypto{}, fmt.Errorf("get %s: %w", symbol, err)
	}
	getcrypto.LastUpdated = getcrypto.LastUpdated.UTC()
	return getcrypto, nil
}

func (cr *CryptoRepo) UpdatePrice(symbol string, p domain.PricePoint) error {
	tx, err := cr.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(context.Background())

	r, err := tx.Exec(context.Background(), "UPDATE cryptos SET current_price = $2, last_updated = $3 WHERE symbol = $1", symbol, p.Price, p.Timestamp)
	if err != nil {
		return fmt.Errorf("update price: %w", err)
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("update price %s: %w", symbol, domain.ErrCryptoNotFound)
	}

	_, err = tx.Exec(context.Background(), "INSERT INTO price_history (symbol, price, recorded_at) VALUES ($1, $2, $3)", symbol, p.Price, p.Timestamp)
	if err != nil {

		return fmt.Errorf("update crypto price history: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
func (cr *CryptoRepo) History(symbol string) ([]domain.PricePoint, error) {
	r, err := cr.pool.Query(context.Background(), "SELECT *	FROM (SELECT price, recorded_at FROM price_history WHERE symbol = $1 ORDER BY recorded_at DESC LIMIT $2 ) sub ORDER BY recorded_at ASC;", symbol, cr.historyLimit)

	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer r.Close()

	list := make([]domain.PricePoint, 0)

	for r.Next() {
		var c domain.PricePoint
		if err := r.Scan(&c.Price, &c.Timestamp); err != nil {
			return nil, fmt.Errorf("scan timestamp: %w", err)
		}
		c.Timestamp = c.Timestamp.UTC()
		list = append(list, c)
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("rows history: %w", err)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("history %s: %w", symbol, domain.ErrCryptoNotFound)
	}
	return list, nil
}
func (cr *CryptoRepo) GetWithHistory(symbol string) (domain.Crypto, []domain.PricePoint, error) {

	tx, err := cr.pool.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Crypto{}, nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(context.Background())

	getcrypto := domain.Crypto{}
	list := make([]domain.PricePoint, 0)

	if err := tx.QueryRow(context.Background(), "SELECT symbol, name, current_price, last_updated FROM cryptos WHERE symbol = $1", symbol).Scan(&getcrypto.Symbol, &getcrypto.Name, &getcrypto.CurrentPrice, &getcrypto.LastUpdated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Crypto{}, nil, fmt.Errorf("get %s: %w", symbol, domain.ErrCryptoNotFound)
		}
		return domain.Crypto{}, nil, fmt.Errorf("get with history %s: %w", symbol, err)
	}
	getcrypto.LastUpdated = getcrypto.LastUpdated.UTC()
	r, err := tx.Query(context.Background(), "SELECT *	FROM (SELECT price, recorded_at FROM price_history WHERE symbol = $1 ORDER BY recorded_at DESC LIMIT $2 ) sub ORDER BY recorded_at ASC;", symbol, cr.historyLimit)

	if err != nil {
		return domain.Crypto{}, nil, fmt.Errorf("list: %w", err)
	}
	defer r.Close()

	for r.Next() {
		var c domain.PricePoint
		if err := r.Scan(&c.Price, &c.Timestamp); err != nil {
			return domain.Crypto{}, nil, fmt.Errorf("scan timestamp: %w", err)
		}
		c.Timestamp = c.Timestamp.UTC()
		list = append(list, c)
	}
	if err := r.Err(); err != nil {
		return domain.Crypto{}, nil, fmt.Errorf("rows timestamp: %w", err)
	}
	if len(list) == 0 {
		return domain.Crypto{}, nil, fmt.Errorf("history %s: %w", symbol, domain.ErrCryptoNotFound)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return domain.Crypto{}, nil, fmt.Errorf("commit: %w", err)
	}
	return getcrypto, list, nil
}

func (cr *CryptoRepo) Delete(symbol string) error {
	r, err := cr.pool.Exec(context.Background(), "DELETE FROM cryptos WHERE symbol = $1", symbol)
	if err != nil {
		return fmt.Errorf("delete %s: %w", symbol, err)
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("delete %s: %w", symbol, domain.ErrCryptoNotFound)
	}
	return nil

}
