# Stellar Sponsorship Service — Technical Specification

> Implementation spec for the [PRD v2](./prd-v2.md). Tech stack: Go (API), Next.js (Admin Dashboard), PostgreSQL.

---

## 1. Project Structure

Monorepo with two main components:

```
sponsorship-service/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point
├── internal/
│   ├── config/
│   │   └── config.go                # Environment config loading
│   ├── server/
│   │   ├── server.go                # HTTP server setup, middleware
│   │   └── routes.go                # Route registration
│   ├── handler/
│   │   ├── sign.go                  # POST /v1/sign
│   │   ├── info.go                  # GET /v1/info
│   │   ├── usage.go                 # GET /v1/usage
│   │   ├── health.go                # GET /v1/health
│   │   └── admin/
│   │       ├── apikeys.go           # API key CRUD handlers
│   │       ├── fund.go              # Fund/activate handlers
│   │       ├── sweep.go             # Sweep handler
│   │       └── transactions.go      # Transaction logs handler
│   ├── middleware/
│   │   ├── auth.go                  # API key authentication
│   │   ├── admin_auth.go            # Admin token authentication
│   │   └── ratelimit.go             # Per-key rate limiting
│   ├── stellar/
│   │   ├── signer.go                # Transaction signing with signing key
│   │   ├── verifier.go              # Transaction verification rules (7.1–7.5)
│   │   ├── account.go               # Account queries via Horizon
│   │   ├── builder.go               # Build funding/sweep transactions
│   │   └── operations.go            # Operation type parsing and validation
│   ├── store/
│   │   ├── store.go                 # Store interface
│   │   ├── postgres.go              # PostgreSQL implementation
│   │   ├── apikeys.go               # API key queries
│   │   └── transactions.go          # Transaction log queries
│   └── model/
│       ├── apikey.go                # API key model
│       └── transaction.go           # Transaction log model
├── migrations/
│   ├── 001_create_api_keys.up.sql
│   ├── 001_create_api_keys.down.sql
│   ├── 002_create_transaction_logs.up.sql
│   └── 002_create_transaction_logs.down.sql
├── dashboard/                        # Next.js admin dashboard
│   ├── package.json
│   ├── next.config.js
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx              # Dashboard overview
│   │   │   ├── api-keys/
│   │   │   │   ├── page.tsx          # API keys list
│   │   │   │   ├── new/
│   │   │   │   │   └── page.tsx      # Create API key
│   │   │   │   └── [id]/
│   │   │   │       └── page.tsx      # API key detail
│   │   │   ├── transactions/
│   │   │   │   └── page.tsx          # Transaction logs
│   │   │   └── settings/
│   │   │       └── page.tsx          # Settings
│   │   ├── components/
│   │   │   ├── wallet-provider.tsx   # Stellar Wallets Kit provider
│   │   │   ├── sign-transaction.tsx  # Freighter signing component
│   │   │   ├── api-key-table.tsx
│   │   │   ├── create-key-form.tsx
│   │   │   ├── fund-form.tsx
│   │   │   ├── transaction-table.tsx
│   │   │   └── charts/
│   │   │       ├── signing-volume.tsx
│   │   │       └── xlm-usage.tsx
│   │   ├── lib/
│   │   │   ├── api.ts               # Admin API client
│   │   │   └── stellar.ts           # Stellar Wallets Kit setup
│   │   └── hooks/
│   │       ├── use-wallet.ts         # Wallet connection hook
│   │       └── use-api-keys.ts       # API keys data hook
│   └── tsconfig.json
├── docker/
│   ├── Dockerfile                    # Multi-stage: Go API
│   ├── Dockerfile.dashboard          # Next.js dashboard
│   └── docker-compose.yml            # Local dev: API + Dashboard + PostgreSQL
├── go.mod
├── go.sum
└── Makefile
```

---

## 2. Dependencies

### Go (API Server)

| Package | Purpose |
|---------|---------|
| `github.com/stellar/go` (monorepo) | Stellar SDK — XDR parsing (`xdr`), transaction building (`txnbuild`), signing (`keypair`), Horizon client (`clients/horizonclient`), amount conversion (`amount`), network passphrases (`network`) |
| `github.com/go-chi/chi/v5` | HTTP router — lightweight, idiomatic, middleware-friendly |
| `github.com/jackc/pgx/v5` | PostgreSQL driver — high performance, native Go |
| `github.com/golang-migrate/migrate/v4` | Database migrations |
| `github.com/rs/zerolog` | Structured JSON logging |
| `github.com/prometheus/client_golang` | Prometheus metrics |
| `github.com/sethvargo/go-envconfig` | Environment variable config parsing |
| `github.com/google/uuid` | UUID generation |

