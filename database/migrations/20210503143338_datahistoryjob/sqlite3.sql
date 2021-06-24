-- +goose Up
CREATE TABLE IF NOT EXISTS datahistoryjob
(
    id text not null primary key,
    nickname TEXT NOT NULL,
    exchange_name_id uuid REFERENCES exchange(id) NOT NULL,
    asset TEXT NOT NULL,
    base TEXT NOT NULL,
    quote TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    interval REAL NOT NULL,
    data_type REAL NOT NULL,
    request_size REAL NOT NULL,
    max_retries REAL NOT NULL,
    status REAL NOT NULL,
    created TIMESTAMP NOT NULL,

    CONSTRAINT uniquenickname
        unique(nickname) ON CONFLICT REPLACE,
    CONSTRAINT uniquejob
        unique(exchange_name_id, asset, base, quote, start_time, end_time, interval, data_type, request_size) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS datahistoryjobresult
(
    id text not null primary key,
    job_id uuid REFERENCES datahistoryjob(id) NOT NULL,
    result TEXT NULL,
    status REAL NOT NULL,
    interval_start_time TIMESTAMP NOT NULL,
    interval_end_time TIMESTAMP NOT NULL,
    run_time TIMESTAMP NOT NULL,
    CONSTRAINT uniquejobtimestamp
        unique(job_id, interval_start_time, interval_end_time) ON CONFLICT REPLACE
);
-- +goose Down
DROP TABLE datahistoryjob;
DROP TABLE datahistoryjobresult;

