# Stellar Sponsorship Service — Documentation

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Setup](#setup)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Transaction Verification Rules](#transaction-verification-rules)
- [Monitoring](#monitoring)
- [Deployment](#deployment)

---

## Overview

The Stellar Sponsorship Service enables wallets to sponsor Stellar base reserves (account creation, trustlines, etc.) without holding or transferring XLM to end users. This avoids regulatory overhead since no XLM is ever sent to external accounts.

**Key design decisions:**

- Each API key gets its own dedicated Stellar sponsor account with a specific XLM budget
- Budget enforcement happens on-chain via the Stellar network — not tracked in the database
- Every signing request (approved and rejected) is logged in the database as an audit trail. The service also checks whether signed transactions were actually submitted to the network. This provides observability without being the source of truth for balances.
- A single signing key is added as a signer on all sponsor accounts
- Transactions are validated against strict rules before co-signing (operation allowlists, sponsorship blocks, no XLM transfers)

### Signing Flow

1. Wallet sends `POST /v1/sign` with unsigned transaction XDR and API key
2. Middleware validates API key (hash lookup, expiration, status, rate limit)
3. Verifier validates the transaction structure:
   - Parses XDR and checks sponsor account is not misused
   - Validates operation types against the key's allowlist
   - Rejects any XLM transfer operations
   - Checks sponsorship block nesting
4. Service performs a pre-sign balance check against the on-chain sponsor account balance
5. Signer co-signs the transaction
6. Transaction is logged to the database (hash, operations, reserves locked, status)
7. Response includes signed XDR for the wallet to submit to the network

### API Key Lifecycle

1. Admin creates API key via dashboard → service generates a sponsor account keypair
2. Admin funds the sponsor account by signing a funding transaction with their wallet (Freighter)
3. API key becomes `active` once the funding transaction is confirmed on-chain
4. API key can be revoked, and remaining funds swept back to the master account

---

## Architecture

```
┌──────────────┐    POST /v1/sign     ┌──────────────────┐    Horizon API     ┌─────────────┐
│              │ ───────────────────→ │                  │ ─────────────────→ │   Stellar   │
│   Wallets    │    (API Key Auth)    │   Go API Server  │                    │   Network   │
│              │ ←─────────────────── │                  │ ←───────────────── │             │
└──────────────┘    Signed XDR        └──────────────────┘                    └─────────────┘
                                             │
                                             │
                                      ┌──────┴──────┐
                                      │ PostgreSQL  │
                                      │   (API keys │
                                      │    + logs)  │
                                      └─────────────┘

┌──────────────┐   Admin API (OAuth)  ┌──────────────────┐
│  Dashboard   │ ───────────────────→ │   Go API Server  │
│  (Next.js)   │ ←─────────────────── │  /v1/admin/*     │
└──────────────┘                      └──────────────────┘
```

**Components:**

| Component   | Technology             | Purpose                                                     |
| ----------- | ---------------------- | ----------------------------------------------------------- |
| API Server  | Go 1.25, chi/v5        | Transaction signing, API key management, admin API          |
| Dashboard   | Next.js 15, TypeScript | Admin UI for managing API keys and viewing logs             |
| Database    | PostgreSQL 16          | API keys, transaction logs                                  |
| Stellar SDK | go-stellar-sdk         | XDR parsing, transaction building, signing, Horizon queries |

---

## Project Structure

```
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/                  # Environment config loading
│   ├── server/                  # HTTP server and route registration
│   ├── handler/                 # HTTP handlers (sign, info, usage, admin/*)
│   ├── middleware/              # Auth, rate limiting, security headers
│   ├── stellar/                 # Stellar operations (signer, verifier, builder, account)
│   ├── service/                 # Business logic (signing, API keys, funding)
│   ├── store/                   # PostgreSQL data access layer
│   ├── model/                   # Data models (API key, transaction)
│   └── httputil/                # Response helpers, pagination
├── migrations/                  # PostgreSQL migrations (001-007)
├── dashboard/                   # Next.js admin dashboard
├── wallet-demo/                 # Example wallet integration (Next.js)
├── docker/                      # Dockerfile, docker-compose.yml
├── docs/                        # PRD, technical spec, this file
├── Makefile                     # Build, run, test, docker commands
└── .env.example                 # Environment variable template
```

---

## Setup

### Prerequisites

- Go 1.25+
- Node.js 20+
- PostgreSQL 16+
- Docker & Docker Compose (optional)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (for manual migration)

### Option 1: Docker Compose

```bash
cp .env.example .env
# Edit .env with your Stellar keys and Google OAuth credentials

cp dashboard/.env.example dashboard/.env
# Edit dashboard/.env

make docker-up
# API: http://localhost:8080
# Dashboard: http://localhost:3000
# DB: localhost:5432
```

### Option 2: Local Development

```bash
# Start PostgreSQL
docker run -d --name sponsorship-db \
  -e POSTGRES_USER=sponsorship \
  -e POSTGRES_PASSWORD=sponsorship \
  -e POSTGRES_DB=sponsorship \
  -p 5432:5432 postgres:16-alpine

# Configure environment
cp .env.example .env
# Edit .env with your values

# Run migrations
make migrate-up

# Start API server
make run

# In another terminal — start dashboard
cd dashboard
cp .env.example .env
# Edit .env
npm install
npm run dev
```

### Verify

```bash
curl http://localhost:8080/v1/health
```

---

## Configuration

### API Server (.env)

| Variable                    | Required | Default | Description                                             |
| --------------------------- | -------- | ------- | ------------------------------------------------------- |
| `STELLAR_NETWORK`           | Yes      | —       | `testnet` or `mainnet`                                  |
| `SIGNING_SECRET_KEY`        | Yes      | —       | Stellar secret key (S...) used to co-sign transactions  |
| `MASTER_FUNDING_PUBLIC_KEY` | Yes      | —       | Stellar public key (G...) of the master funding account |
| `DATABASE_URL`              | Yes      | —       | PostgreSQL connection string                            |
| `GOOGLE_CLIENT_ID`          | Yes      | —       | Google OAuth 2.0 Client ID                              |
| `GOOGLE_ALLOWED_DOMAIN`     | Yes      | —       | Google Workspace domain for admin auth                  |
| `GOOGLE_ALLOWED_EMAILS`     | Yes      | —       | Comma-separated authorized admin emails                 |
| `PORT`                      | No       | `8080`  | Server port                                             |
| `HORIZON_URL`               | No       | Auto    | Custom Horizon URL                                      |
| `LOG_LEVEL`                 | No       | `info`  | `debug`, `info`, `warn`, `error`                        |
| `CORS_ORIGINS`              | No       | —       | Comma-separated allowed CORS origins                    |

### Dashboard (dashboard/.env)

| Variable                      | Required | Description                             |
| ----------------------------- | -------- | --------------------------------------- |
| `NEXT_PUBLIC_API_URL`         | Yes      | URL of the API server                   |
| `NEXT_PUBLIC_STELLAR_NETWORK` | Yes      | `testnet` or `mainnet`                  |
| `NEXTAUTH_URL`                | Yes      | Dashboard base URL                      |
| `NEXTAUTH_SECRET`             | Yes      | Generate with `openssl rand -base64 32` |
| `GOOGLE_CLIENT_ID`            | Yes      | Google OAuth Client ID                  |
| `GOOGLE_CLIENT_SECRET`        | Yes      | Google OAuth Client Secret              |
| `GOOGLE_ALLOWED_DOMAIN`       | Yes      | Google Workspace domain                 |

### Key Concepts

- **Signing Key**: A single Stellar key pair used by the service to co-sign all transactions. It's added as a signer on every sponsor account.
- **Master Funding Account**: The operator's main Stellar account used to fund sponsor accounts. The service never holds this key — funding transactions are signed by the admin via the dashboard (Freighter wallet).
- **Sponsor Account**: A dedicated Stellar account per API key, funded with a specific XLM budget. The network enforces the budget limit.

---

## API Reference

### Public Endpoints

#### `GET /v1/info`

Returns service info (network, supported operations).

#### `GET /v1/health`

Health check with service status and metrics.

### Wallet Endpoints (API Key Auth)

Authentication: `Authorization: Bearer <api-key>`

#### `POST /v1/sign`

Sign a transaction. The request body contains the unsigned transaction XDR. The service validates and co-signs it.

#### `GET /v1/usage`

Returns the API key's current usage, budget, and limits.

### Admin Endpoints (Google OAuth)

Authentication: Google OAuth session cookie via the dashboard.

| Method   | Path                                  | Description                                                                 |
| -------- | ------------------------------------- | --------------------------------------------------------------------------- |
| `GET`    | `/v1/admin/api-keys`                  | List all API keys                                                           |
| `GET`    | `/v1/admin/api-keys/{id}`             | Get API key details                                                         |
| `POST`   | `/v1/admin/api-keys`                  | Create new API key                                                          |
| `PATCH`  | `/v1/admin/api-keys/{id}`             | Update API key settings (name, allowed operations, rate limits, expiration) |
| `POST`   | `/v1/admin/api-keys/{id}/regenerate`  | Regenerate API key secret                                                   |
| `DELETE` | `/v1/admin/api-keys/{id}`             | Revoke API key                                                              |
| `POST`   | `/v1/admin/api-keys/{id}/activate`    | Activate a pending API key                                                  |
| `POST`   | `/v1/admin/api-keys/{id}/fund`        | Build funding transaction                                                   |
| `POST`   | `/v1/admin/api-keys/{id}/fund/submit` | Submit signed funding transaction                                           |
| `POST`   | `/v1/admin/api-keys/{id}/sweep`       | Sweep funds from sponsor account                                            |
| `GET`    | `/v1/admin/transactions`              | List transaction logs                                                       |
| `GET`    | `/v1/admin/transactions/{id}/check`   | Check on-chain submission status                                            |

---

## Database Schema

### api_keys

| Column                    | Type         | Description                                                          |
| ------------------------- | ------------ | -------------------------------------------------------------------- |
| `id`                      | UUID         | Primary key                                                          |
| `name`                    | VARCHAR(255) | Display name                                                         |
| `key_hash`                | VARCHAR(64)  | SHA-256 hash of the API key                                          |
| `key_prefix`              | VARCHAR(20)  | Visible prefix (e.g., `sk_live_abc1...`)                             |
| `sponsor_account`         | VARCHAR(56)  | Stellar public key of the sponsor account (nullable)                 |
| `xlm_budget`              | BIGINT       | Budget in stroops (1 XLM = 10,000,000 stroops)                       |
| `allowed_operations`      | JSONB        | Allowed operation types (e.g., `["CREATE_ACCOUNT", "CHANGE_TRUST"]`) |
| `allowed_source_accounts` | JSONB        | Optional allowlist of source accounts                                |
| `rate_limit_max`          | INTEGER      | Max requests per window (default: 100)                               |
| `rate_limit_window`       | INTEGER      | Window in seconds (default: 60)                                      |
| `status`                  | ENUM         | `pending_funding`, `active`, `revoked`                               |
| `expires_at`              | TIMESTAMPTZ  | Expiration timestamp                                                 |
| `created_at`              | TIMESTAMPTZ  | Creation timestamp                                                   |
| `updated_at`              | TIMESTAMPTZ  | Last update timestamp                                                |

### transaction_logs

Transaction logs serve as an audit trail and observability layer. Every signing request is recorded — both successful and rejected. The service also tracks on-chain submission status by querying Horizon, so admins can see whether signed transactions were actually submitted to the network. Note that XLM budget enforcement is on-chain; these logs are for visibility, not accounting.

| Column              | Type         | Description                                 |
| ------------------- | ------------ | ------------------------------------------- |
| `id`                | UUID         | Primary key                                 |
| `api_key_id`        | UUID         | Foreign key to `api_keys`                   |
| `transaction_hash`  | VARCHAR(64)  | Stellar transaction hash (null if rejected) |
| `transaction_xdr`   | TEXT         | Transaction XDR                             |
| `operations`        | JSONB        | Operation types in the transaction          |
| `source_account`    | VARCHAR(56)  | Transaction source account                  |
| `status`            | ENUM         | `signed`, `rejected`                        |
| `rejection_reason`  | VARCHAR(255) | Reason if rejected                          |
| `submission_status` | ENUM         | `confirmed`, `not_found`                    |
| `reserves_locked`   | INTEGER      | Number of base reserves locked              |
| `created_at`        | TIMESTAMPTZ  | Creation timestamp                          |

### Migrations

Migrations are in the `migrations/` directory (001 through 007). Run with:

```bash
make migrate-up    # Apply all pending migrations
make migrate-down  # Rollback last migration
```

---

## Transaction Verification Rules

The verifier (`internal/stellar/verifier.go`) enforces these rules before signing:

1. **Valid XDR** — Must be a valid Stellar V1 transaction envelope
2. **Source account** — Transaction source must not be the sponsor account
3. **Operation validation** (per operation):
   - Sponsor account must not be used as operation source (except in `BEGIN_SPONSORING_FUTURE_RESERVES`)
   - Operation type must be in the API key's allowlist
   - XLM transfers are always rejected (`PAYMENT`, `PATH_PAYMENT_STRICT_RECEIVE`, `PATH_PAYMENT_STRICT_SEND`, `ACCOUNT_MERGE`)
   - Operations must be wrapped in valid `BEGIN_SPONSORING` / `END_SPONSORING` blocks
   - Source account must be in the allowlist (if configured)
4. **Budget check** — Estimated reserves must not exceed the sponsor account's XLM budget

---

## Monitoring

### Prometheus Metrics (`/metrics`)

| Metric                                 | Type      | Description                  |
| -------------------------------------- | --------- | ---------------------------- |
| `sponsorship_transactions_total`       | Counter   | Transactions signed/rejected |
| `sponsorship_request_duration_seconds` | Histogram | Request latency              |
| `sponsorship_sponsor_balance`          | Gauge     | Per-account XLM balance      |
| `sponsorship_active_api_keys`          | Gauge     | Number of active API keys    |

### Health Endpoint (`GET /v1/health`)

Returns service status, Stellar network, master account balance, and active API key count.

### Logging

Structured JSON logs via zerolog. Configure level with `LOG_LEVEL` env var.

---

## Deployment

### Requirements

- Docker runtime or Go 1.25+ binary
- PostgreSQL 16
- Reverse proxy (nginx/Caddy) for HTTPS termination
- Network access to Stellar Horizon API

### Security Considerations

- **Signing key** is loaded from the environment variable only, held in memory, never logged
- **API keys** are stored as SHA-256 hashes — the full key is shown only once at creation
- **Admin auth** uses Google OAuth with domain and email allowlist enforcement
- **Security headers** include HSTS, X-Content-Type-Options, X-Frame-Options, Content-Type validation
- **Rate limiting** is per-API-key with configurable window and max requests

### Makefile Commands

```bash
make build              # Build Go binary
make run                # Run API server
make test               # Run unit tests
make test-integration   # Run integration tests (requires PostgreSQL)
make migrate-up         # Apply database migrations
make migrate-down       # Rollback migrations
make docker-up          # Start all services via Docker Compose
make docker-down        # Stop all services
make lint               # Run golangci-lint
```
