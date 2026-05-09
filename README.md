# ProxyMetrics

Know what's actually happening on your proxy stack. Self-hosted MITM router + dashboard
that records every request — status, latency, bytes, target, exit IP geo, and cost — 
across every vendor you use, then rolls it up so you can compare providers, audit pool 
quality, debug failures, and attribute spend

If you're running scrapers across any mix of residential / ISP / datacenter
pools, this is for you. You already know the problem: every proxy vendor has
its own siloed dashboard, the numbers don't agree, and at the end of the month
you still can't answer:

- *Which vendor is the most expensive per successful request on `amazon.com`?*
- *How many GB did I burn on 403s and 429s last week?*
- *Which team's scraper caused the $300 overage?*

ProxyMetrics measures every byte that flows through it, tags each request, and
puts the answers in a dashboard you can self-host on a $5 VPS.

---

## Why it exists

You can't optimize what you can't measure, and proxy spend is one of the
biggest line items in any scraping operation. Vendor dashboards won't tell you
which provider is wasting your budget on 403s, which "residential" pool is
actually datacenter, or which target host costs the most per successful
request. ProxyMetrics does.

It's one Go binary + an embedded React dashboard. You point your scraper at
it, it forwards traffic to your upstream proxy, and it records:

- Bytes in / bytes out, per request
- HTTP status code (200 / 301 / 403 / 429 / 5xx / timeout)
- Latency (p50/p95/p99)
- Target host
- Provider, proxy type, price-per-GB — lifted out of the upstream username
- Team / project tags — for chargeback

Then it rolls those into 1-minute, 1-hour, and 1-day buckets in SQLite and serves
them to a dashboard that leads with the chart that matters: **dollars wasted on
non-2xx responses, by provider, this billing cycle.**

---

## Features

- **MITM proxy router** (`:8080`) — handles HTTPS via a generated CA, forwards
  to any upstream HTTP/HTTPS proxy
- **Admin API + dashboard** (`:8081`) — embedded Vite/React SPA, no external web
  server required
- **Cost attribution** — multiplies measured bytes by per-vendor `price-per-GB`,
  rolls up by provider, target, status, team, project
- **Wasted-spend view** — top 5 wastiest providers and target hosts, headline KPI
  on the overview page
- **Live traffic** — server-sent events stream of in-flight requests
- **Audit module** — one-click "send N requests through this upstream and tell me
  what comes out": IP geolocation cross-check (ipapi.is + MaxMind GeoLite2),
  ASN/datacenter/VPN flags, latency percentiles, location-match verdict, shareable
  per-run report
- **Profile registry** — observed providers/types are auto-registered as you send
  traffic; no upfront config required
- **Pricing engine** — per-profile `price_per_gb` overrides, plus inline
  `price-<cents>` credtag in the upstream username
- **Sticky sessions** — preserve session keys end-to-end so vendor session
  affinity isn't broken
- **SQLite storage** — embedded, single-file, zero external services. Rollups run
  in-process on a 1-minute tick.
- **Cross-platform** — Linux/macOS, amd64 + arm64

---

## Quickstart (5 minutes)

You need a real upstream proxy URL from your vendor.

### Install

```bash
brew install drsoft-oss/tap/proxymetrics
```