### Dashboard (Next.js)

| Package | Purpose |
|---------|---------|
| `@stellar/stellar-sdk` | Stellar SDK for building transactions client-side |
| `@creit.tech/stellar-wallets-kit` | Unified wallet interface (Freighter, etc.) |
| `@tanstack/react-query` | Server state management, API data fetching |
| `recharts` | Charts for dashboard analytics |
| `shadcn/ui` | UI component library |
| `tailwindcss` | Styling |

---

## 3. Database Schema

### Migration 001: API Keys

```sql
CREATE TYPE api_key_status AS ENUM ('pending_funding', 'active', 'revoked');

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    key_hash        VARCHAR(64) NOT NULL UNIQUE,    -- SHA-256 hex
    key_prefix      VARCHAR(20) NOT NULL,            -- e.g. "sk_live_abc1"
    sponsor_account VARCHAR(56) NOT NULL UNIQUE,     -- Stellar public key
    xlm_budget      BIGINT NOT NULL,                 -- initial funding in stroops
    allowed_operations JSONB NOT NULL DEFAULT '[]',
    allowed_source_accounts JSONB DEFAULT NULL,
    rate_limit_max  INTEGER NOT NULL DEFAULT 100,
    rate_limit_window INTEGER NOT NULL DEFAULT 60,   -- seconds
    status          api_key_status NOT NULL DEFAULT 'pending_funding',
    funding_tx_xdr  TEXT,                            -- unsigned funding transaction
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_status ON api_keys (status);
CREATE INDEX idx_api_keys_sponsor_account ON api_keys (sponsor_account);
```

### Migration 002: Transaction Logs

```sql
CREATE TYPE transaction_status AS ENUM ('signed', 'rejected');

CREATE TABLE transaction_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id        UUID NOT NULL REFERENCES api_keys(id),
    transaction_hash  VARCHAR(64),                    -- null if rejected
    transaction_xdr   TEXT NOT NULL,
    operations        JSONB NOT NULL DEFAULT '[]',
    source_account    VARCHAR(56) NOT NULL,
    status            transaction_status NOT NULL,
    rejection_reason  VARCHAR(255),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transaction_logs_api_key_id ON transaction_logs (api_key_id);
CREATE INDEX idx_transaction_logs_status ON transaction_logs (status);
CREATE INDEX idx_transaction_logs_created_at ON transaction_logs (created_at);
```

---

## 4. Configuration

```go
// internal/config/config.go

type Config struct {
    StellarNetwork        string `env:"STELLAR_NETWORK,required"`         // "testnet" or "mainnet"
    SigningSecretKey       string `env:"SIGNING_SECRET_KEY,required"`      // Stellar secret key (S...)
    MasterFundingPublicKey string `env:"MASTER_FUNDING_PUBLIC_KEY,required"` // Stellar public key (G...)
    DatabaseURL           string `env:"DATABASE_URL,required"`
    AdminAPIKey           string `env:"ADMIN_API_KEY,required"`
    Port                  int    `env:"PORT,default=8080"`
    HorizonURL            string `env:"HORIZON_URL"`                      // defaults based on network
    LogLevel              string `env:"LOG_LEVEL,default=info"`
}

func (c *Config) NetworkPassphrase() string {
    if c.StellarNetwork == "mainnet" {
        return network.PublicNetworkPassphrase
    }
    return network.TestNetworkPassphrase
}

func (c *Config) DefaultHorizonURL() string {
    if c.HorizonURL != "" {
        return c.HorizonURL
    }
    if c.StellarNetwork == "mainnet" {
        return "https://horizon.stellar.org"
    }
    return "https://horizon-testnet.stellar.org"
}
```

---

## 5. Core Implementation

### 5.1 Transaction Verifier

The verifier is the most critical component. It implements the rules from PRD section 7.

```go
// internal/stellar/verifier.go

type VerifyResult struct {
    Valid           bool
    HTTPStatus      int       // HTTP status code for error responses (400, 401, 403, etc.)
    ErrorCode       string
    ErrorMessage    string
    Operations      []string  // operation type names found (excluding BEGIN/END_SPONSORING structural ops)
    SourceAccount   string    // transaction source account (for logging)
}

type Verifier struct {
    networkPassphrase string
}

func (v *Verifier) Verify(txXDR string, apiKey *model.APIKey) VerifyResult
```

**Verification steps (in order):**

1. **Decode XDR**: Parse the transaction envelope from base64 XDR using `txnbuild.TransactionFromXDR()`. Return `invalid_transaction` (HTTP 400) if decoding fails. Only V1 transaction envelopes are supported (not fee bump transactions).

2. **Network passphrase**: Verify the transaction was built for the correct network. The network passphrase is embedded in the transaction hash, so this is validated implicitly during signing.

