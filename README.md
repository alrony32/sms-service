# SMS Service

A centralized, multi-tenant SMS platform that handles both single and bulk SMS
requests for many clients. It uses **Redis only** (no SQL) for queues, batch
tracking, delivery status, rate limiting and scheduling, and dispatches to the
SMS provider (Dhaka Colo) fairly across clients using a round-robin scheduler.

## Architecture

```
                 ┌──────────────┐
  HTTP (Gin)     │  API Key MW  │
  /sms/send  ───►│  Handlers    │
  /sms/bulk      │  Service     │── batch_id, per-msg status
                 └──────┬───────┘
                        │ RPUSH
                        ▼
        Redis   ss-db:{client}  (per-client SMS queue)   ss:clients (set)
                        │
       round-robin      │ LRANGE+LTRIM (≤ BatchSize per client per cycle)
                        ▼
                 ┌──────────────┐   provider rate limit (per minute)
                 │ Dispatcher   │── SendBatch ─► Driver (log | dhakacolo HTTP)
                 └──────┬───────┘
                        │ RPUSH delivery report
                        ▼
        Redis   ss-webhook:{client}  (per-client webhook queue)
                        │
   per-client rate cap  │ (e.g. 30/min)
                        ▼
                 ┌──────────────┐
                 │ Webhook      │── POST ─► client webhook_url
                 │ Worker       │
                 └──────────────┘
```

### Redis key layout

| Purpose | Key | Type |
|---|---|---|
| SMS queue per client | `ss-db:{client}` | LIST |
| Webhook queue per client | `ss-webhook:{client}` | LIST |
| Active clients (round-robin) | `ss:clients` | SET |
| Batch metadata | `ss:batch:{batch_id}` | HASH |
| Per-message status | `ss:msg:{client}:{id}` | HASH |
| Provider rate-limit window | `ss:rl:provider` | STRING + TTL |
| Webhook rate-limit window | `ss:rl:webhook:{client}` | STRING + TTL |

### Fairness

If messages were processed FIFO from a single queue, a client submitting 10,000
SMS would starve a client submitting 50. The dispatcher instead iterates the
client set each cycle and takes **up to `DHAKACOLO_BATCH_SIZE` (default 50)**
messages per client per pass — the same cap as the provider's per-request limit —
so every client makes progress on every cycle.

### Durability

The app issues a best-effort `CONFIG SET appendonly yes` at startup, but the real
guarantee belongs in `redis.conf`. **Run Redis with `appendonly yes` (AOF) and/or
RDB snapshots in production** so queued (especially single) SMS are never lost.

> Note: dequeuing uses `LRANGE` + `LTRIM` in a transaction rather than `LPOP count`,
> so it works on Redis < 6.2 as well.

## API

All endpoints require the `X-API-KEY` header (value from `API_KEY`).

### Single SMS — `POST /api/v1/sms/send`

```json
{
  "client": "dev",
  "from": "9998771827",
  "webhook_url": "https://apidev.smartcomm.ai/sms/webhook",
  "id": "msg-001",
  "to": "01XXXXXXXXX",
  "message": "Hello World"
}
```

### Bulk SMS — `POST /api/v1/sms/bulk`

```json
{
  "client": "dev",
  "from": "9998771827",
  "webhook_url": "https://apidev.smartcomm.ai/sms/webhook",
  "messages": [
    { "id": "msg-001", "to": "8801XXXXXXXXX", "message": "Hello World" },
    { "id": "msg-002", "to": "8801XXXXXXXXX", "message": "Hello World" }
  ]
}
```

Both return `202 Accepted`:

```json
{ "status": "queued", "batch_id": "ae84a8c7c11562ec48f3ab875d77ce14", "count": 2 }
```

### Batch tracking — `GET /api/v1/sms/batch/:batch_id`

```json
{
  "batch_id": "ae84...", "client": "abc", "total": "2",
  "queued": "0", "sent": "2", "failed": "0", "delivered": "2",
  "created_at": "2026-06-13T04:52:25Z"
}
```

### Queue sizes (ops) — `GET /api/v1/sms/queues`

```json
{ "ss-db:dev": 0, "ss-webhook:dev": 0, "ss-db:abc": 0, "ss-webhook:abc": 0 }
```

Phone numbers (`01XXXXXXXXX`, `8801XXXXXXXXX`, `+8801XXXXXXXXX`) are validated and
normalized to `+8801XXXXXXXXX`.

## Configuration

