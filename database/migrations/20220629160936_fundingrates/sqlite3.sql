-- +goose Up
CREATE TABLE funding_rate
(
    id text NOT NULL primary key,
    exchange_name_id text NOT NULL,
    asset text NOT NULL,
    currency text NOT NULL,
    rate real NOT NULL,
    datetime timestamp NOT NULL,
    created timestamp NOT NULL default CURRENT_TIMESTAMP,
    source_job_id TEXT REFERENCES datahistoryjob(id),
    validation_job_id TEXT REFERENCES datahistoryjob(id),
    validation_issues TEXT
    FOREIGN KEY(exchange_name_id) REFERENCES exchange(id) ON DELETE RESTRICT,
    UNIQUE(id) ON CONFLICT REPLACE,
);


-- +goose Down
DROP TABLE fundingrate;