3. **Source account check**: Extract the transaction source account. If it matches `apiKey.SponsorAccount`, return `sponsor_as_source` (HTTP 400).

4. **Operation iteration**: For each operation in the transaction:

   a. **Operation source check**: If the operation has an explicit source account and it matches `apiKey.SponsorAccount`, return `sponsor_as_source` (HTTP 400). **Exception**: `BEGIN_SPONSORING_FUTURE_RESERVES` operations where the sponsor account is the source are expected — the sponsor account is the source of this operation by design (it's the one doing the sponsoring).

   b. **Structural operations**: `BEGIN_SPONSORING_FUTURE_RESERVES` and `END_SPONSORING_FUTURE_RESERVES` are **structural operations** that control sponsoring blocks. They are **always allowed** regardless of `apiKey.AllowedOperations` and are **not** included in the `Operations` list in the result. However, the sponsor in `BEGIN_SPONSORING_FUTURE_RESERVES` must match `apiKey.SponsorAccount` — return `invalid_sponsor` (HTTP 400) if it doesn't.

   c. **Operation type check**: For non-structural operations, map the XDR operation type to a string name. Reject with `disallowed_operation` (HTTP 400) if not in `apiKey.AllowedOperations`.

   d. **XLM transfer check**: Reject these operation types unconditionally with `xlm_transfer_detected` (HTTP 400):
      - `OperationTypePayment` where asset type is native
      - `OperationTypePathPaymentStrictSend` where dest asset is native
      - `OperationTypePathPaymentStrictReceive` where dest asset is native
      - `OperationTypeAccountMerge`
      - `OperationTypeInflation`
      - `OperationTypeClawback` where asset is native

   e. **Sponsoring block check**: Verify that sponsorable operations are wrapped in `BEGIN_SPONSORING_FUTURE_RESERVES` / `END_SPONSORING_FUTURE_RESERVES` pairs. The blocks must be properly nested: each `BEGIN` must have a matching `END`, and every sponsorable operation must be inside a block. Return `invalid_transaction` (HTTP 400) if blocks are malformed.

   f. **Source account allowlist**: If `apiKey.AllowedSourceAccounts` is set, verify the transaction source and each non-structural operation source (or inherited source) is in the list. The sponsor account is **not** checked against this list since it appears as source only in `BEGIN_SPONSORING_FUTURE_RESERVES`.

**Operation type mapping:**

```go
// internal/stellar/operations.go

var operationTypeNames = map[xdr.OperationType]string{
    xdr.OperationTypeCreateAccount:            "CREATE_ACCOUNT",
    xdr.OperationTypeChangeTrust:              "CHANGE_TRUST",
    xdr.OperationTypeManageSellOffer:          "MANAGE_SELL_OFFER",
    xdr.OperationTypeManageBuyOffer:           "MANAGE_BUY_OFFER",
    xdr.OperationTypeSetOptions:               "SET_OPTIONS",
    xdr.OperationTypeManageData:               "MANAGE_DATA",
    xdr.OperationTypeCreateClaimableBalance:   "CREATE_CLAIMABLE_BALANCE",
    xdr.OperationTypeBeginSponsoringFutureReserves: "BEGIN_SPONSORING_FUTURE_RESERVES",
    xdr.OperationTypeEndSponsoringFutureReserves:   "END_SPONSORING_FUTURE_RESERVES",
}
```

### 5.2 Transaction Signer

```go
// internal/stellar/signer.go

type Signer struct {
    signingKey    *keypair.Full
    networkPassphrase string
}

func NewSigner(secretKey string, networkPassphrase string) (*Signer, error) {
    kp, err := keypair.ParseFull(secretKey)
    if err != nil {
        return nil, fmt.Errorf("invalid signing key: %w", err)
    }
    return &Signer{signingKey: kp, networkPassphrase: networkPassphrase}, nil
}

// PublicKey returns the public key (G...) of the signing key.
func (s *Signer) PublicKey() string {
    return s.signingKey.Address()
}

// Sign adds the signing key's signature to the transaction envelope.
// The transaction must already be verified by the Verifier.
// Returns: (signedXDR, transactionHashHex, error)
func (s *Signer) Sign(txXDR string) (string, string, error) {
    // 1. Parse transaction from XDR using txnbuild.TransactionFromXDR()
    // 2. Extract the *Transaction (reject fee bump transactions)
    // 3. Sign with s.signingKey: tx.Sign(s.networkPassphrase, s.signingKey)
    //    IMPORTANT: Sign() returns a new *Transaction — must reassign
    // 4. Get the transaction hash: tx.HashHex(s.networkPassphrase)
    // 5. Get signed XDR: tx.Base64()
    // 6. Return signedXDR, hashHex, nil
}
```

### 5.3 Transaction Builder

Builds unsigned transactions for admin operations (account creation, funding, sweeping).

```go
// internal/stellar/builder.go

type Builder struct {
    horizonClient    *horizonclient.Client
    signingPublicKey string
    masterPublicKey  string
    networkPassphrase string
}

// BuildCreateSponsorAccount builds an unsigned transaction that:
// 1. Creates a new Stellar account funded with xlmBudget
// 2. Sets the signing key as a signer on the new account
// Source account is the master funding account (signed externally via Freighter)
func (b *Builder) BuildCreateSponsorAccount(
    sponsorKeypair *keypair.Full,
    xlmBudget int64,
) (unsignedXDR string, err error) {
    // Load master account from Horizon for sequence number
    masterAccount, err := b.horizonClient.AccountDetail(horizonclient.AccountRequest{
        AccountID: b.masterPublicKey,
    })

    // Build transaction:
    // - Operation 1: CreateAccount (destination=sponsorKeypair.Address(), startingBalance=xlmBudget)
    // - Operation 2: SetOptions on new account (signer=signingPublicKey, weight=1)
    //   This operation needs the sponsor account as source, signed by sponsorKeypair
    //   before its secret key is discarded

    // The transaction source is the master funding account
    // The master account signature will be added by the operator via Freighter
    // The sponsor account signature is added here (for the SetOptions op) before the key is discarded

    tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
        SourceAccount:        &masterAccount,
        IncrementSequenceNum: true,
        BaseFee:              txnbuild.MinBaseFee,
        Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
        Operations: []txnbuild.Operation{
            &txnbuild.CreateAccount{
                Destination: sponsorKeypair.Address(),
                Amount:      amount.StringFromInt64(xlmBudget), // use Stellar SDK amount conversion — no float precision issues
            },
            &txnbuild.SetOptions{
                SourceAccount: sponsorKeypair.Address(),
                Signer: &txnbuild.Signer{
                    Address: b.signingPublicKey,
                    Weight:  txnbuild.Threshold(1),
                },
            },
        },
    })

    // Sign with sponsor keypair (for the SetOptions operation)
    tx, err = tx.Sign(b.networkPassphrase, sponsorKeypair)

    // Return XDR — still needs master account signature (added by operator via Freighter)
    return tx.Base64()
}

// BuildFundTransaction builds an unsigned payment from master to sponsor account
func (b *Builder) BuildFundTransaction(
    sponsorAccount string,
    amount int64,
) (unsignedXDR string, err error)

// BuildAndSubmitSweepTransaction builds a payment from sponsor account back to master,
// signs it with the service's signing key (which is a signer on the sponsor account),
// and submits it to the Stellar network. No external wallet signing is needed.
// Only callable on revoked API keys. Returns the transaction hash and amounts.
func (b *Builder) BuildAndSubmitSweepTransaction(
    signer *Signer,
    sponsorAccount string,
) (txHash string, xlmSwept string, xlmRemainingLocked string, err error)
```

### 5.4 Account Queries

```go
// internal/stellar/account.go

type AccountService struct {
    horizonClient *horizonclient.Client
}

// GetBalance returns the native XLM balance for an account.
// Uses the Stellar SDK's amount package for precise arithmetic (no floats).
func (a *AccountService) GetBalance(accountID string) (available string, locked string, err error) {
    account, err := a.horizonClient.AccountDetail(horizonclient.AccountRequest{
        AccountID: accountID,
    })
    // 1. Find native balance from account.Balances (where asset_type == "native")
    // 2. Parse balance string to stroops using amount.ParseInt64()
    // 3. Calculate minimum balance in stroops:
    //    minBalance = (2 + account.SubentryCount + account.NumSponsoring - account.NumSponsored) * baseReserve
    //    where baseReserve = 5_000_000 stroops (0.5 XLM)
    // 4. locked = minBalance (in stroops) → convert to string with amount.StringFromInt64()
    // 5. available = balance - minBalance → convert to string with amount.StringFromInt64()
    //    If available < 0, return "0.0000000"
}
```

---

## 6. API Handlers

### 6.1 Sign Handler

```go
// internal/handler/sign.go

func (h *SignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Extract API key from context (set by auth middleware)
    apiKey := middleware.GetAPIKey(r.Context())

    // 2. Parse request body (transaction_xdr, network_passphrase)
    var req SignRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 3. Verify transaction
    result := h.verifier.Verify(req.TransactionXDR, apiKey)
    if !result.Valid {
        // Log rejection
        h.store.CreateTransactionLog(r.Context(), &model.TransactionLog{
            APIKeyID:        apiKey.ID,
            TransactionXDR:  req.TransactionXDR,
            Operations:      result.Operations,
            SourceAccount:   result.SourceAccount,
            Status:          model.StatusRejected,
            RejectionReason: result.ErrorMessage,
        })
        // Return error response
        respondError(w, result.HTTPStatus, result.ErrorCode, result.ErrorMessage)
        return
    }

    // 4. Sign transaction
    signedXDR, txHash, err := h.signer.Sign(req.TransactionXDR)

    // 5. Get sponsor account balance
    available, _, err := h.accounts.GetBalance(apiKey.SponsorAccount)

    // 6. Log signed transaction
    h.store.CreateTransactionLog(r.Context(), &model.TransactionLog{
        APIKeyID:        apiKey.ID,
        TransactionHash: txHash,
        TransactionXDR:  signedXDR,
        Operations:      result.Operations,
        SourceAccount:   result.SourceAccount,
        Status:          model.StatusSigned,
    })

    // 7. Respond
    respondJSON(w, http.StatusOK, SignResponse{
        SignedTransactionXDR:   signedXDR,
        SponsorPublicKey:       apiKey.SponsorAccount,
        SponsorAccountBalance:  available,
    })
}
```

### 6.2 Admin: Create API Key Handler

```go
// internal/handler/admin/apikeys.go

func (h *CreateAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req CreateAPIKeyRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 1. Generate API key (prefix varies by network: sk_live_ for mainnet, sk_test_ for testnet)
    rawKey := generateAPIKey(h.config.StellarNetwork)
    keyHash := sha256Hex(rawKey)
    keyPrefix := rawKey[:16] + "..."

    // 2. Generate sponsor account keypair
    sponsorKP, err := keypair.Random()

    // 3. Build unsigned funding transaction
    unsignedXDR, err := h.builder.BuildCreateSponsorAccount(
        sponsorKP,
        req.XLMBudget,  // converted to stroops
    )

    // 4. Store API key with pending_funding status
    apiKey := &model.APIKey{
        Name:                 req.Name,
        KeyHash:              keyHash,
        KeyPrefix:            keyPrefix,
        SponsorAccount:       sponsorKP.Address(),
        XLMBudget:            req.XLMBudget,
        AllowedOperations:    req.AllowedOperations,
        AllowedSourceAccounts: req.AllowedSourceAccounts,
        RateLimitMax:         req.RateLimit.MaxRequests,
        RateLimitWindow:      req.RateLimit.WindowSeconds,
        Status:               model.StatusPendingFunding,
        FundingTxXDR:         unsignedXDR,
        ExpiresAt:            req.ExpiresAt,
    }
    h.store.CreateAPIKey(r.Context(), apiKey)

    // 5. Respond with API key (shown once) and unsigned funding XDR
    respondJSON(w, http.StatusCreated, CreateAPIKeyResponse{
        ID:                   apiKey.ID,
        Name:                 apiKey.Name,
        APIKey:               rawKey,           // only time this is returned
        SponsorAccount:       sponsorKP.Address(),
        XLMBudget:            req.XLMBudget,
        AllowedOperations:    req.AllowedOperations,
        ExpiresAt:            req.ExpiresAt,
        FundingTransactionXDR: unsignedXDR,
        Status:               "pending_funding",
        CreatedAt:            apiKey.CreatedAt,
    })
}
```

### 6.3 Admin: Activate API Key Handler

```go
// internal/handler/admin/fund.go

func (h *ActivateAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    var req ActivateRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 1. Get API key — must be in pending_funding status
    apiKey, err := h.store.GetAPIKeyByID(r.Context(), id)
    if apiKey.Status != model.StatusPendingFunding {
        respondError(w, 400, "invalid_status", "API key is not pending funding")
        return
    }

    // 2. Submit the signed transaction to Stellar
    //    SubmitTransactionXDR returns (hProtocol.Transaction, error)
    //    The transaction hash is in resp.Hash
    resp, err := h.horizonClient.SubmitTransactionXDR(req.SignedTransactionXDR)
    txHash := resp.Hash

    // 3. Update API key status to active
    h.store.UpdateAPIKeyStatus(r.Context(), id, model.StatusActive)

    // 4. Respond
    respondJSON(w, http.StatusOK, ActivateResponse{
        ID:              apiKey.ID,
        Status:          "active",
        SponsorAccount:  apiKey.SponsorAccount,
        TransactionHash: txHash,
    })
}
```

---

## 7. Middleware

### 7.1 API Key Authentication

```go
// internal/middleware/auth.go

func APIKeyAuth(store store.Store) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract Bearer token from Authorization header
            token := extractBearerToken(r)
            if token == "" {
                respondError(w, 401, "invalid_api_key", "Missing API key")
                return
            }

            // 2. Hash the token and look up in database
            keyHash := sha256Hex(token)
            apiKey, err := store.GetAPIKeyByHash(r.Context(), keyHash)
            if err != nil {
                respondError(w, 401, "invalid_api_key", "Invalid API key")
                return
            }

            // 3. Check expiration
            if time.Now().After(apiKey.ExpiresAt) {
                respondError(w, 401, "invalid_api_key", "API key has expired")
                return
            }

            // 4. Check active status
            if apiKey.Status != model.StatusActive {
                respondError(w, 403, "key_disabled", "API key is not active")
                return
            }

            // 5. Set API key in context
            ctx := context.WithValue(r.Context(), apiKeyContextKey, apiKey)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 7.2 Rate Limiting

```go
// internal/middleware/ratelimit.go

// Per-API-key rate limiting using a sliding window counter stored in memory.
// For multi-instance deployments, replace with Redis-based rate limiting.

type RateLimiter struct {
    mu       sync.Mutex
    counters map[string]*window  // keyed by API key ID
}

type window struct {
    count     int
    windowStart time.Time
}

func (rl *RateLimiter) Allow(apiKey *model.APIKey) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    w, exists := rl.counters[apiKey.ID.String()]
    now := time.Now()

    if !exists || now.Sub(w.windowStart) > time.Duration(apiKey.RateLimitWindow)*time.Second {
        rl.counters[apiKey.ID.String()] = &window{count: 1, windowStart: now}
        return true
    }

    if w.count >= apiKey.RateLimitMax {
        return false
    }

    w.count++
    return true
}
```

---

## 8. Routes

```go
// internal/server/routes.go

