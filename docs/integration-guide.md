# Sponsorship API — Integration Guide

This guide is for wallet developers and backend services that want to integrate with the Sponsorship API to cover Stellar reserve costs on behalf of their users.

---

## Overview

The Sponsorship API lets you build unsigned Stellar transactions that use [sponsored reserves](https://developers.stellar.org/docs/learn/encyclopedia/transactions-specialized/sponsored-reserves) and submit them for co-signing. The sponsor's key signs the `BEGIN_SPONSORING_FUTURE_RESERVES` operation so your users never need to lock XLM for reserves.

Your application is responsible for:

1. Building the unsigned transaction (client-side or server-side)
2. Sending the unsigned XDR to the API for sponsor co-signing
3. Signing the transaction with the user's key
4. Submitting the fully-signed transaction to the Stellar network

The API is responsible for:

- Validating that the transaction follows sponsorship rules
- Co-signing with the sponsor account's key
- Enforcing budget limits, rate limits, and operation allowlists
- Logging all signing requests for audit

---

## Authentication

All authenticated endpoints require a Bearer token:

```
Authorization: Bearer <api-key>
```

API keys are provisioned by an admin. Each key is scoped to:

- A **dedicated sponsor account** with a funded XLM budget
- An **allowed operations list** (e.g. `CREATE_ACCOUNT`, `CHANGE_TRUST`)
- A **rate limit** (max requests per time window)
- An optional **source account allowlist** to restrict which Stellar accounts can submit transactions
- An **expiration date**

Keep API keys server-side. Never expose them in client-side code or browser requests. If your frontend needs to call the API, proxy requests through your backend and inject the `Authorization` header there.

### Rate Limit Headers

Authenticated responses include rate limit headers:

| Header                  | Description                              |
| ----------------------- | ---------------------------------------- |
| `X-RateLimit-Limit`     | Max requests allowed per window          |
| `X-RateLimit-Remaining` | Requests remaining in the current window |
| `X-RateLimit-Reset`     | Unix timestamp when the window resets    |

---

## Endpoints

### `GET /v1/info`

Returns the network configuration of the sponsorship service. No authentication required.

**Response:**

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

Use this on startup to verify your app and the sponsorship service are on the same Stellar network.

---

### `GET /v1/usage`

Returns budget, rate limit, and status information for your API key.

**Response:**

```json
{
  "api_key_name": "my-wallet",
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

| Field                    | Description                                                                               |
| ------------------------ | ----------------------------------------------------------------------------------------- |
| `sponsor_account`        | Public key of the sponsor. Use this as the source for `BEGIN_SPONSORING_FUTURE_RESERVES`. |
| `xlm_budget`             | Total XLM allocated to this API key                                                       |
| `xlm_available`          | XLM remaining after subtracting locked reserves                                           |
| `xlm_locked_in_reserves` | XLM currently locked in on-chain sponsorships                                             |
| `allowed_operations`     | Operations the API will co-sign (excluding structural `BEGIN/END_SPONSORING`)             |
| `is_active`              | Whether the key is active and can sign transactions                                       |
| `rate_limit.remaining`   | Requests left before hitting the rate limit                                               |

---

### `POST /v1/sign`

Validates and co-signs an unsigned transaction with the sponsor's key.

**Request:**

```json
{
  "transaction_xdr": "<unsigned-base64-xdr>",
  "network_passphrase": "Test SDF Network ; September 2015"
}
```

Both fields are required. The `network_passphrase` must match the service's configured network.

**Response (200):**

```json
{
  "signed_transaction_xdr": "<sponsor-signed-base64-xdr>",
  "sponsor_public_key": "GABCD...",
  "sponsor_account_balance": "950.0000000"
}
```

The returned XDR has the sponsor's signature attached. You still need to add the user's signature (and any other required signatures) before submitting to the network.

---

## Transaction Structure

Every sponsored operation must be wrapped in a `BEGIN_SPONSORING` / `END_SPONSORING` block. The API enforces this — transactions without proper sponsorship blocks are rejected.

### Sponsored Trustline

Adds a trustline where the sponsor covers the 0.5 XLM reserve.

| #   | Operation                          | Source          | Notes                          |
| --- | ---------------------------------- | --------------- | ------------------------------ |
| 1   | `BEGIN_SPONSORING_FUTURE_RESERVES` | Sponsor account | `sponsoredId` = user's account |
| 2   | `CHANGE_TRUST`                     | User account    | The asset to trust             |
| 3   | `END_SPONSORING_FUTURE_RESERVES`   | User account    | Closes the block               |

**Transaction source:** User account

**Signatures required:** Sponsor (from the API) + User

### Sponsored Account Creation

Creates a new Stellar account where the sponsor covers the 2 base reserves (1 XLM).

| #   | Operation                          | Source          | Notes                                               |
| --- | ---------------------------------- | --------------- | --------------------------------------------------- |
| 1   | `BEGIN_SPONSORING_FUTURE_RESERVES` | Sponsor account | `sponsoredId` = new account                         |
| 2   | `CREATE_ACCOUNT`                   | User account    | `startingBalance` = `"0"`                           |
| 3   | `END_SPONSORING_FUTURE_RESERVES`   | New account     | Stellar allows this within the creating transaction |

**Transaction source:** User account (an existing funded account)

**Signatures required:** Sponsor (from the API) + User + New account

### Other Supported Operations

The same `BEGIN_SPONSORING` / `END_SPONSORING` pattern applies to all supported operations:

| Operation                  | Reserves Locked | Notes                                     |
| -------------------------- | --------------- | ----------------------------------------- |
| `CREATE_ACCOUNT`           | 2               | Base reserves for a new account           |
| `CHANGE_TRUST`             | 1               | Trustline entry (0 if removing)           |
| `MANAGE_SELL_OFFER`        | 1               | New offer only (0 if updating/deleting)   |
| `MANAGE_BUY_OFFER`         | 1               | New offer only (0 if updating/deleting)   |
| `SET_OPTIONS`              | 1               | Only when adding a signer (0 otherwise)   |
| `MANAGE_DATA`              | 1               | Only when setting a value (0 if deleting) |
| `CREATE_CLAIMABLE_BALANCE` | 1               | Claimable balance entry                   |

### Multiple Operations

You can include multiple sponsored operations in a single transaction. Each operation must be inside its own `BEGIN/END` block:

```
Op 1: BEGIN_SPONSORING (sponsor → user)
Op 2: CHANGE_TRUST USDC  (user)
Op 3: END_SPONSORING      (user)
Op 4: BEGIN_SPONSORING (sponsor → user)
Op 5: CHANGE_TRUST EURC  (user)
Op 6: END_SPONSORING      (user)
```

---

## Signing Flow

```
Your Backend                         Sponsorship API                Stellar Network
    |                                      |                              |
    |  1. GET /v1/usage (once, on startup) |                              |
    |  ──────────────────────────────────> |                              |
    |  <── sponsor_account, budget ─────── |                              |
    |     (cache sponsor_account)          |                              |
    |                                      |                              |
    |  2. Build unsigned XDR               |                              |
    |     (using cached sponsor_account    |                              |
    |      in BEGIN_SPONSORING)            |                              |
    |                                      |                              |
    |  3. POST /v1/sign                    |                              |
    |     { transaction_xdr, passphrase }  |                              |
    |  ──────────────────────────────────> |                              |
    |                                      |  Validate operations         |
    |                                      |  Check budget                |
    |                                      |  Co-sign with sponsor key    |
    |  <── signed_transaction_xdr ──────── |                              |
    |                                      |                              |
    |  4. Add user signature(s)            |                              |
    |                                      |                              |
    |  5. Submit to Horizon ───────────────────────────────────────────> |
    |  <── transaction result ─────────────────────────────────────────  |
```

### Step by Step

1. **Fetch and cache sponsor info** — Call `GET /v1/usage` once on startup (or periodically in the background) to get the `sponsor_account` public key. Cache this value — it doesn't change for the lifetime of the API key. There's no need to call this before every signing request.

2. **Build the unsigned transaction** — Construct a Stellar transaction with the user's account as the transaction source. Include `BEGIN_SPONSORING_FUTURE_RESERVES` (source: sponsor account), the target operation (source: user account), and `END_SPONSORING_FUTURE_RESERVES` (source: user account). Set a reasonable timeout (e.g. 300 seconds).

3. **Get the sponsor signature** — Send the unsigned XDR to `POST /v1/sign` along with the `network_passphrase`. The API validates the transaction and returns it with the sponsor's signature attached.

4. **Add the user's signature** — Sign the returned XDR with the user's secret key. For `CREATE_ACCOUNT` operations, also sign with the new account's key.

5. **Submit to the network** — Submit the fully-signed XDR to Horizon. The user's account pays the transaction fee (typically 100 stroops / 0.00001 XLM). The reserves are locked against the sponsor account, not the user's.

---

## Validation Rules

The API rejects transactions that violate any of the following rules:

| Rule                          | Error Code              | Description                                                                                                |
| ----------------------------- | ----------------------- | ---------------------------------------------------------------------------------------------------------- |
| Invalid XDR                   | `invalid_transaction`   | The XDR cannot be parsed or is not a V1 transaction envelope                                               |
| Sponsor as transaction source | `sponsor_as_source`     | The transaction source account must not be the sponsor account                                             |
| Sponsor as operation source   | `sponsor_as_source`     | Only `BEGIN_SPONSORING_FUTURE_RESERVES` may use the sponsor as its operation source                        |
| Disallowed operation          | `disallowed_operation`  | The operation type is not in the API key's allowed list                                                    |
| XLM transfer                  | `xlm_transfer_detected` | Transactions cannot transfer native XLM (PAYMENT, PATH_PAYMENT, ACCOUNT_MERGE, INFLATION, native CLAWBACK) |
| Missing sponsorship block     | `invalid_transaction`   | Non-structural operations must be wrapped in `BEGIN/END_SPONSORING`                                        |
| Unmatched blocks              | `invalid_transaction`   | Every `BEGIN_SPONSORING` must have a matching `END_SPONSORING`                                             |
| Source mismatch in block      | `invalid_transaction`   | The operation source must match the `sponsoredId` of the enclosing `BEGIN_SPONSORING`                      |
| Wrong sponsor in BEGIN        | `invalid_sponsor`       | The source of `BEGIN_SPONSORING` must be the API key's sponsor account                                     |
| Source account not allowed    | `disallowed_operation`  | The source account is not in the API key's allowlist (if configured)                                       |
| Insufficient balance          | `insufficient_balance`  | The sponsor account does not have enough XLM to cover the required reserves                                |
| Network mismatch              | `invalid_network`       | The `network_passphrase` does not match the service's configured network                                   |

---

## Error Responses

All errors follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable description of what went wrong"
}
```

### HTTP Status Codes

| Status | Meaning                                                                     |
| ------ | --------------------------------------------------------------------------- |
| `400`  | Bad request — invalid transaction, disallowed operation, validation failure |
| `401`  | Unauthorized — missing or invalid API key                                   |
| `403`  | Forbidden — API key is revoked or expired                                   |
| `429`  | Rate limited — too many requests in the current window                      |
| `500`  | Internal error — unexpected server failure                                  |
| `502`  | Bad gateway — upstream Stellar Horizon issue                                |
| `503`  | Service unavailable — unable to verify sponsor balance                      |

---

## Handling Failures and Fallbacks

In production, the sponsorship API may become temporarily unavailable or the sponsor account may run out of funds. Your application should handle these scenarios gracefully so users can still complete onboarding.

### When sponsorship can fail

- **API downtime** — The sponsorship service is unreachable or returning 5xx errors.
- **Budget exhausted** — The sponsor account's XLM is fully locked in existing reserves (`insufficient_balance`).
- **Rate limit hit** — The API key has exceeded its request quota (`429`).
- **API key expired or revoked** — The key is no longer active (`401`/`403`).

### Recommended fallback: your own sponsorship account

When the API is unavailable, fall back to sponsoring with your application's own Stellar account. The transaction structure stays the same — you just use your own account as the sponsor and sign with your own key instead of calling the API.

This means your application should have:

- A **fallback Stellar account** funded with XLM, controlled by your backend
- The ability to build and sign sponsorship transactions directly (the same `BEGIN_SPONSORING` / `END_SPONSORING` pattern)

The fallback flow:

1. `POST /v1/sign` fails (5xx, `insufficient_balance`, `429`, `401`, etc.)
2. Rebuild the transaction using your fallback account as the sponsor in `BEGIN_SPONSORING_FUTURE_RESERVES`
3. Sign with your fallback account's key on your backend
4. Have the user sign as usual
5. Submit to the network

This keeps the user experience identical — users never need to hold XLM for reserves, even when the API is down.

### When you don't have a fallback account

If maintaining your own sponsorship account isn't practical, **queue and retry** — hold the operation and retry with the API once it recovers. Show the user a pending state. This works well for non-urgent operations like adding trustlines.

### Monitor and alert

Track sponsorship API failures on your side. Set up alerts for:

- Sustained 5xx responses or network errors from `POST /v1/sign`
- `insufficient_balance` errors (sponsor budget running low)
- `429` responses (rate limit consistently hit)
- `401`/`403` responses (API key expiring or revoked)

### Summary

| Scenario            | Detection                              | Fallback                                                                                                                     |
| ------------------- | -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| API unreachable     | Network error or 5xx on `/v1/sign`     | Sponsor with your own account                                                                                                |
| Budget exhausted    | `insufficient_balance` from `/v1/sign` | Sponsor with your own account                                                                                                |
| Rate limited        | `429` from `/v1/sign`                  | Short retry with backoff, then sponsor with your own account, check sponsorship service admin for rate increase if necessary |
| Key expired/revoked | `401`/`403` from `/v1/sign`            | Sponsor with your own account, alert sponsorship service admin for new key                                                   |
