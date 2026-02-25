# Stellar Sponsorship Service — Product Requirements Document (v2)

## 1. Overview

### Problem Statement

Wallets and applications building on the Stellar network need to fund base reserves for their users (account creation, trustlines, etc.), which requires holding and transferring XLM. This creates a significant KYC/KYB/due diligence burden, as handling XLM may classify the wallet operator as a money services business in various jurisdictions.

### Solution

A self-hostable, API-based sponsorship service that:

- Creates a **dedicated sponsor account per API key**, funded with the wallet's XLM budget — the Stellar network itself enforces the spending limit
- Uses a **single signing key** added as a signer on all sponsor accounts, keeping key management simple
- Issues **API keys** to wallets with configurable expiration dates and allowed operation types
- **Never transfers XLM** to external accounts — funds are only used to lock base reserves via Stellar's sponsorship mechanism (`BEGIN_SPONSORING_FUTURE_RESERVES` / `END_SPONSORING_FUTURE_RESERVES`)
- Reduces the regulatory burden on wallet operators since they never custody or transfer XLM

### How It Works

1. An operator creates an API key for a wallet via the admin API or dashboard
2. The service automatically creates a dedicated Stellar sponsor account for that key and funds it with the configured XLM limit from a master funding account
3. The wallet's backend sends a Stellar transaction (XDR) to the Sponsorship Service API, authenticating with its API key
4. The service verifies the request: valid API key, valid transaction structure, and allowed operation types
5. The service signs the transaction with the signing key (which is a signer on the wallet's sponsor account) and returns the signed XDR
6. The wallet submits the fully-signed transaction to the Stellar network
7. On-chain: the user account is created/funded, and the corresponding reserves are locked in the wallet's dedicated sponsor account

### Key Architectural Decision: Per-Key Sponsor Accounts

Instead of tracking XLM usage in a database, each API key gets its own dedicated Stellar account funded with exactly its XLM budget. This provides:

- **On-chain budget enforcement**: The Stellar network itself enforces the limit — if the account doesn't have enough XLM, the transaction fails on-chain
- **Automatic reserve recovery**: When a sponsored entry is removed (e.g., trustline removed), XLM is unlocked and returned to that specific sponsor account — no DB reconciliation needed
- **No sync issues**: No risk of DB state drifting from on-chain reality
- **Natural isolation**: One wallet's usage cannot affect another
- **Simple key management**: A single signing key is added as a signer on all sponsor accounts

---

## 2. Goals & Non-Goals

### Goals

- **Reduce regulatory burden**: Wallets can sponsor reserves without holding or transferring XLM
- **Self-hostable**: Any entity (SDF, OpenZeppelin, or any other organization) can deploy and operate their own instance
- **Configurable**: Operators can control which operations each API key is allowed to sponsor, set XLM budgets, and define expiration dates
- **On-chain truth**: XLM budgets are enforced by the Stellar network via per-key sponsor accounts, not database tracking
- **Secure**: Robust transaction validation to prevent misuse, XLM leakage, or unauthorized operations
- **Observable**: Admin dashboard and API for monitoring usage, managing keys, and viewing analytics
- **Multi-network**: Support for both Stellar Testnet and Mainnet

### Non-Goals

- **Wallet functionality**: This service does not manage user wallets, private keys (other than the signing key), or user accounts
- **XLM transfers**: The service never transfers XLM to external accounts — it only locks reserves via sponsorship
- **On-chain submission**: The service does not submit transactions to the network — the wallet is responsible for submission
- **Fee payments**: The service does not pay transaction fees — only base reserves are sponsored
- **User-facing UI**: The admin dashboard is for operators, not end users

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      On-Chain (Stellar)                      │
│                                                              │
│  ┌──────────────┐    5.a Account created    ┌────────────┐   │
│  │   Stellar    │ ───────────────────────>  │    User    │   │
│  │   Network    │                           │  Account   │   │
│  └──────┬───────┘    5.b Reserves locked    └────────────┘   │
│         │         ─────────────────────>                      │
│         │                        ┌──────────────────────┐    │
│         │                        │  Wallet A Sponsor    │    │
│         │                        │  Account (dedicated) │    │
│         │                        └──────────────────────┘    │
│         │                        ┌──────────────────────┐    │
│         │                        │  Wallet B Sponsor    │    │
│         │                        │  Account (dedicated) │    │
│         │                        └──────────────────────┘    │
│         │                        ┌──────────────────────┐    │
│         │                        │  Master Funding      │    │
│         │                        │  Account             │    │
│         │                        └──────────────────────┘    │
└─────────┼────────────────────────────────────────────────────┘
          │
    4. Submit txn
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
                                 │              │(single key)│  │
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
| **Transaction Verifier** | Core logic that validates transactions against rules (allowed ops, structure, sponsor account match) |
| **Transaction Signer** | Signs valid transactions with the single signing key (which is a signer on all sponsor accounts) |
| **Database** | Stores API keys, sponsor account mappings, transaction logs, and configuration |
| **Admin Dashboard** | Web-based UI for managing API keys, monitoring usage, and viewing analytics |
| **Master Funding Account** | Stellar account that holds the total XLM pool — controlled by the operator via an external wallet (e.g., Freighter), never by the service |
| **Per-Key Sponsor Accounts** | Dedicated Stellar accounts created per API key, funded with the key's XLM budget |
| **Signing Key** | A single Stellar keypair added as a signer on all sponsor accounts — used to sign all transactions |

---

## 4. Core Concepts

### Master Funding Account

A Stellar account controlled by the service operator via an external wallet (e.g., Freighter through [Stellar Wallets Kit](https://stellarwalletskit.dev/)). The master account holds the total XLM pool used to fund individual sponsor accounts.

**The master account's secret key is never stored by or accessible to the service.** All funding operations (creating sponsor accounts, adding funds, sweeping revoked accounts) are initiated through the admin dashboard, which builds the transaction and presents it to the operator's external wallet for approval and signing. This keeps the highest-value key completely outside the service's attack surface.

### Per-Key Sponsor Accounts

Each API key has a dedicated Stellar account:

- Created when the API key is provisioned — the funding transaction is signed by the operator via their external wallet
- Funded with the wallet's configured XLM budget from the master funding account
- The single signing key is added as a signer on this account
- The account's on-chain XLM balance is the source of truth for remaining budget
- Reserve recovery (when sponsored entries are removed) automatically returns XLM to this account

### Signing Key

A single Stellar keypair managed by the service:

- Added as a signer (with appropriate weight) on every per-key sponsor account
- Used to sign all sponsorship transactions regardless of which API key is used
- Simplifies key management — only one secret key to secure
- The signing key itself does not hold XLM

### API Keys

Each wallet onboarded to the service receives an API key with:

| Property | Description |
|----------|-------------|
| `key` | Unique identifier used for authentication |
| `name` | Human-readable label for the wallet/partner |
| `sponsor_account` | Public key of this key's dedicated sponsor account |
| `xlm_budget` | XLM amount the sponsor account was funded with |
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
- Budget enforcement is on-chain: the sponsor account's balance is the limit
- When a sponsored entry is removed (e.g., a trustline is removed), the reserves are automatically unlocked and returned to the sponsor account — no manual tracking needed

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
  "sponsor_account_balance": "998.5000000"
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_transaction` | Transaction XDR is malformed or cannot be decoded |
| 400 | `disallowed_operation` | Transaction contains operations not allowed for this API key |
| 400 | `invalid_sponsor` | Sponsor in transaction does not match this key's sponsor account |
| 400 | `sponsor_as_source` | Transaction or operation uses the sponsor account as the source account |
| 400 | `xlm_transfer_detected` | Transaction attempts to transfer XLM (payment, path payment, merge) |
| 401 | `invalid_api_key` | API key is missing, invalid, or expired |
| 403 | `key_disabled` | API key has been deactivated |
| 429 | `rate_limited` | Too many requests — rate limit exceeded |

> Note: There is no `xlm_limit_exceeded` error at the API level. Budget enforcement happens on-chain — if the sponsor account lacks sufficient XLM, the transaction will fail when submitted to the Stellar network. The `sponsor_account_balance` in the response helps wallets check availability before submitting.

#### 5.2 Get Sponsor Info

```
GET /v1/info
```

Returns the service's network info and supported operations. No authentication required.

**Response (200):**

```json
{
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

Returns the current usage and limits for the authenticated API key. Balance is fetched from the Stellar network (on-chain source of truth).

**Response (200):**

```json
{
  "api_key_name": "Wallet Co",
  "sponsor_account": "GCXYZ...",
  "xlm_budget": "1000.0000000",
  "xlm_available": "849.5000000",
  "xlm_locked_in_reserves": "150.5000000",
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

> `xlm_available` and `xlm_locked_in_reserves` are derived from the sponsor account's on-chain balance, not database tracking.

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
      "sponsor_account": "GCXYZ...",
      "xlm_budget": "1000.0000000",
      "xlm_available": "849.5000000",
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

Creates an API key and provisions a dedicated Stellar sponsor account. The funding transaction is built by the service but **signed by the operator via their external wallet** (e.g., Freighter) in the admin dashboard.

**Request Body:**

```json
{
  "name": "Wallet Co",
  "xlm_budget": "1000.0000000",
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
  "sponsor_account": "GCXYZ...",
  "xlm_budget": "1000.0000000",
  "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
  "expires_at": "2027-01-01T00:00:00Z",
  "funding_transaction_xdr": "AAAAAG5o...",
  "status": "pending_funding",
  "created_at": "2026-02-19T12:00:00Z"
}
```

> The full API key is only returned once at creation time. It should be securely shared with the wallet operator.

**What happens on creation:**

1. A new Stellar keypair is generated for the sponsor account
2. The service builds a transaction from the master funding account that:
   - Creates the new sponsor account with `xlm_budget` XLM
   - Adds the signing key as a signer on the new account
3. The transaction XDR is returned as `funding_transaction_xdr`
4. The admin dashboard presents this transaction to the operator's external wallet (e.g., Freighter via Stellar Wallets Kit) for approval and signing
5. Once signed, the dashboard submits the transaction to the network via `POST /v1/admin/api-keys/:id/activate`
6. The sponsor account's generated secret key can be discarded — the signing key is the only key needed going forward
7. The API key status moves from `pending_funding` to `active`

##### Activate API Key (after funding)

```
POST /v1/admin/api-keys/:id/activate
```

Called after the operator signs the funding transaction in their external wallet. Submits the signed transaction to the Stellar network and activates the API key.

**Request Body:**

```json
{
  "signed_transaction_xdr": "AAAAAG5o..."
}
```

**Response (200):**

```json
{
  "id": "uuid",
  "status": "active",
  "sponsor_account": "GCXYZ...",
  "transaction_hash": "abc123..."
}
```

##### Update API Key

```
PATCH /v1/admin/api-keys/:id
```

Allows updating: `name`, `allowed_operations`, `expires_at`, `is_active`, `rate_limit`, `allowed_source_accounts`.

##### Build Fund Transaction

```
POST /v1/admin/api-keys/:id/fund
```

Builds a transaction to send additional XLM from the master funding account to the API key's sponsor account. The transaction must be signed by the operator's external wallet.

**Request Body:**

```json
{
  "amount": "500.0000000"
}
```

**Response (200):**

```json
{
  "sponsor_account": "GCXYZ...",
  "xlm_to_add": "500.0000000",
  "funding_transaction_xdr": "AAAAAG5o..."
}
```

The admin dashboard presents this transaction to the operator's external wallet for signing, then submits:

```
POST /v1/admin/api-keys/:id/fund/submit
```

**Request Body:**

```json
{
  "signed_transaction_xdr": "AAAAAG5o..."
}
```

**Response (200):**

```json
{
  "sponsor_account": "GCXYZ...",
  "xlm_added": "500.0000000",
  "xlm_available": "1349.5000000",
  "transaction_hash": "abc123..."
}
```

##### Revoke API Key

```
DELETE /v1/admin/api-keys/:id
```

Permanently revokes an API key. The API key is deactivated and will no longer be accepted for signing requests. The sponsor account and its funds remain on-chain — the operator can choose to sweep funds back to the master account separately if desired.

#### 5.5 Admin — Sweep Funds

##### Build Sweep Transaction

```
POST /v1/admin/api-keys/:id/sweep
```

Builds a transaction to transfer available (unlocked) XLM from a revoked API key's sponsor account back to the master funding account. The transaction is signed by the service's signing key (which is a signer on the sponsor account), so no external wallet signing is needed for this operation.

**Response (200):**

```json
{
  "sponsor_account": "GCXYZ...",
  "xlm_swept": "849.5000000",
  "xlm_remaining_locked": "150.5000000",
  "destination": "GMASTER...",
  "transaction_hash": "def456..."
}
```

> Only available on revoked keys. Locked reserves cannot be swept — they will be recovered as sponsored entries are removed over time.

#### 5.6 Admin — Transaction Logs

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

#### 5.7 Health Check

```
GET /v1/health
```

**Response (200):**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "stellar_network": "testnet",
  "master_account_balance": "50000.0000000",
  "total_sponsor_accounts": 12,
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
| `sponsor_account_public_key` | VARCHAR(56) | Public key of this key's dedicated sponsor account |
| `xlm_budget` | BIGINT | XLM originally funded to the sponsor account (stroops) |
| `allowed_operations` | JSONB | Array of allowed Stellar operation types |
| `allowed_source_accounts` | JSONB | Optional array of allowed source account public keys |
| `rate_limit_max` | INTEGER | Max requests per window |
| `rate_limit_window` | INTEGER | Window size in seconds |
| `expires_at` | TIMESTAMP | Expiration date |
| `is_active` | BOOLEAN | Whether the key is currently enabled |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

> Note: There is no `xlm_used` column. Usage is derived from the on-chain balance of the sponsor account.

### Transaction Logs Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `api_key_id` | UUID | Foreign key to API keys |
| `transaction_hash` | VARCHAR(64) | Stellar transaction hash |
| `transaction_xdr` | TEXT | The signed transaction XDR |
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
4. The sponsor account in `BEGIN_SPONSORING_FUTURE_RESERVES` operations matches this API key's dedicated sponsor account

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
3. Operations are wrapped in `BEGIN_SPONSORING_FUTURE_RESERVES` / `END_SPONSORING_FUTURE_RESERVES` where the sponsor is the API key's dedicated sponsor account
4. If `allowed_source_accounts` is configured, the transaction source or operation source must be in the allowlist

### 7.5 Post-Signing

1. Log the transaction in the transaction logs table
2. Return the signed transaction XDR with the sponsor account's current balance

> Note: No database budget update is needed — the Stellar network enforces the budget via the sponsor account's on-chain balance.

---

## 8. Admin Dashboard

A web-based dashboard for service operators to manage the sponsorship service.

### Pages

#### 8.1 Dashboard / Overview

- Total XLM across all sponsor accounts (available + locked in reserves)
- Master funding account balance (live from Stellar)
- Number of active API keys / sponsor accounts
- Transactions signed (today / week / month / all-time)
- Chart: signing volume over time
- Chart: XLM usage over time across all sponsor accounts

#### 8.2 API Keys Management

- Table of all API keys with search and filters
- Each row shows: name, sponsor account, XLM available (live), allowed operations, status, expiration
- Create new API key (form) — provisions sponsor account automatically
- Edit API key (update operations, expiration, rate limit)
- Add funds to an API key's sponsor account
- Activate / deactivate API key
- Revoke API key (with confirmation)
- Sweep funds from revoked keys
- Per-key usage details and charts

#### 8.3 Transaction Logs

- Searchable, filterable table of all signing requests
- Filter by: API key, status (signed/rejected), date range, operation type
- Transaction detail view with full XDR and operation breakdown

#### 8.4 Settings

- Master funding account public key and balance
- Signing key public key
- Network configuration (testnet/mainnet)
- Global rate limiting defaults
- Service version and health status

---

## 9. Security Considerations

### Signing Key

- A single signing key is used across all sponsor accounts
- The secret key **must** be stored securely (environment variable, secrets manager such as AWS Secrets Manager, HashiCorp Vault, etc.)
- The key should **never** be logged, returned in API responses, or stored in the database
- The signing key is added as a signer on each sponsor account — it does not hold XLM itself
- Consider supporting HSM or KMS-based signing in future versions

### Master Funding Account

- The master funding account's secret key is **never stored by or accessible to the service**
- All funding operations (creating sponsor accounts, adding funds) are signed by the operator via an external wallet (e.g., Freighter) through the admin dashboard
- The service only builds unsigned transactions — the operator reviews and approves each one in their wallet
- This eliminates the highest-value secret key from the service's attack surface
- The operator can further protect the master account with multisig, hardware wallets, or any Stellar-compatible signing method

### Sponsor Account Isolation

- Each API key's sponsor account is fully isolated — one wallet's usage cannot affect another
- If a sponsor account is compromised, only that wallet's XLM budget is at risk (not the entire pool)
- The signing key being shared is the single point of trust — its security is critical

### API Key Security

- API keys are hashed before storage (e.g., SHA-256 or bcrypt)
- Only the key prefix is stored in plaintext for identification
- Full API key is shown only once at creation time
- Keys can be revoked instantly — revocation is enforced at the API layer

### Transaction Safety

- Strict validation prevents XLM transfers to external accounts
- Only sponsorship-related operations are allowed
- All operations must be wrapped in proper sponsoring blocks
- Sponsor account can never be the source of a transaction or operation
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
| `SIGNING_SECRET_KEY` | Yes | Secret key used to sign transactions (added as signer on all sponsor accounts) |
| `MASTER_FUNDING_PUBLIC_KEY` | Yes | Public key of the master funding account (used to build funding transactions — secret key stays in the operator's external wallet) |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `ADMIN_API_KEY` | Yes | Secret key for admin API access |
| `PORT` | No | Server port (default: 8080) |
| `HORIZON_URL` | No | Custom Horizon server URL |
| `LOG_LEVEL` | No | Logging level (default: `info`) |

### Docker

```bash
docker run -d \
  -e STELLAR_NETWORK=testnet \
  -e SIGNING_SECRET_KEY=SXXX... \
  -e MASTER_FUNDING_PUBLIC_KEY=GYYY... \
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
| `sponsorship_request_duration_seconds` | Histogram | API request latency |
| `sponsorship_master_balance` | Gauge | Current XLM balance of master funding account |
| `sponsorship_sponsor_balance` | Gauge | Current XLM balance per sponsor account (labeled by API key) |
| `sponsorship_active_api_keys` | Gauge | Number of active API keys |

### Alerts (recommended)

- Master funding account balance below threshold
- Sponsor account balance approaching zero
- High rate of rejected transactions
- Service health check failing
- Unusual signing volume spike

### Logging

- Structured JSON logging for all requests
- Transaction signing events include: API key ID, transaction hash, operations, result
- Sensitive data (secret keys, full API keys) must **never** be logged

---

## 12. API Key Lifecycle

### Creation Flow

1. Admin creates API key via `POST /v1/admin/api-keys` with name, budget, operations, and expiration
2. Service generates a new Stellar keypair for the sponsor account
3. Service builds a funding transaction (unsigned) that creates the sponsor account, funds it, and adds the signing key as a signer
4. The admin dashboard presents the transaction to the operator's external wallet (e.g., Freighter via Stellar Wallets Kit) for review and signing
5. Once signed, the dashboard calls `POST /v1/admin/api-keys/:id/activate` with the signed XDR
6. Service submits the transaction to the Stellar network and activates the API key
7. The sponsor account's generated secret key is discarded — the signing key is the only key needed going forward
8. Service returns the API key (shown only once) and sponsor account public key

### Active Usage

- Wallet authenticates with the API key to sign transactions
- The signing key signs on behalf of the wallet's sponsor account
- XLM reserves are locked/unlocked in the dedicated sponsor account
- On-chain balance is the source of truth for available budget

### Adding Funds

- Admin requests a funding transaction via `POST /v1/admin/api-keys/:id/fund`
- Service builds the transaction; operator signs it in their external wallet
- Dashboard submits the signed transaction via `POST /v1/admin/api-keys/:id/fund/submit`

### Revocation

- Admin revokes via `DELETE /v1/admin/api-keys/:id`
- API key is deactivated — signing requests are immediately rejected
- Sponsor account and funds remain untouched on-chain
- Admin can optionally sweep available (unlocked) funds back to master account via `POST /v1/admin/api-keys/:id/sweep` — this is signed by the service's signing key since it's a signer on the sponsor account
- Locked reserves remain until sponsored entries are removed by users over time

---

## 13. Future Considerations

These are explicitly out of scope for v1 but worth considering for future versions:

- **Webhooks**: Notify wallets when their sponsor account balance is low or API key is expiring
- **HSM/KMS signing**: Hardware security module or cloud KMS integration for signing key security
- **Fee sponsorship**: Optionally sponsor transaction fees in addition to base reserves
- **Allowlist/blocklist for destination accounts**: Control which accounts can be sponsored
- **Batch signing**: Accept multiple transactions in a single API call
- **SDK/Client libraries**: Provide client libraries for common languages (JavaScript, Python, Go, etc.)
- **Audit log**: Immutable audit trail of all administrative actions
- **Multisig master account**: Require multiple signatures for funding operations
- **Auto-fund thresholds**: Automatically top up sponsor accounts when balance drops below a threshold
