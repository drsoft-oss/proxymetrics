package duckdb

const schemaSQL = `
CREATE TABLE IF NOT EXISTS profiles (
  id                    TEXT PRIMARY KEY,
  label                 TEXT NOT NULL,
  vendor                TEXT NOT NULL,
  type                  TEXT NOT NULL,
  region                TEXT,
  upstream_url          TEXT NOT NULL,
  price_per_gb          DOUBLE,
  price_per_gb_overage  DOUBLE,
  included_gb           DOUBLE,
  currency              TEXT NOT NULL DEFAULT 'USD',
  default_team          TEXT,
  default_project       TEXT,
  created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS events (
  ts                TIMESTAMP NOT NULL,
  request_id        TEXT      NOT NULL,
  profile_id        TEXT      NOT NULL,
  vendor            TEXT      NOT NULL,
  type              TEXT      NOT NULL,
  region            TEXT,
  target_host       TEXT,
  target_path_hash  TEXT,
  status_code       INTEGER,
  status_class      TEXT      NOT NULL,
  bytes_in          BIGINT    NOT NULL,
  bytes_out         BIGINT    NOT NULL,
  latency_ms        INTEGER   NOT NULL,
  cost_usd          DOUBLE    NOT NULL,
  team              TEXT,
  project           TEXT
);

CREATE TABLE IF NOT EXISTS rollups_1min (
  ts_bucket    TIMESTAMP NOT NULL,
  profile_id   TEXT      NOT NULL,
  vendor       TEXT      NOT NULL,
  type         TEXT      NOT NULL,
  region       TEXT,
  status_class TEXT      NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   BIGINT NOT NULL,
  bytes_in_total  BIGINT NOT NULL,
  bytes_out_total BIGINT NOT NULL,
  latency_ms_avg  DOUBLE NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  DOUBLE NOT NULL,
  success_count   BIGINT NOT NULL,
  failure_count   BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS rollups_1hour (
  ts_bucket    TIMESTAMP NOT NULL,
  profile_id   TEXT      NOT NULL,
  vendor       TEXT      NOT NULL,
  type         TEXT      NOT NULL,
  region       TEXT,
  status_class TEXT      NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   BIGINT NOT NULL,
  bytes_in_total  BIGINT NOT NULL,
  bytes_out_total BIGINT NOT NULL,
  latency_ms_avg  DOUBLE NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  DOUBLE NOT NULL,
  success_count   BIGINT NOT NULL,
  failure_count   BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS rollups_1day (
  ts_bucket    DATE      NOT NULL,
  profile_id   TEXT      NOT NULL,
  vendor       TEXT      NOT NULL,
  type         TEXT      NOT NULL,
  region       TEXT,
  status_class TEXT      NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   BIGINT NOT NULL,
  bytes_in_total  BIGINT NOT NULL,
  bytes_out_total BIGINT NOT NULL,
  latency_ms_avg  DOUBLE NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  DOUBLE NOT NULL,
  success_count   BIGINT NOT NULL,
  failure_count   BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_runs (
  id                   VARCHAR PRIMARY KEY,
  started_at           TIMESTAMP NOT NULL,
  finished_at          TIMESTAMP,
  status               VARCHAR NOT NULL,
  proxy_url            VARCHAR NOT NULL,
  provider             VARCHAR,
  expected_country     VARCHAR,
  expected_state      VARCHAR,
  expected_city        VARCHAR,
  expected_lat         DOUBLE NOT NULL,
  expected_lon         DOUBLE NOT NULL,
  expected_type        VARCHAR NOT NULL,
  check_level          VARCHAR NOT NULL,
  request_count        INTEGER NOT NULL,
  completed_count      INTEGER NOT NULL DEFAULT 0,
  location_match_count INTEGER NOT NULL DEFAULT 0,
  type_match_count     INTEGER NOT NULL DEFAULT 0,
  error_count          INTEGER NOT NULL DEFAULT 0,
  latency_p50_ms       INTEGER,
  latency_p95_ms       INTEGER,
  fallback_used        BOOLEAN NOT NULL DEFAULT FALSE,
  error                VARCHAR,
  session_key          VARCHAR,
  unique_ip_count      INTEGER
);

CREATE TABLE IF NOT EXISTS audit_requests (
  run_id           VARCHAR NOT NULL,
  seq              INTEGER NOT NULL,
  started_at       TIMESTAMP NOT NULL,
  duration_ms      INTEGER NOT NULL,
  observed_ip      VARCHAR,
  observed_country VARCHAR,
  observed_state   VARCHAR,
  observed_city    VARCHAR,
  observed_lat     DOUBLE,
  observed_lon     DOUBLE,
  is_datacenter    BOOLEAN,
  is_mobile        BOOLEAN,
  is_proxy         BOOLEAN,
  is_vpn           BOOLEAN,
  asn              INTEGER,
  company          VARCHAR,
  location_match   BOOLEAN,
  type_match       BOOLEAN,
  error            VARCHAR,
  geo_source       VARCHAR NOT NULL,
  attempts         INTEGER DEFAULT 1,
  PRIMARY KEY (run_id, seq)
);

CREATE INDEX IF NOT EXISTS audit_requests_run_id_idx ON audit_requests(run_id);

CREATE TABLE IF NOT EXISTS geo_city_centroids (
  country_code VARCHAR NOT NULL,
  state_norm   VARCHAR NOT NULL,
  city_norm    VARCHAR NOT NULL,
  lat          DOUBLE NOT NULL,
  lon          DOUBLE NOT NULL,
  resolved_at  TIMESTAMP NOT NULL,
  PRIMARY KEY (country_code, state_norm, city_norm)
);
`
