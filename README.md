# Stellar Sponsorship Service

A self-hostable API service that lets wallets and applications sponsor Stellar base reserves without holding or transferring XLM. Each API key gets a dedicated on-chain sponsor account with an XLM budget enforced by the Stellar network itself.

## How It Works

1. Admin creates an API key via the dashboard, which generates a dedicated Stellar sponsor account
2. Admin funds the sponsor account with XLM using their own wallet (e.g. Freighter)
3. Wallets send unsigned transactions to `POST /v1/sign` with the API key
4. The service validates the transaction (operation allowlist, sponsorship blocks, no XLM transfers) and co-signs it
5. The wallet submits the signed transaction to the Stellar network

## Tech Stack

- **API**: Go, chi router, PostgreSQL, Stellar Go SDK
- **Dashboard**: Next.js 15, TypeScript, shadcn/ui, Google OAuth
- **Infrastructure**: Docker, Docker Compose

## Quick Start

```bash
# Clone and configure
cp .env.example .env        # Edit with your values
cp dashboard/.env.example dashboard/.env

# Run with Docker Compose
make docker-up

# Or run locally
make migrate-up
make run                    # API on :8080
cd dashboard && npm install && npm run dev  # Dashboard on :3000
```

## Documentation

- [Setup, Architecture & API Reference](docs/DOCUMENTATION.md) — full setup instructions, configuration, and admin endpoints
- [Integration Guide](docs/integration-guide.md) — how to integrate the sponsorship API into your wallet or application
- [Wallet Demo](wallet-demo/README.md) — a working Next.js demo app that shows the full sponsorship flow end-to-end

## License

Private — All rights reserved.