func (s *Server) routes() {
    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RequestID)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   s.config.CORSOrigins,
        AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
    }))

    // Public endpoints
    r.Get("/v1/info", s.infoHandler.ServeHTTP)
    r.Get("/v1/health", s.healthHandler.ServeHTTP)

    // Wallet endpoints (API key auth)
    r.Group(func(r chi.Router) {
        r.Use(APIKeyAuth(s.store))
        r.Use(RateLimitMiddleware(s.rateLimiter))
        r.Post("/v1/sign", s.signHandler.ServeHTTP)
        r.Get("/v1/usage", s.usageHandler.ServeHTTP)
    })

    // Admin endpoints (admin token auth)
    r.Group(func(r chi.Router) {
        r.Use(AdminAuth(s.config.AdminAPIKey))

        r.Get("/v1/admin/api-keys", s.listAPIKeysHandler.ServeHTTP)
        r.Post("/v1/admin/api-keys", s.createAPIKeyHandler.ServeHTTP)
        r.Patch("/v1/admin/api-keys/{id}", s.updateAPIKeyHandler.ServeHTTP)
        r.Delete("/v1/admin/api-keys/{id}", s.revokeAPIKeyHandler.ServeHTTP)
        r.Post("/v1/admin/api-keys/{id}/activate", s.activateAPIKeyHandler.ServeHTTP)
        r.Post("/v1/admin/api-keys/{id}/fund", s.buildFundHandler.ServeHTTP)
        r.Post("/v1/admin/api-keys/{id}/fund/submit", s.submitFundHandler.ServeHTTP)
        r.Post("/v1/admin/api-keys/{id}/sweep", s.sweepHandler.ServeHTTP)
        r.Get("/v1/admin/transactions", s.transactionsHandler.ServeHTTP)
    })

    // Prometheus metrics
    r.Handle("/metrics", promhttp.Handler())

    s.handler = r
}
```

---

## 9. Admin Dashboard Implementation

### 9.1 Stellar Wallets Kit Integration

```tsx
// dashboard/src/lib/stellar.ts

