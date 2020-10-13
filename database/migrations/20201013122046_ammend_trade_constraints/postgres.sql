-- +goose Up
-- +goose StatementBegin
ALTER TABLE trade DROP CONSTRAINT uniquetrade;

CREATE UNIQUE INDEX unique_trade_no_id ON trade (base,quote,asset,price,amount,side, timestamp)
    WHERE tid IS NULL;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
ALTER TABLE trade DROP CONSTRAINT unique_trade_no_id;
-- +goose StatementEnd