Or grab a prebuilt binary for your OS/arch from the [latest GitHub
release](https://github.com/drsoft-oss/proxymetrics/releases/latest) and put it
on your `$PATH`.

### Or run with Docker

If you'd rather skip installing the binary, run the prebuilt image from Docker
Hub: [`drsoft/proxymetrics`](https://hub.docker.com/r/drsoft/proxymetrics).
Multi-arch (`linux/amd64`, `linux/arm64`). Tags: `latest` plus `vX.Y.Z` per
release.

```bash
mkdir -p ./data
curl -fsSL https://raw.githubusercontent.com/drsoft-oss/proxymetrics/main/config.example.yaml \
  -o ./data/config.yaml
export PROXYMETRICS_SECRET=$(openssl rand -hex 16)

docker run -d \
  --name proxymetrics \
  -p 8080:8080 -p 8081:8081 \
  -v "$(pwd)/data:/data" \
  -e PROXYMETRICS_SECRET \
  drsoft/proxymetrics:latest
```

The `/data` volume holds the SQLite database, the generated MITM CA, and your
`config.yaml`. The container runs as a non-root UID (`65532`); make sure the
host directory is writable by it. If you went the Docker route, skip **Run the
server** below and continue at **Trust the CA**.

### Run the server

The binary needs a config file — by default `./config.yaml` (override with
`-c <path>` or `$PROXYMETRICS_CONFIG`), and it exits if that path is missing.
`config.example.yaml` in this repo is the template; copy it to `config.yaml`
and edit.

```bash
curl -fsSL https://raw.githubusercontent.com/drsoft-oss/proxymetrics/main/config.example.yaml -o config.yaml
export PROXYMETRICS_SECRET=$(openssl rand -hex 16)

proxymetrics serve -c config.yaml
```

`serve` generates the MITM CA on first run and stores it under `./data/`.
That's the server. Now wire a client up.

### 1. Trust the CA

ProxyMetrics terminates TLS to measure bytes and status codes, so your client
needs to trust its CA. Either install it system-wide, or pass it per-request:

```bash
curl http://localhost:8081/cacert -o proxymetrics-ca.crt
```

Open the dashboard at `http://localhost:8081` — the **How to connect** dialog has
a fingerprint to verify and copy-paste snippets in curl, Python, Node.js, Go, and
Ruby.

### 2. Build the proxy URL

The proxy URL is:

```
http://<base64url(upstream_url)>:<deployment-secret>@<router-host>:8080
```

The username slot is your **real upstream proxy URL** (with vendor credentials),
base64url-encoded. The password is `PROXYMETRICS_SECRET`.

Inside that upstream username you can drop three reserved tags so ProxyMetrics
can attribute cost. Everything else (customer, zone, country, session, …) is
forwarded to the vendor verbatim:

| Tag                              | What it means                                  |
|----------------------------------|------------------------------------------------|
| `provider-<name>`                | `anonymous`, `iproyal`, `smartproxy`, … |
| `type-<residential\|isp\|datacenter\|mobile>` | The pool type                       |
| `price-<cents-per-gb>`           | Integer cents/GB. `price-400` = $4.00/GB     |

Example upstream URL (residential, US, $4/GB):

```
http://customer-acme-zone-residential-country-us-provider-anonymous-type-residential-price-400:upstream_password@rotating.dnsproxifier.com:31230
```

### 3. Send a request

```bash
UPSTREAM='http://customer-acme-zone-residential-country-us-provider-anonymous-type-residential-price-400:upstream_password@rotating.dnsproxifier.com:31230'
PROXY_USER=$(printf %s "$UPSTREAM" | base64 | tr -d '=' | tr '+/' '-_')

curl --cacert ./proxymetrics-ca.crt \
  -x "http://$PROXY_USER:$PROXYMETRICS_SECRET@localhost:8080" \
  https://api.infoip.io/
```

Open `http://localhost:8081` — your request is now in the overview, the live
traffic stream, the providers page, and the per-target table.

To tail events from the CLI instead:

```bash
proxymetrics events tail -c config.yaml
```

---

## How it works

```
                    ┌─────────────────────────────────────────────┐
                    │              ProxyMetrics binary            │
   ┌─────────┐      │  ┌──────────┐   ┌──────────┐   ┌─────────┐  │      ┌─────────────┐
   │ scraper │ ─────┼─▶│  proxy   │──▶│ writer + │──▶│ SQLite  │  │      │             │
   │ (curl,  │      │  │  :8080   │   │ rollups  │   │  file   │  │      │  upstream   │
   │ Scrapy, │      │  │ (MITM)   │──┼┐          │   └─────────┘  │      │   proxy     │
   │ etc.)   │      │  └──────────┘  ││          │        ▲       │      │             │
   └─────────┘      │                ││          │        │       │      └──────┬──────┘
                    │  ┌──────────┐  ││  ┌───────┴──────┐ │       │             │
        dashboard ──┼─▶│ admin    │──┼┴─▶│  embedded    │─┘       │             │
        (browser)   │  │  :8081   │  │   │  React SPA   │         │             │
                    │  │ + REST   │  │   └──────────────┘         │             │
                    │  └──────────┘  │                            │             │
                    │                └────────────────────────────┼─────────────┘
                    └─────────────────────────────────────────────┘
                                                                        (forwarded)
```

1. The scraper sends a request to the MITM proxy on `:8080`.
2. ProxyMetrics decodes the username, extracts `provider`/`type`/`price` credtags,
   and forwards the request to the upstream URL with those tags **stripped** —
   the vendor never sees them.
3. As bytes flow back, the proxy counts them and emits an event with status code,
   latency, byte counts, target host, profile, team, project, and computed cost.
4. A buffered writer batches events into SQLite. A rollup scheduler tickers every
   minute, materializing 1-min/1-hour/1-day aggregates.
5. The dashboard reads from rollup tables for charts and drills down to raw
   `events` rows on click.

### Storage layout

- `data/events.db` — single embedded file. Contains raw events, three rollup
  tables, profiles, audit runs and per-request rows.
- `data/ca.{pem,der,key}` — generated MITM CA. Rotate with
  `proxymetrics cacert rotate`.
- `data/maxmind/` *(optional)* — drop GeoLite2 City + Connection-Type DBs here to
  enable the MaxMind fallback for audits when ipapi.is is unreachable.
- `config.yaml` — listen ports, retention windows, batch sizes, default tags.

Nothing else. No Postgres, no Redis, no Prometheus required.

---

## Using the dashboard

Open `http://localhost:8081`.

| Page             | What you'll find                                                                   |
|------------------|------------------------------------------------------------------------------------|
| **Overview**     | KPI strip (total spend, wasted spend, success rate), spend trend, top wastiest providers and targets, status-code donut |
| **Providers**    | Per-vendor table: requests, GB, $ spent, $ wasted, success %                       |
| **Targets**      | Same view broken out by target host                                                |
| **Status codes** | Distribution + drill-down by code: how much each 403/429/5xx is costing you        |
| **Live traffic** | Real-time stream of every request hitting the proxy                                |
| **Profiles**     | Auto-registered (provider, type) combos with their inferred price                  |
| **Audits**       | Launch a new audit, view history, drill into per-request results                   |
| **Settings**     | CA fingerprint, deployment secret, version info                                    |

Every page accepts a global filter (time range, provider, type, team, project)
that scopes everything below it.

### The audit module

Click **New audit**, paste an upstream URL, set the expected country/state/city
and proxy type, choose a request count (default 200), and run.

ProxyMetrics fans out N requests through that upstream, geolocates each exit IP
via ipapi.is (with MaxMind GeoLite2 as fallback), and returns:

- % requests that landed in the expected country / state / city
- % residential vs datacenter ASNs (catches "residential" pools that are actually
  DC)
- p50/p95 latency
- Unique IP count (pool diversity)
- Per-request table with observed IP, geo, ASN, datacenter/VPN flags

Audit reports persist to SQLite; share one by linking to the run ID.

---

## CLI reference

```
proxymetrics serve               # run the proxy + admin servers
proxymetrics cacert init         # generate the MITM CA on first run
proxymetrics cacert show         # print PEM, fingerprint, validity
proxymetrics cacert path         # print absolute paths to CA files
proxymetrics cacert rotate       # archive old CA, generate a new one
proxymetrics profile list        # tabular view of observed profiles
proxymetrics profile test <id>   # send one IP-check request via this profile
proxymetrics events tail -n 100  # tail recent rows from SQLite
proxymetrics db ...              # low-level DB inspection
proxymetrics version             # print version, git commit, build date
proxymetrics config check        # validate config.yaml
```

All commands accept `-c <path>` for the config file. The default is
`./config.yaml`, overridable with `PROXYMETRICS_CONFIG`.

---

## Configuration

A config file is required at the path passed to `-c` (default `./config.yaml`);
`config.example.yaml` in this repo is the template — copy it to `config.yaml`
and edit. The fields you'll touch:

```yaml
server:
  proxy_listen: ":8080"          # client-facing MITM port
  api_listen:   ":8081"          # admin/dashboard port
  deployment_secret: "${PROXYMETRICS_SECRET}"

storage:
  data_dir: "./data"             # SQLite file + CA + maxmind dir

events:
  channel_size: 10000            # in-memory event buffer
  batch_size: 500                # flush size
  flush_interval: "1s"           # max time between flushes

defaults:
  team: "default"                # if not set per-request via tags
  project: "default"
```

Provide the deployment secret via env var, not the file:

```bash
export PROXYMETRICS_SECRET=$(openssl rand -hex 16)
```

---

## Development

```bash
make dev          # vite + air, hot reload both halves
make test         # Go unit tests
make test-ui      # vitest
make test-integration   # end-to-end (build tag: integration)
make bench        # proxy hot-path benchmarks
make lint         # golangci-lint
```

The repo layout:

```
cmd/proxymetrics/    main.go (cobra entrypoint)
internal/
  api/               REST handlers (overview, providers, targets, …)
  audit/             audit module: manager, runner, geo verdict, store
  proxy/             MITM core, CA, credtag parsing
  profile/           upstream profile registry + HTTP CRUD
  store/             SQLite store, schema, queries
  rollup/            1-min/1-hour/1-day aggregator
  events/            buffered writer
  geo/               ipapi.is + MaxMind cascade
  cli/               cobra subcommands
ui/                  Vite + React + TanStack + shadcn dashboard
docs/superpowers/    design specs and implementation plans for each feature
```

UI builds copy into `internal/ui/ui-dist/` and are embedded via `//go:embed` at
`go build` time. One artifact, no Node on the deploy host.

---

## Troubleshooting

**"x509: unknown authority" from my client.** Your client doesn't trust the CA.
Either pass the CA file per-request (`--cacert proxymetrics-ca.crt`) or install
it into your OS / language trust store. The fingerprint is on the dashboard's
Settings page.

**`407 Proxy Authentication Required`.** The password slot must be exactly your
`PROXYMETRICS_SECRET`. Anyone hitting the proxy without it is rejected — that's
what stops randos from using your router as a free open proxy.

**No events showing up.** Check `proxymetrics events tail` — if the CLI sees them
but the dashboard doesn't, refresh the time-range filter. New traffic lands in
the 1-minute rollup on the next tick.

**Audit returns "ipapi.is unreachable".** Configure the MaxMind fallback by
dropping `GeoLite2-City.mmdb` and `GeoLite2-Connection-Type.mmdb` into
`data/maxmind/` and pointing `geo.maxmind.city_db` at them in `config.yaml`.

---

## License

MIT — see [`LICENSE`](LICENSE).

---

## Status

Active development. The core proxy + storage, REST aggregations, dashboard
shell, data viewer pages, profiles, credtags, and audit module are all shipped.
Packaging is in place: Homebrew tap, prebuilt binaries on GitHub Releases, and
the [`drsoft/proxymetrics`](https://hub.docker.com/r/drsoft/proxymetrics)
Docker image (multi-arch).

Issues and PRs welcome. If you're using ProxyMetrics in production, open an issue
and tell us — it shapes what ships next.