import {
  StellarWalletsKit,
  WalletNetwork,
  allowAllModules,
  FREIGHTER_ID,
} from "@creit.tech/stellar-wallets-kit";

export function createWalletKit(network: "testnet" | "mainnet") {
  return new StellarWalletsKit({
    network:
      network === "mainnet"
        ? WalletNetwork.PUBLIC
        : WalletNetwork.TESTNET,
    selectedWalletId: FREIGHTER_ID,
    modules: allowAllModules(),
  });
}
```

### 9.2 Signing Component

Used by the Create API Key and Fund flows to sign transactions via Freighter.

```tsx
// dashboard/src/components/sign-transaction.tsx

import { useStellarWallet } from "@/hooks/use-wallet";

interface SignTransactionProps {
  unsignedXDR: string;
  onSigned: (signedXDR: string) => void;
  onError: (error: Error) => void;
  label: string;
}

export function SignTransaction({
  unsignedXDR,
  onSigned,
  onError,
  label,
}: SignTransactionProps) {
  const { kit, connected } = useStellarWallet();
  const [signing, setSigning] = useState(false);

  const handleSign = async () => {
    setSigning(true);
    try {
      const { signedTxXdr } = await kit.signTransaction(unsignedXDR);
      onSigned(signedTxXdr);
    } catch (err) {
      onError(err as Error);
    } finally {
      setSigning(false);
    }
  };

  if (!connected) {
    return <ConnectWalletButton />;
  }

  return (
    <Button onClick={handleSign} disabled={signing}>
      {signing ? "Signing..." : label}
    </Button>
  );
}
```

### 9.3 Create API Key Flow

The Create API Key page follows a multi-step flow:

1. **Form**: Admin fills in name, XLM budget, allowed operations, expiration, rate limit
2. **API Call**: `POST /v1/admin/api-keys` → receives `funding_transaction_xdr` and `api_key`
3. **Display API Key**: Show the API key once with a copy button and warning that it won't be shown again
4. **Sign**: Present the funding transaction to Freighter via `SignTransaction` component
5. **Activate**: After signing, call `POST /v1/admin/api-keys/:id/activate` with the signed XDR
6. **Confirmation**: Show success with sponsor account details

### 9.4 Dashboard Pages

| Page | Route | Data Source |
|------|-------|-------------|
| Overview | `/` | `GET /v1/health` + `GET /v1/admin/api-keys` (aggregate balances from Horizon) |
| API Keys List | `/api-keys` | `GET /v1/admin/api-keys` |
| Create API Key | `/api-keys/new` | `POST /v1/admin/api-keys` → Freighter sign → `POST /activate` |
| API Key Detail | `/api-keys/[id]` | `GET /v1/admin/api-keys` + Horizon balance |
| Transaction Logs | `/transactions` | `GET /v1/admin/transactions` |
| Settings | `/settings` | `GET /v1/health` + env config display |

---

## 10. Docker Setup

### API Dockerfile

```dockerfile
# docker/Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /sponsorship-service ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /sponsorship-service /sponsorship-service
COPY migrations/ /migrations/
EXPOSE 8080
ENTRYPOINT ["/sponsorship-service"]
```

### Dashboard Dockerfile

```dockerfile
# docker/Dockerfile.dashboard
FROM node:20-alpine AS builder
WORKDIR /app
COPY dashboard/package*.json ./
RUN npm ci
COPY dashboard/ .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
```

### Docker Compose

```yaml
# docker/docker-compose.yml
services:
  api:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    environment:
      STELLAR_NETWORK: testnet
      SIGNING_SECRET_KEY: ${SIGNING_SECRET_KEY}
      MASTER_FUNDING_PUBLIC_KEY: ${MASTER_FUNDING_PUBLIC_KEY}
      DATABASE_URL: postgres://sponsorship:sponsorship@db:5432/sponsorship?sslmode=disable
      ADMIN_API_KEY: ${ADMIN_API_KEY}
    depends_on:
      db:
        condition: service_healthy

  dashboard:
    build:
      context: ..
      dockerfile: docker/Dockerfile.dashboard
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080
      NEXT_PUBLIC_STELLAR_NETWORK: testnet

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: sponsorship
      POSTGRES_PASSWORD: sponsorship
      POSTGRES_DB: sponsorship
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sponsorship"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

