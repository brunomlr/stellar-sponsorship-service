# Wallet Demo

A Next.js app that demonstrates how to integrate with the Sponsorship Service API. It lets you generate or import a Stellar keypair and perform sponsored operations — adding trustlines and creating accounts — without the end-user paying for reserves.

## Prerequisites

- Node.js 18+
- The [Sponsorship Service](../README.md) running locally (default `http://localhost:8080`)
- An active API key from the sponsorship service

## Setup

1. Install dependencies:

```bash
npm install
```

2. Configure environment variables in `.env`:

```env
NEXT_PUBLIC_STELLAR_NETWORK=testnet                          # "testnet" or "mainnet"
SPONSORSHIP_API_URL=http://localhost:8080                     # Go API server URL (server-side only)
SPONSORSHIP_API_KEY=sk_test_...                               # Your API key
NEXT_PUBLIC_HORIZON_URL=https://horizon-testnet.stellar.org   # Horizon server URL
```

3. Start the dev server:

```bash
npm run dev
```

The app runs on [http://localhost:3001](http://localhost:3001).

## How to Use

### 1. Connect a Wallet

Click **Connect Wallet** and choose one of:

- **Generate New Keypair** — creates a random Stellar keypair in-browser. Copy and save the secret key, it won't be stored anywhere.
- **Import Secret Key** — paste an existing Stellar secret key to load that account.

### 2. Fund Your Account (Testnet)

If you're on testnet and the account doesn't exist on-chain yet, click **Fund with Friendbot** in the Account Info panel. This gives you a small amount of testnet XLM so the account can sign transactions.

### 3. Add a Trustline (Sponsored)

1. Go to the **Add Trustline** tab.
2. Pick a preset asset (e.g. USDC, SRT) or enter a custom asset code and issuer.
3. Click **Add Trustline**.

The demo builds an unsigned transaction, sends it to the sponsorship API for co-signing, signs it with your keypair, and submits it to the network. The sponsor covers the 0.5 XLM reserve so your account doesn't need to lock any XLM.

### 4. Create an Account (Sponsored)

1. Go to the **Create Account** tab.
2. Generate a new keypair for the account to be created, or paste an existing public key.
3. Click **Create Account**.

The sponsor covers the 1 XLM base reserve for the new account. The new account is created with a 0 XLM starting balance.

### 5. Transaction History

All operations appear in the **Transaction History** section at the bottom. Each entry shows its status and links to the Stellar Explorer once confirmed.

---

## How the Sponsorship API Works

The wallet-demo proxies all requests through Next.js API routes (`src/app/api/*`) so the API key stays server-side and is never exposed to the browser.

### Authentication

The Sponsorship API uses **Bearer token authentication** with API keys. Every request to `/v1/sign` and `/v1/usage` must include the header:

```
Authorization: Bearer <api-key>
```

The `/v1/info` endpoint is public and requires no authentication.

API keys are created and managed by an admin through the service's admin panel. Each key is tied to a dedicated sponsor account and defines a budget, allowed operations, rate limits, and an optional source-account allowlist.

**How the demo handles it:** The API key is stored in the `SPONSORSHIP_API_KEY` server-side environment variable and is never sent to the browser. The demo's Next.js API routes (`src/app/api/*`) act as a proxy — the browser calls `/api/sign`, and the server-side route injects the `Authorization` header before forwarding to the Go API at `/v1/sign`. This keeps the key secret from end-users.

```
Browser                      Next.js Server                   Sponsorship API
   |                              |                                 |
   |  POST /api/sign              |                                 |
   |  (no auth header)            |                                 |
   |  ──────────────────────────> |                                 |
   |                              |  POST /v1/sign                  |
   |                              |  Authorization: Bearer sk_...   |
   |                              |  ──────────────────────────────>|
   |                              |  <──────────────────────────────|
   |  <────────────────────────── |                                 |
```

**Direct API integration (no proxy):** If you're calling the Sponsorship API from your own backend, skip the proxy and pass the API key directly:

```bash
curl -X POST https://your-sponsorship-api/v1/sign \
  -H "Authorization: Bearer sk_test_..." \
  -H "Content-Type: application/json" \
  -d '{"transaction_xdr": "<unsigned-xdr>"}'
```

### Endpoints

#### `GET /v1/info`

Returns network configuration. No authentication required.

**Response:**

```json
{
  "network_passphrase": "Test SDF Network ; September 2015",
  "base_reserve": "0.5",
  "supported_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"]
}
```

#### `GET /v1/usage`

Returns budget and rate limit information for the authenticated API key.

**Headers:** `Authorization: Bearer <api-key>`

**Response:**

```json
{
  "api_key_name": "my-wallet-app",
  "sponsor_account": "GABCD...",
  "xlm_budget": "1000.0000000",
  "xlm_available": "950.0000000",
  "xlm_locked_in_reserves": "50.0000000",
  "allowed_operations": ["CREATE_ACCOUNT", "CHANGE_TRUST"],
  "expires_at": "2026-12-31T23:59:59Z",
  "is_active": true,
  "transactions_signed": 42,
  "rate_limit": {
    "max_requests": 100,
    "window_seconds": 60,
    "remaining": 97
  }
}
```

#### `POST /v1/sign`

Validates and co-signs an unsigned transaction XDR with the sponsor's key.

**Headers:** `Authorization: Bearer <api-key>`

**Request:**

```json
{
  "transaction_xdr": "<unsigned-xdr-string>"
}
```

**Response:**

```json
{
  "signed_transaction_xdr": "<sponsor-signed-xdr>",
  "sponsor_public_key": "GABCD...",
  "sponsor_account_balance": "950.0000000"
}
```

### Signing Flow

The full lifecycle of a sponsored transaction:

```
Wallet (browser)                    Sponsorship API                 Stellar Network
      |                                   |                               |
      |  1. Build unsigned XDR            |                               |
      |  (BEGIN_SPONSORING +              |                               |
      |   operation + END_SPONSORING)     |                               |
      |                                   |                               |
      |  2. POST /v1/sign ───────────────>|                               |
      |                                   |  Validate operations          |
      |                                   |  Check budget & rate limit    |
      |                                   |  Co-sign with sponsor key     |
      |  <──────── signed XDR ────────────|                               |
      |                                   |                               |
      |  3. Sign with user keypair        |                               |
      |                                   |                               |
      |  4. Submit fully-signed XDR ──────────────────────────────────────>|
      |  <──────── transaction hash ──────────────────────────────────────|
```

### Transaction Structure

Every sponsored operation follows the same pattern of three operations wrapped in a single transaction:

**Add Trustline:**

| # | Operation | Source | Description |
|---|-----------|--------|-------------|
| 1 | `BEGIN_SPONSORING_FUTURE_RESERVES` | Sponsor | Sponsor agrees to cover reserves for the user |
| 2 | `CHANGE_TRUST` | User | Adds the trustline (reserve covered by sponsor) |
| 3 | `END_SPONSORING_FUTURE_RESERVES` | User | Closes the sponsorship block |

**Create Account:**

| # | Operation | Source | Description |
|---|-----------|--------|-------------|
| 1 | `BEGIN_SPONSORING_FUTURE_RESERVES` | Sponsor | Sponsor agrees to cover reserves for the new account |
| 2 | `CREATE_ACCOUNT` | User | Creates the new account with 0 XLM balance |
| 3 | `END_SPONSORING_FUTURE_RESERVES` | New Account | Closes the sponsorship block |

### Validation Rules

The API rejects transactions that:

- Contain operations not in the API key's allowed list
- Include XLM transfers (PAYMENT, PATH_PAYMENT, ACCOUNT_MERGE)
- Use the sponsor account as the transaction source
- Have improperly nested sponsorship blocks
- Would exceed the sponsor's remaining budget
- Come from a source account not in the allowlist (if configured)

### Error Responses

All errors follow this shape:

```json
{
  "error": "error_code",
  "message": "Human-readable description"
}
```

Common error codes:

| Code | Description |
|------|-------------|
| `invalid_xdr` | Transaction XDR could not be parsed |
| `operation_not_allowed` | Transaction contains a disallowed operation type |
| `budget_exceeded` | Sponsor doesn't have enough XLM to cover reserves |
| `rate_limit_exceeded` | Too many requests in the current window |
| `unauthorized` | Missing or invalid API key |

---

## Project Structure

```
src/
├── app/
│   ├── page.tsx                    # Main demo page
│   ├── layout.tsx                  # Root layout
│   └── api/                        # Next.js API routes (proxy to Go API)
│       ├── info/route.ts           # GET  /api/info  → /v1/info
│       ├── usage/route.ts          # GET  /api/usage → /v1/usage
│       └── sign/route.ts           # POST /api/sign  → /v1/sign
├── components/
│   ├── connect-wallet-button.tsx   # Generate/import keypair UI
│   ├── account-info.tsx            # Account balance and trustlines
│   ├── sponsor-info.tsx            # Sponsor budget and rate limits
│   ├── add-trustline-form.tsx      # Add trustline form + transaction flow
│   ├── create-account-form.tsx     # Create account form + transaction flow
│   ├── transaction-history.tsx     # Past transaction list
│   ├── transaction-status.tsx      # Step-by-step progress indicator
│   └── providers.tsx               # React Query + Wallet context
├── hooks/
│   ├── use-wallet.ts               # Wallet state (connect, sign, disconnect)
│   └── use-sponsor.ts              # Sponsor info & usage queries
├── lib/
│   ├── api.ts                      # Client-side API calls
│   ├── transaction-builder.ts      # Build unsigned Stellar transactions
│   ├── stellar.ts                  # Network config helpers
│   └── utils.ts                    # Formatting, explorer URLs
└── types/
    └── index.ts                    # TypeScript interfaces for API responses
```
