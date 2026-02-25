# Stellar Sponsorship Service — Product Requirements Document

## 1. Overview

### Problem Statement

Wallets and applications building on the Stellar network need to fund base reserves for their users (account creation, trustlines, etc.), which requires holding and transferring XLM. This creates a significant KYC/KYB/due diligence burden, as handling XLM may classify the wallet operator as a money services business in various jurisdictions.

### Solution

A self-hostable, API-based sponsorship service that:

- Holds XLM in a **sponsor account** and signs transactions on behalf of wallets
- Issues **API keys** to wallets with configurable XLM usage limits, expiration dates, and allowed operation types
- **Never transfers XLM** to external accounts — funds are only used to lock base reserves via Stellar's sponsorship mechanism (`BEGIN_SPONSORING_FUTURE_RESERVES` / `END_SPONSORING_FUTURE_RESERVES`)
- Reduces the regulatory burden on wallet operators since they never custody or transfer XLM

### How It Works

1. A wallet's backend sends a Stellar transaction (XDR) to the Sponsorship Service API, authenticating with its API key
2. The service verifies the request: valid API key, sufficient XLM budget remaining, valid transaction structure, and allowed operation types
3. The service signs the transaction with the sponsor account's secret key and returns the signed XDR
4. The wallet submits the fully-signed transaction to the Stellar network
5. On-chain: the user account is created/funded, and the corresponding reserves are locked in the sponsor's reserve account

---

## 2. Goals & Non-Goals

### Goals

- **Reduce regulatory burden**: Wallets can sponsor reserves without holding or transferring XLM
- **Self-hostable**: Any entity (SDF, OpenZeppelin, or any other organization) can deploy and operate their own instance
- **Configurable**: Operators can control which operations each API key is allowed to sponsor, set XLM usage limits, and define expiration dates
- **Secure**: Robust transaction validation to prevent misuse, XLM leakage, or unauthorized operations
- **Observable**: Admin dashboard and API for monitoring usage, managing keys, and viewing analytics
- **Multi-network**: Support for both Stellar Testnet and Mainnet

### Non-Goals

- **Wallet functionality**: This service does not manage user wallets, private keys (other than the sponsor key), or user accounts
- **XLM transfers**: The service never transfers XLM to external accounts — it only locks reserves via sponsorship
- **On-chain submission**: The service does not submit transactions to the network — the wallet is responsible for submission
- **Fee payments**: The service does not pay transaction fees — only base reserves are sponsored
- **User-facing UI**: The admin dashboard is for operators, not end users

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      On-Chain (Stellar)                 │
│                                                         │
│  ┌──────────────┐    4.a Account created    ┌────────┐  │
│  │   Stellar    │ ────────────────────────>  │  User  │  │
│  │   Network    │                           │Account │  │
│  └──────┬───────┘    4.b Reserves locked    └────────┘  │
│         │         ──────────────────────>                │
│         │                        ┌─────────────────┐    │
│         │                        │ Sponsor Reserve │    │
│         │                        │    Account      │    │
│         │                        └─────────────────┘    │
└─────────┼───────────────────────────────────────────────┘
          │
    3. Submit txn
          │
┌─────────┴──────┐                ┌──────────────────────────────┐
│    Wallet's    │  1. Request    │     Sponsorship Service      │
│    Backend     │ ──signature──> │                              │
│                │   (API key)   │  ┌────────┐  ┌────────────┐  │
│                │               │  │  API   │─>│  Verify    │  │
│                │  2. Signed    │  │ Server │  │  Request   │  │
│                │ <───txn────── │  └────────┘  └─────┬──────┘  │
└────────────────┘               │                    │         │
                                 │              ┌─────▼──────┐  │
                                 │              │   Sign     │  │
                                 │              │Transaction │  │
                                 │              └────────────┘  │
                                 │                              │
                                 │  ┌────────┐  ┌────────────┐  │
                                 │  │Database│  │   Admin    │  │
                                 │  │        │  │ Dashboard  │  │
                                 │  └────────┘  └────────────┘  │
                                 └──────────────────────────────┘