---

## 11. Testing Strategy

### Unit Tests

| Package | What to test |
|---------|-------------|
| `internal/stellar/verifier` | All verification rules — valid transactions, each rejection case (sponsor as source, disallowed ops, XLM transfers, missing sponsoring blocks, etc.) |
| `internal/stellar/signer` | Signing produces valid signatures, rejects invalid XDR |
| `internal/stellar/operations` | Operation type mapping, XLM transfer detection |
| `internal/stellar/builder` | Funding transaction structure, sweep transaction structure |
| `internal/middleware/auth` | API key extraction, hashing, expiration, status checks |
| `internal/middleware/ratelimit` | Window reset, counter increment, limit enforcement |
| `internal/handler/*` | Request parsing, response format, error codes |

### Integration Tests

| Test | Description |
|------|-------------|
| Full signing flow | Create API key → fund → sign a valid sponsorship transaction → verify signature |
| Rejection scenarios | Submit transactions that violate each verification rule, assert correct error codes |
| Admin CRUD | Create, list, update, revoke API keys; verify database state |
| Rate limiting | Exceed rate limit, verify 429 response, verify reset after window |

### E2E Tests (Testnet)

| Test | Description |
|------|-------------|
| Account sponsorship | Create a sponsored account on testnet, verify on-chain |
| Trustline sponsorship | Add a sponsored trustline, verify on-chain |
| Reserve recovery | Remove a sponsored trustline, verify XLM returned to sponsor account |
| Fund and sweep | Add funds to sponsor account, revoke, sweep back to master |

