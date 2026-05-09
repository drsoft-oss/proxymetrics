package sqlite

// SQLite uses type affinity, not strict types. The column declarations below
// match the original DuckDB schema 1:1 in semantics; affinities map as:
//   TEXT/VARCHAR     -> TEXT
//   INTEGER/BIGINT   -> INTEGER (8-byte on SQLite anyway)
//   DOUBLE           -> REAL
//   BOOLEAN          -> INTEGER (0/1; modernc handles bool<->int)
//   TIMESTAMP/DATE   -> DATETIME (TEXT 'YYYY-MM-DD HH:MM:SS', UTC)
const schemaSQL = `
CREATE TABLE IF NOT EXISTS profiles (
  id                    TEXT PRIMARY KEY,
  label                 TEXT NOT NULL,
  vendor                TEXT NOT NULL,
  type                  TEXT NOT NULL,
  region                TEXT,
  upstream_url          TEXT NOT NULL,
  price_per_gb          REAL,
  price_per_gb_overage  REAL,
  included_gb           REAL,
  currency              TEXT NOT NULL DEFAULT 'USD',
  default_team          TEXT,
  default_project       TEXT,
  created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS events (
  ts                DATETIME NOT NULL,
  request_id        TEXT     NOT NULL,
  profile_id        TEXT     NOT NULL,
  vendor            TEXT     NOT NULL,
  type              TEXT     NOT NULL,
  region            TEXT,
  target_host       TEXT,
  target_path_hash  TEXT,
  status_code       INTEGER,
  status_class      TEXT     NOT NULL,
  bytes_in          INTEGER  NOT NULL,
  bytes_out         INTEGER  NOT NULL,
  latency_ms        INTEGER  NOT NULL,
  cost_usd          REAL     NOT NULL,
  team              TEXT,
  project           TEXT
);

CREATE TABLE IF NOT EXISTS rollups_1min (
  ts_bucket    DATETIME NOT NULL,
  profile_id   TEXT     NOT NULL,
  vendor       TEXT     NOT NULL,
  type         TEXT     NOT NULL,
  region       TEXT,
  status_class TEXT     NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   INTEGER NOT NULL,
  bytes_in_total  INTEGER NOT NULL,
  bytes_out_total INTEGER NOT NULL,
  latency_ms_avg  REAL    NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  REAL    NOT NULL,
  success_count   INTEGER NOT NULL,
  failure_count   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS rollups_1hour (
  ts_bucket    DATETIME NOT NULL,
  profile_id   TEXT     NOT NULL,
  vendor       TEXT     NOT NULL,
  type         TEXT     NOT NULL,
  region       TEXT,
  status_class TEXT     NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   INTEGER NOT NULL,
  bytes_in_total  INTEGER NOT NULL,
  bytes_out_total INTEGER NOT NULL,
  latency_ms_avg  REAL    NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  REAL    NOT NULL,
  success_count   INTEGER NOT NULL,
  failure_count   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS rollups_1day (
  ts_bucket    DATETIME NOT NULL,
  profile_id   TEXT     NOT NULL,
  vendor       TEXT     NOT NULL,
  type         TEXT     NOT NULL,
  region       TEXT,
  status_class TEXT     NOT NULL,
  target_host  TEXT,
  team         TEXT,
  project      TEXT,
  request_count   INTEGER NOT NULL,
  bytes_in_total  INTEGER NOT NULL,
  bytes_out_total INTEGER NOT NULL,
  latency_ms_avg  REAL    NOT NULL,
  latency_ms_p50  INTEGER,
  latency_ms_p95  INTEGER,
  latency_ms_p99  INTEGER,
  cost_usd_total  REAL    NOT NULL,
  success_count   INTEGER NOT NULL,
  failure_count   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_runs (
  id                   TEXT PRIMARY KEY,
  started_at           DATETIME NOT NULL,
  finished_at          DATETIME,
  status               TEXT NOT NULL,
  proxy_url            TEXT NOT NULL,
  provider             TEXT,
  expected_country     TEXT,
  expected_state       TEXT,
  expected_city        TEXT,
  expected_lat         REAL NOT NULL,
  expected_lon         REAL NOT NULL,
  expected_type        TEXT NOT NULL,
  check_level          TEXT NOT NULL,
  request_count        INTEGER NOT NULL,
  completed_count      INTEGER NOT NULL DEFAULT 0,
  location_match_count INTEGER NOT NULL DEFAULT 0,
  type_match_count     INTEGER NOT NULL DEFAULT 0,
  error_count          INTEGER NOT NULL DEFAULT 0,
  latency_p50_ms       INTEGER,
  latency_p95_ms       INTEGER,
  fallback_used        INTEGER NOT NULL DEFAULT 0,
  error                TEXT,
  session_key          TEXT,
  unique_ip_count      INTEGER
);

CREATE TABLE IF NOT EXISTS audit_requests (
  run_id           TEXT NOT NULL,
  seq              INTEGER NOT NULL,
  started_at       DATETIME NOT NULL,
  duration_ms      INTEGER NOT NULL,
  observed_ip      TEXT,
  observed_country TEXT,
  observed_state   TEXT,
  observed_city    TEXT,
  observed_lat     REAL,
  observed_lon     REAL,
  is_datacenter    INTEGER,
  is_mobile        INTEGER,
  is_proxy         INTEGER,
  is_vpn           INTEGER,
  asn              INTEGER,
  company          TEXT,
  location_match   INTEGER,
  type_match       INTEGER,
  error            TEXT,
  geo_source       TEXT NOT NULL,
  attempts         INTEGER DEFAULT 1,
  PRIMARY KEY (run_id, seq)
);

CREATE INDEX IF NOT EXISTS audit_requests_run_id_idx ON audit_requests(run_id);

CREATE TABLE IF NOT EXISTS geo_city_centroids (
  country_code TEXT NOT NULL,
  state_norm   TEXT NOT NULL,
  city_norm    TEXT NOT NULL,
  lat          REAL NOT NULL,
  lon          REAL NOT NULL,
  resolved_at  DATETIME NOT NULL,
  PRIMARY KEY (country_code, state_norm, city_norm)
);
`
