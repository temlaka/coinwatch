-- +goose Up
CREATE TABLE
    users (
        id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        username text NOT NULL UNIQUE,
        password_hash bytea NOT NULL
    );

CREATE TABLE
    cryptos (
        symbol text NOT NULL UNIQUE PRIMARY KEY,
        name text NOT NULL,
        current_price numeric(20, 8) NOT NULL,
        last_updated timestamptz NOT NULL DEFAULT now ()
    );

CREATE TABLE
    schedule (
        id int CHECK (id = 1) PRIMARY KEY,
        enabled boolean NOT NULL DEFAULT true,
        interval_seconds integer NOT NULL CHECK (interval_seconds BETWEEN 10 AND 3600),
        last_update timestamptz NOT NULL DEFAULT now (),
        next_update timestamptz NOT NULL DEFAULT now ()
    );

CREATE TABLE
    price_history (
        id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        symbol text NOT NULL REFERENCES cryptos (symbol) ON DELETE CASCADE,
        price numeric(20, 8) NOT NULL,
        recorded_at timestamptz NOT NULL DEFAULT now ()
    );

INSERT INTO
    schedule (id, enabled, interval_seconds)
VALUES
    (1, true, 30);

CREATE INDEX idx_price_history_symbol_recorded_at ON price_history (symbol, recorded_at DESC);


-- +goose Down
DROP TABLE IF EXISTS price_history,
cryptos,
users,
schedule;