### Running Tests

```bash
# Unit tests
go test ./internal/... -v

# Integration tests (requires PostgreSQL)
go test ./internal/... -tags=integration -v

# E2E tests (requires testnet funded accounts)
go test ./e2e/... -tags=e2e -v

# Dashboard tests
cd dashboard && npm test
```

---

## 12. Makefile

```makefile
.PHONY: build run test migrate docker-up docker-down

build:
	go build -o bin/sponsorship-service ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./internal/... -v

test-integration:
	go test ./internal/... -tags=integration -v

migrate-up:
	migrate -path migrations -database $(DATABASE_URL) up

migrate-down:
	migrate -path migrations -database $(DATABASE_URL) down

docker-up:
	docker compose -f docker/docker-compose.yml up --build -d

docker-down:
	docker compose -f docker/docker-compose.yml down

lint:
	golangci-lint run ./...

dashboard-dev:
	cd dashboard && npm run dev

dashboard-build:
	cd dashboard && npm run build
```

---

## 13. Metrics

```go
// Registered in server setup

var (
    transactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "sponsorship_transactions_total",
        Help: "Total transactions processed",
    }, []string{"status", "api_key_name"})

    requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "sponsorship_request_duration_seconds",
        Help:    "Request duration in seconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"endpoint"})

    sponsorBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "sponsorship_sponsor_balance",
        Help: "Current XLM balance per sponsor account",
    }, []string{"api_key_name", "sponsor_account"})

    activeAPIKeys = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "sponsorship_active_api_keys",
        Help: "Number of active API keys",
    })
)
```

