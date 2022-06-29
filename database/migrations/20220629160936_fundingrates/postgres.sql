-- +goose Up
CREATE TABLE IF NOT EXISTS funding_rate
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_name_id uuid REFERENCES exchange(id) NOT NULL,
    asset varchar NOT NULL,
    currency varchar(30) NOT NULL,
    rate DOUBLE PRECISION NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    CONSTRAINT uniquefundingrateid
    unique(exchange_name_id, id),
    CONSTRAINT uniquefundingrate
    unique(exchange_name_id, currency, asset, timestamp)
    );
-- +goose Down
DROP TABLE funding_rate;