-- +goose Up
CREATE TABLE IF NOT EXISTS trade
(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_id uuid REFERENCES script(id),
    currency varchar NOT NULL,
    asset varchar NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    side varchar NOT NULL,
    timestamp bigint NOT NULL
);
-- +goose Down
DROP TABLE trade;