---

## 14. Error Response Format

All API errors follow a consistent JSON format:

```go
type ErrorResponse struct {
    Error   string `json:"error"`    // machine-readable error code
    Message string `json:"message"`  // human-readable description
}
```

Example:

```json
{
  "error": "sponsor_as_source",
  "message": "Transaction source account matches the sponsor account — this is not allowed"
}
```

---

## 15. Security Implementation Notes

### API Key Generation

```go
func generateAPIKey(network string) string {
    b := make([]byte, 32)
    crypto_rand.Read(b)
    prefix := "sk_live_"
    if network == "testnet" {
        prefix = "sk_test_"
    }
    return prefix + hex.EncodeToString(b)
}

func sha256Hex(input string) string {
    h := sha256.Sum256([]byte(input))
    return hex.EncodeToString(h[:])
}
```

- Keys are prefixed with `sk_live_` (mainnet) or `sk_test_` (testnet) for easy identification
- 32 random bytes = 256 bits of entropy
- SHA-256 hash stored in database — fast lookups, irreversible

### Signing Key Storage

The signing key secret is loaded from the environment at startup and held in memory. It is never:
- Written to disk
- Logged
- Returned in any API response
- Stored in the database

### Request Validation

All request bodies are validated with size limits:
- Max request body: 1MB (transaction XDRs can be large)
- All string fields trimmed and length-validated
- UUID parameters validated before database queries

---

## 16. Startup Sequence

```go
// cmd/server/main.go

func main() {
    // 1. Load config from environment
    cfg := config.Load()

    // 2. Initialize the transaction signer with the SIGNING key (SIGNING_SECRET_KEY env var).
    //    This is the service's own signing key — NOT the master funding account key.
    //    The master funding account key is never held by the service; it stays in the
    //    operator's external wallet (e.g., Freighter) and is used only through the
    //    admin dashboard to sign funding transactions.
    signer, err := stellar.NewSigner(cfg.SigningSecretKey, cfg.NetworkPassphrase())

    // 3. Connect to database
    pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)

    // 4. Run migrations
    migrate.Up(cfg.DatabaseURL, "migrations")

    // 5. Initialize Horizon client based on configured network
    var horizonClient *horizonclient.Client
    if cfg.HorizonURL != "" {
        horizonClient = &horizonclient.Client{HorizonURL: cfg.DefaultHorizonURL()}
    } else if cfg.StellarNetwork == "mainnet" {
        horizonClient = horizonclient.DefaultPublicNetClient
    } else {
        horizonClient = horizonclient.DefaultTestNetClient
    }

    // 6. Initialize services
    store := store.NewPostgres(pool)
    verifier := stellar.NewVerifier(cfg.NetworkPassphrase())
    builder := stellar.NewBuilder(horizonClient, signer.PublicKey(), cfg.MasterFundingPublicKey, cfg.NetworkPassphrase())
    accounts := stellar.NewAccountService(horizonClient)

    // 7. Start HTTP server
    server := server.New(cfg, store, signer, verifier, builder, accounts)
    server.ListenAndServe()
}
```
