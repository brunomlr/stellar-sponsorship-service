// Mirrors Go API GET /v1/info response
export interface InfoResponse {
  network_passphrase: string;
  base_reserve: string;
  supported_operations: string[];
}

// Mirrors Go API GET /v1/usage response
export interface UsageResponse {
  api_key_name: string;
  sponsor_account: string;
  xlm_budget: string;
  xlm_available: string;
  xlm_locked_in_reserves: string;
  allowed_operations: string[];
  expires_at: string;
  is_active: boolean;
  transactions_signed: number;
  rate_limit: {
    max_requests: number;
    window_seconds: number;
    remaining: number;
  };
}

// Mirrors Go API POST /v1/sign response
export interface SignResponse {
  signed_transaction_xdr: string;
  sponsor_public_key: string;
  sponsor_account_balance: string;
}

// API error shape
export interface ApiErrorBody {
  error: string;
  message: string;
}

// Client-side transaction record (stored in React state)
export interface TransactionRecord {
  id: string;
  type: "add_trustline" | "create_account";
  description: string;
  status:
    | "building"
    | "signing_sponsor"
    | "signing_wallet"
    | "submitting"
    | "success"
    | "failed";
  txHash?: string;
  error?: string;
  timestamp: Date;
}