Copy `.env.example` to `.env` and adjust. Key variables:

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP port |
| `REDIS_HOST` / `REDIS_PORT` | `127.0.0.1` / `6379` | Redis address |
| `REDIS_PASSWORD` / `REDIS_DB` | _empty_ / `0` | Redis auth / db index |
| `API_KEY` | — | Required `X-API-KEY` value |
| `SMS_DRIVER` | `log` | `log` or `dhakacolo` |
| `DHAKACOLO_URL` / `DHAKACOLO_API_KEY` / `DHAKACOLO_SENDER` | — | Provider HTTP settings |
| `DHAKACOLO_BATCH_SIZE` | `50` | Max messages per provider request / per round-robin slice |
| `DHAKACOLO_RATE_PER_MIN` | `60` | Provider requests per minute |
| `WEBHOOK_RATE_PER_MIN` | `30` | Webhook deliveries per minute, per client |
| `WEBHOOK_TIMEOUT_SEC` | `10` | Webhook POST timeout |
| `WEBHOOK_MAX_RETRIES` | `5` | Bounded re-queue attempts on failure |
| `SCHEDULER_INTERVAL_MS` | `1000` | Idle sleep when queues are empty |

> The DhakaColo HTTP payload/auth shape in `internal/driver/dhakacolo_driver.go`
> (Bearer token, `{sender, recipients[]}` body, 2xx = accepted) is a sensible
> default — adjust it to match the actual provider contract.

## Running

```bash
cp .env.example .env        # then set API_KEY etc.
redis-server --appendonly yes   # or set appendonly yes in redis.conf
go run ./cmd/api
```

Quick test (with the `log` driver):

```bash
curl -X POST localhost:8080/api/v1/sms/send \
  -H 'X-API-KEY: change-me' -H 'Content-Type: application/json' \
  -d '{"client":"dev","from":"9998771827","webhook_url":"http://127.0.0.1:9009/hook","id":"msg-001","to":"01712345678","message":"Hello World"}'
```

## Project layout

```
cmd/api               main: wiring, workers, graceful shutdown
internal/config       env-based configuration
internal/dto          request payloads (single / bulk)
internal/entity       SMS + WebhookEvent models, status constants
internal/queue        Redis repository (queues, batch & status tracking)
internal/ratelimit    fixed-window limiter + provider/webhook wrappers
internal/driver       provider drivers (log, dhakacolo HTTP) — batch send
internal/service      batch creation, phone normalization, enqueue
internal/worker       round-robin dispatcher + webhook worker
internal/handler      HTTP handlers
internal/router       routes
internal/middleware   API-key auth
internal/validator    BD phone validation/normalization
pkg/redis             Redis client
pkg/validation        JSON validation + error formatting
pkg/logger            file logger
```

---

## Original requirements

> The following is the original specification this service implements.

### API Requests

#### Common Fields

Both single and bulk SMS requests will contain:

* `client` (3-letter client code, e.g. `DEV`)
* `from`
* `webhook_url`

#### Single SMS Request

```json
{
  "client": "dev",
  "from": "9998771827",
  "webhook_url": "https://apidev.smartcomm.ai/sms/webhook",
  "id": "msg-001",
  "to": "01XXXXXXXXX",
  "message": "Hello World"
}
```

#### Bulk SMS Request

```json
{
  "client": "dev",
  "from": "9998771827",
  "webhook_url": "https://apidev.smartcomm.ai/sms/webhook",
  "messages": [
    { "id": "msg-001", "to": "8801XXXXXXXXX", "message": "Hello World" },
    { "id": "msg-002", "to": "8801XXXXXXXXX", "message": "Hello World" }
  ]
}
```

### Queue Architecture

The service must generate a `batch_id` for every request and store all messages
in Redis (persistent mode). No MySQL or PostgreSQL should be used.

Each client will have dedicated queues:

* `ss-db:{client}`
* `ss-webhook:{client}`

Examples: `ss-db:dev`, `ss-webhook:dev`.

Regardless of how many requests a client sends, all messages must be pushed into
that client's dedicated queue.

### Business Problem

The SMS provider (Dhaka Colo) allows a maximum of 50 SMS per API request and has
rate limits. There may be 100+ clients using this SMS service. For example:

* Client `DEV` may submit 10,000 SMS.
* Client `ABC` may submit 50 SMS.
* Client `XYZ` may submit 500 SMS.

If messages are processed FIFO from a single queue, one client can starve all
others.

### Message Dispatch Strategy

The SMS service must use a round-robin scheduler across client queues:

1. Take up to 50 SMS from `ss-db:dev`
2. Take up to 50 SMS from `ss-db:abc`
3. Take up to 50 SMS from `ss-db:xyz`
4. Repeat continuously

This ensures fair usage across all clients.

### Webhook Processing

Delivery reports should not be sent immediately. All webhook events must be
pushed into the client's webhook queue (e.g. `ss-webhook:dev`). A dedicated
webhook worker should process these queues. To protect client systems from
overload, webhook delivery should be rate-limited — for example, a maximum of 30
webhook requests per minute per client. This prevents a Laravel application from
being overwhelmed when thousands of SMS are delivered at once.

### Storage

Use Redis only: SMS queues, webhook queues, batch tracking, delivery status, rate
limiting, scheduling metadata. Redis persistence (AOF/RDB) should be enabled to
avoid message loss — single SMS can't be lost.

### Goals

* Multi-tenant SMS platform
* Fair scheduling using round-robin
* Per-client isolation
* Batch tracking
* Webhook rate limiting
* Redis-only architecture
* Support for millions of queued SMS
* Protection against client backend overload