```

### Components

| Component | Description |
|-----------|-------------|
| **API Server** | Receives signing requests, authenticates via API key, validates transactions, and returns signed XDR |
| **Transaction Verifier** | Core logic that validates transactions against rules (allowed ops, budget, structure) |
| **Transaction Signer** | Signs valid transactions with the sponsor account's secret key |
| **Database** | Stores API keys, usage tracking, transaction logs, and configuration |
| **Admin Dashboard** | Web-based UI for managing API keys, monitoring usage, and viewing analytics |
| **Sponsor Account** | Stellar account that holds XLM and whose key is used to sign sponsorship transactions |

---

## 4. Core Concepts

### Sponsor Account

A Stellar account controlled by the service operator. It holds XLM that gets locked as base reserves when sponsoring operations. The secret key is stored securely by the service and used to co-sign transactions.

### API Keys

Each wallet onboarded to the service receives an API key with:

| Property | Description |
|----------|-------------|
| `key` | Unique identifier used for authentication |
| `name` | Human-readable label for the wallet/partner |
| `xlm_limit` | Maximum total XLM (in stroops) that can be used for reserves |
| `xlm_used` | Current XLM consumed (tracked by the service) |
| `allowed_operations` | List of sponsorable operation types this key can use |
| `expires_at` | Expiration date after which the key is invalid |
| `is_active` | Whether the key is currently enabled |
| `rate_limit` | Maximum requests per time window |
| `allowed_source_accounts` | Optional allowlist of Stellar accounts that can appear as transaction source |

### Sponsorable Operations

The following Stellar operations can be sponsored, configurable per API key:

| Operation | Reserve Cost | Description |
|-----------|-------------|-------------|
| `CREATE_ACCOUNT` | 1 base reserve (1 XLM) | Creating a new Stellar account |
| `CHANGE_TRUST` | 0.5 base reserve (0.5 XLM) | Adding a trustline to an asset |
| `MANAGE_SELL_OFFER` | 0.5 base reserve (0.5 XLM) | Creating an offer on the DEX |
| `MANAGE_BUY_OFFER` | 0.5 base reserve (0.5 XLM) | Creating a buy offer on the DEX |
| `SET_OPTIONS` (signer) | 0.5 base reserve (0.5 XLM) | Adding a signer to an account |
| `MANAGE_DATA` | 0.5 base reserve (0.5 XLM) | Adding a data entry to an account |
| `CREATE_CLAIMABLE_BALANCE` | 0.5 base reserve (0.5 XLM) | Creating a claimable balance |

> Note: Base reserve values are based on the current Stellar network configuration (0.5 XLM per base reserve, 1 XLM minimum account balance). These may change via network votes.

### XLM Usage Rules

- XLM is **only used to lock base reserves** via the Stellar sponsorship mechanism
- **No XLM is ever transferred** to external accounts (no `PAYMENT`, `PATH_PAYMENT_*`, or `ACCOUNT_MERGE` operations allowed)
- Usage is tracked per API key and checked before signing
- When a sponsored entry is removed (e.g., a trustline is removed), the reserves are unlocked and returned to the sponsor account — the service should track this to update usage

---

## 5. API Specification

### Authentication

All API requests are authenticated via an API key passed in the `Authorization` header:

```
Authorization: Bearer <api_key>
```

Admin endpoints require a separate admin token or role-based access.

### Endpoints

#### 5.1 Sign Transaction

```
POST /v1/sign
```

Signs a transaction that includes valid sponsorship operations.

**Request Body:**

```json
{
  "transaction_xdr": "AAAAAG5o...",
  "network_passphrase": "Test SDF Network ; September 2015"
}
```

**Response (200):**

```json
{
  "signed_transaction_xdr": "AAAAAG5o...",
  "sponsor_public_key": "GCXYZ...",
  "xlm_reserved": "1.5000000",
  "xlm_remaining": "998.5000000"
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_transaction` | Transaction XDR is malformed or cannot be decoded |
| 400 | `disallowed_operation` | Transaction contains operations not allowed for this API key |
| 400 | `invalid_sponsor` | Sponsor in transaction does not match the service's sponsor account |
| 400 | `sponsor_as_source` | Transaction or operation uses the sponsor account as the source account |
| 400 | `xlm_transfer_detected` | Transaction attempts to transfer XLM (payment, path payment, merge) |
| 401 | `invalid_api_key` | API key is missing, invalid, or expired |
| 403 | `key_disabled` | API key has been deactivated |
| 409 | `xlm_limit_exceeded` | Signing this transaction would exceed the API key's XLM limit |
| 429 | `rate_limited` | Too many requests — rate limit exceeded |

#### 5.2 Get Sponsor Info

```
GET /v1/info
```

Returns the sponsor account's public key and network info. No authentication required.

**Response (200):**

```json
{
  "sponsor_public_key": "GCXYZ...",
  "network_passphrase": "Test SDF Network ; September 2015",
  "base_reserve": "0.5000000",
  "supported_operations": [
    "CREATE_ACCOUNT",
    "CHANGE_TRUST",
    "MANAGE_SELL_OFFER",
    "MANAGE_BUY_OFFER",
    "SET_OPTIONS",
    "MANAGE_DATA",
    "CREATE_CLAIMABLE_BALANCE"
  ]
}
```

#### 5.3 Get API Key Usage

```
GET /v1/usage
```

Returns the current usage and limits for the authenticated API key.

**Response (200):**

```json
{
  "api_key_name": "Wallet Co",
  "xlm_limit": "1000.0000000",
  "xlm_used": "150.5000000",
  "xlm_remaining": "849.5000000",
  "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
  "expires_at": "2027-01-01T00:00:00Z",
  "is_active": true,
  "transactions_signed": 1523,
  "rate_limit": {
    "max_requests": 100,
    "window_seconds": 60,
    "remaining": 87
  }
}
```

#### 5.4 Admin — Manage API Keys

##### List API Keys

```
GET /v1/admin/api-keys
```

**Response (200):**

```json
{
  "api_keys": [
    {
      "id": "uuid",
      "name": "Wallet Co",
      "key_prefix": "sk_live_abc1....",
      "xlm_limit": "1000.0000000",
      "xlm_used": "150.5000000",
      "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
      "expires_at": "2027-01-01T00:00:00Z",
      "is_active": true,
      "created_at": "2026-01-15T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "per_page": 20
}
```

##### Create API Key

```
POST /v1/admin/api-keys
```

**Request Body:**

```json
{
  "name": "Wallet Co",
  "xlm_limit": "1000.0000000",
  "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
  "expires_at": "2027-01-01T00:00:00Z",
  "rate_limit": {
    "max_requests": 100,
    "window_seconds": 60
  },
  "allowed_source_accounts": ["GABC...", "GDEF..."]
}
```

**Response (201):**

```json
{
  "id": "uuid",
  "name": "Wallet Co",
  "api_key": "sk_live_abc123...",
  "xlm_limit": "1000.0000000",
  "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
  "expires_at": "2027-01-01T00:00:00Z",
  "created_at": "2026-02-19T12:00:00Z"
}
```

> The full API key is only returned once at creation time. It should be securely shared with the wallet operator.

##### Update API Key

```
PATCH /v1/admin/api-keys/:id
```

Allows updating: `name`, `xlm_limit`, `allowed_operations`, `expires_at`, `is_active`, `rate_limit`, `allowed_source_accounts`.

##### Revoke API Key

```
DELETE /v1/admin/api-keys/:id
```

Permanently revokes an API key. This cannot be undone.

#### 5.5 Admin — Transaction Logs

```
GET /v1/admin/transactions
```

Returns a paginated list of signed transactions with filters.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `api_key_id` | string | Filter by API key |
| `status` | string | `signed`, `rejected` |
| `from` | datetime | Start date |
| `to` | datetime | End date |
| `page` | integer | Page number |
| `per_page` | integer | Items per page (max 100) |

**Response (200):**

```json
{
  "transactions": [
    {
      "id": "uuid",
      "api_key_id": "uuid",
      "api_key_name": "Wallet Co",
      "transaction_hash": "abc123...",
      "xlm_reserved": "1.5000000",
      "operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
      "status": "signed",
      "created_at": "2026-02-19T12:05:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "per_page": 20
}
```

#### 5.6 Health Check

```
GET /v1/health
```

**Response (200):**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "stellar_network": "testnet",
  "sponsor_account_balance": "9850.0000000",
  "uptime_seconds": 86400
}
```

---

## 6. Data Model

### API Keys Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(255) | Human-readable label |
| `key_hash` | VARCHAR(255) | Hashed API key (never store plaintext) |
| `key_prefix` | VARCHAR(20) | First few characters of the key for identification |
| `xlm_limit` | BIGINT | Maximum XLM in stroops (1 XLM = 10,000,000 stroops) |
| `xlm_used` | BIGINT | Current XLM used in stroops |
| `allowed_operations` | JSONB | Array of allowed Stellar operation types |
| `allowed_source_accounts` | JSONB | Optional array of allowed source account public keys |
| `rate_limit_max` | INTEGER | Max requests per window |
| `rate_limit_window` | INTEGER | Window size in seconds |
| `expires_at` | TIMESTAMP | Expiration date |
| `is_active` | BOOLEAN | Whether the key is currently enabled |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

### Transaction Logs Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `api_key_id` | UUID | Foreign key to API keys |
| `transaction_hash` | VARCHAR(64) | Stellar transaction hash |
| `transaction_xdr` | TEXT | The signed transaction XDR |
| `xlm_reserved` | BIGINT | XLM locked by this transaction (stroops) |
| `operations` | JSONB | Array of operation types in this transaction |
| `source_account` | VARCHAR(56) | Transaction source account public key |
| `status` | VARCHAR(20) | `signed` or `rejected` |
| `rejection_reason` | VARCHAR(255) | Reason for rejection (if rejected) |
| `created_at` | TIMESTAMP | Timestamp of the signing request |

---

## 7. Transaction Verification Rules

When a transaction is submitted for signing, the service **must** perform the following checks in order:

### 7.1 API Key Validation

1. API key is present and valid (hash matches a stored key)
2. API key has not expired (`expires_at > now`)
3. API key is active (`is_active = true`)
4. Rate limit has not been exceeded

### 7.2 Transaction Structure Validation

1. Transaction XDR is valid and can be decoded
2. Network passphrase matches the configured network
3. Transaction contains at least one operation
4. The sponsor account in `BEGIN_SPONSORING_FUTURE_RESERVES` operations matches the service's sponsor public key

### 7.3 Source Account Validation

**Critical security rule**: The sponsor account must **NEVER** appear as:

1. The **transaction source account** — a transaction sourced from the sponsor account would authorize any operation to act on behalf of the sponsor
2. The **source account of any individual operation** — same risk at the operation level
3. The sponsor account should **ONLY** appear inside `BEGIN_SPONSORING_FUTURE_RESERVES` as the sponsoring account

This prevents a malicious wallet from crafting a transaction that drains the sponsor account's XLM balance by using it as the source of payments, merges, or any other operation.

### 7.4 Operation Validation

For each operation in the transaction:

1. The operation type is in the API key's `allowed_operations` list
2. Operations that can transfer XLM are **always rejected**:
   - `PAYMENT` (when asset is native XLM)
   - `PATH_PAYMENT_STRICT_SEND` (when destination asset is native)
   - `PATH_PAYMENT_STRICT_RECEIVE` (when destination asset is native)
   - `ACCOUNT_MERGE`
   - `INFLATION` (deprecated but should be blocked)
   - `CLAWBACK` (native XLM)
3. Operations are wrapped in `BEGIN_SPONSORING_FUTURE_RESERVES` / `END_SPONSORING_FUTURE_RESERVES` where the sponsor is the service's account
4. If `allowed_source_accounts` is configured, the transaction source or operation source must be in the allowlist

### 7.5 Budget Validation

1. Calculate the total XLM reserves required by all operations in the transaction
2. Check that `xlm_used + required_reserves <= xlm_limit` for the API key
3. If the budget would be exceeded, reject with `xlm_limit_exceeded`

### 7.6 Post-Signing

1. Update `xlm_used` for the API key
2. Log the transaction in the transaction logs table
3. Return the signed transaction XDR

---

## 8. Admin Dashboard

A web-based dashboard for service operators to manage the sponsorship service.

### Pages

#### 8.1 Dashboard / Overview

- Total XLM locked across all API keys
- Sponsor account balance (live from Stellar)
- Number of active API keys
- Transactions signed (today / week / month / all-time)
- Chart: signing volume over time
- Chart: XLM usage over time

#### 8.2 API Keys Management

- Table of all API keys with search and filters
- Create new API key (form)
- Edit API key (update limits, operations, expiration)
- Activate / deactivate API key
- Revoke API key (with confirmation)
- Per-key usage details and charts

#### 8.3 Transaction Logs

- Searchable, filterable table of all signing requests
- Filter by: API key, status (signed/rejected), date range, operation type
- Transaction detail view with full XDR and operation breakdown

#### 8.4 Settings

- Sponsor account public key display
- Network configuration (testnet/mainnet)
- Global rate limiting defaults
- Service version and health status

---

## 9. Security Considerations

### Sponsor Secret Key

- The sponsor account's secret key **must** be stored securely (environment variable, secrets manager such as AWS Secrets Manager, HashiCorp Vault, etc.)
- The key should **never** be logged, returned in API responses, or stored in the database
- Consider supporting HSM or KMS-based signing in future versions

### API Key Security

- API keys are hashed before storage (e.g., SHA-256 or bcrypt)
- Only the key prefix is stored in plaintext for identification
- Full API key is shown only once at creation time
- Keys can be revoked instantly

### Transaction Safety

- Strict validation prevents XLM transfers to external accounts
- Only sponsorship-related operations are allowed
- All operations must be wrapped in proper sponsoring blocks
- Source account allowlisting provides additional control

### Rate Limiting

- Per-API-key rate limiting to prevent abuse
- Global rate limiting to protect service availability
- Rate limit headers returned in responses (`X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`)

### Network Security

- HTTPS required for all API communication
- CORS configuration for admin dashboard
- Admin endpoints require separate authentication (admin token or role-based access)

---

## 10. Deployment

### Self-Hosting Requirements

The service should be deployable with minimal infrastructure:

- **Runtime**: Containerized (Docker) for portability
- **Database**: PostgreSQL (recommended) or SQLite for simple deployments
- **Reverse proxy**: Any (nginx, Caddy, etc.) — HTTPS termination
- **No external dependencies**: Beyond the database and Stellar network access

### Environment Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `STELLAR_NETWORK` | Yes | `testnet` or `mainnet` |
| `SPONSOR_SECRET_KEY` | Yes | Secret key of the sponsor Stellar account |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `ADMIN_API_KEY` | Yes | Secret key for admin API access |
| `PORT` | No | Server port (default: 8080) |
| `HORIZON_URL` | No | Custom Horizon server URL |
| `LOG_LEVEL` | No | Logging level (default: `info`) |

### Docker

```bash
docker run -d \
  -e STELLAR_NETWORK=testnet \
  -e SPONSOR_SECRET_KEY=SXXX... \
  -e DATABASE_URL=postgres://... \
  -e ADMIN_API_KEY=admin_secret \
  -p 8080:8080 \
  sponsorship-service:latest
```

### Docker Compose

A `docker-compose.yml` should be provided for running the service with PostgreSQL locally for development and evaluation.

---

## 11. Monitoring & Observability

### Metrics (Prometheus-compatible)

| Metric | Type | Description |
|--------|------|-------------|
| `sponsorship_transactions_total` | Counter | Total transactions signed, labeled by status and API key |
| `sponsorship_xlm_reserved_total` | Counter | Total XLM reserved across all keys |
| `sponsorship_api_key_usage_ratio` | Gauge | XLM used / XLM limit per API key |
| `sponsorship_request_duration_seconds` | Histogram | API request latency |
| `sponsorship_sponsor_balance` | Gauge | Current XLM balance of sponsor account |
| `sponsorship_active_api_keys` | Gauge | Number of active API keys |

### Alerts (recommended)

- Sponsor account balance below threshold
- API key approaching XLM limit (>90%)
- High rate of rejected transactions
- Service health check failing
- Unusual signing volume spike

### Logging

- Structured JSON logging for all requests
- Transaction signing events include: API key ID, transaction hash, XLM reserved, operations, result
- Sensitive data (secret keys, full API keys) must **never** be logged

---

## 12. Future Considerations

These are explicitly out of scope for v1 but worth considering for future versions:

- **Reserve recovery tracking**: Monitor on-chain events to detect when sponsored entries are removed and reserves are unlocked, automatically updating `xlm_used` downward
- **Webhooks**: Notify wallets when their API key is approaching limits or expiring
- **Multi-sponsor accounts**: Support multiple sponsor accounts for load distribution or isolation
- **HSM/KMS signing**: Hardware security module or cloud KMS integration for sponsor key security
- **Fee sponsorship**: Optionally sponsor transaction fees in addition to base reserves
- **Allowlist/blocklist for destination accounts**: Control which accounts can be sponsored
- **Batch signing**: Accept multiple transactions in a single API call
- **SDK/Client libraries**: Provide client libraries for common languages (JavaScript, Python, Go, etc.)
- **Audit log**: Immutable audit trail of all administrative actions
