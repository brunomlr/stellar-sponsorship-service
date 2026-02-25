import { getSession } from "next-auth/react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const session = await getSession();
  const token = session?.id_token || "";

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      ...options.headers,
    },
  });

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
    throw new ApiError(401, "unauthorized", "Session expired");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      body.error || "unknown_error",
      body.message || res.statusText
    );
  }

  return res.json();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// --- Health ---

export interface HealthResponse {
  status: string;
  version: string;
  stellar_network: string;
  master_public_key: string;
  master_account_balance: string;
  total_sponsor_accounts: number;
  uptime_seconds: number;
}

export function getHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>("/v1/health");
}

// --- API Keys ---

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  sponsor_account: string;
  xlm_budget: string;
  xlm_available: string;
  allowed_operations: string[];
  allowed_source_accounts?: string[];
  rate_limit_max: number;
  rate_limit_window: number;
  expires_at: string;
  status: string;
  created_at: string;
}

export interface ListAPIKeysResponse {
  api_keys: APIKey[];
  total: number;
  page: number;
  per_page: number;
}

export function listAPIKeys(
  page = 1,
  perPage = 20
): Promise<ListAPIKeysResponse> {
  return apiFetch<ListAPIKeysResponse>(
    `/v1/admin/api-keys?page=${page}&per_page=${perPage}`
  );
}

export function getAPIKey(id: string): Promise<APIKey> {
  return apiFetch<APIKey>(`/v1/admin/api-keys/${id}`);
}

export interface CreateAPIKeyRequest {
  name: string;
  xlm_budget: string;
  allowed_operations: string[];
  expires_at: string;
  rate_limit?: {
    max_requests: number;
    window_seconds: number;
  };
  allowed_source_accounts?: string[];
}

export interface CreateAPIKeyResponse {
  id: string;
  name: string;
  api_key: string;
  xlm_budget: string;
  allowed_operations: string[];
  expires_at: string;
  status: string;
  created_at: string;
}

export function createAPIKey(
  req: CreateAPIKeyRequest
): Promise<CreateAPIKeyResponse> {
  return apiFetch<CreateAPIKeyResponse>("/v1/admin/api-keys", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export interface UpdateAPIKeyRequest {
  name?: string;
  allowed_operations?: string[];
  allowed_source_accounts?: string[];
  rate_limit_max?: number;
  rate_limit_window?: number;
  expires_at?: string;
}

export function updateAPIKey(
  id: string,
  req: UpdateAPIKeyRequest
): Promise<APIKey> {
  return apiFetch<APIKey>(`/v1/admin/api-keys/${id}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export interface RegenerateAPIKeyResponse {
  id: string;
  api_key: string;
  key_prefix: string;
}

export function regenerateAPIKey(
  id: string
): Promise<RegenerateAPIKeyResponse> {
  return apiFetch<RegenerateAPIKeyResponse>(
    `/v1/admin/api-keys/${id}/regenerate`,
    { method: "POST" }
  );
}

export function revokeAPIKey(
  id: string
): Promise<{ id: string; status: string }> {
  return apiFetch<{ id: string; status: string }>(
    `/v1/admin/api-keys/${id}`,
    { method: "DELETE" }
  );
}

// --- Activate ---

export interface BuildActivateResponse {
  sponsor_account: string;
  xlm_budget: string;
  activate_transaction_xdr: string;
}

export function buildActivateTransaction(
  id: string
): Promise<BuildActivateResponse> {
  return apiFetch<BuildActivateResponse>(
    `/v1/admin/api-keys/${id}/activate`,
    { method: "POST" }
  );
}

export interface SubmitActivateResponse {
  id: string;
  status: string;
  sponsor_account: string;
  transaction_hash: string;
}

export function submitActivateTransaction(
  id: string,
  signedTransactionXDR: string
): Promise<SubmitActivateResponse> {
  return apiFetch<SubmitActivateResponse>(
    `/v1/admin/api-keys/${id}/activate/submit`,
    {
      method: "POST",
      body: JSON.stringify({ signed_transaction_xdr: signedTransactionXDR }),
    }
  );
}

// --- Fund ---

export interface BuildFundResponse {
  sponsor_account: string;
  xlm_to_add: string;
  funding_transaction_xdr: string;
}

export function buildFundTransaction(
  id: string,
  amount: string
): Promise<BuildFundResponse> {
  return apiFetch<BuildFundResponse>(`/v1/admin/api-keys/${id}/fund`, {
    method: "POST",
    body: JSON.stringify({ amount }),
  });
}

export interface SubmitFundResponse {
  sponsor_account: string;
  xlm_added: string;
  xlm_available: string;
  transaction_hash: string;
}

export function submitFundTransaction(
  id: string,
  signedTransactionXDR: string
): Promise<SubmitFundResponse> {
  return apiFetch<SubmitFundResponse>(
    `/v1/admin/api-keys/${id}/fund/submit`,
    {
      method: "POST",
      body: JSON.stringify({ signed_transaction_xdr: signedTransactionXDR }),
    }
  );
}

// --- Sweep ---

export interface SweepResponse {
  sponsor_account: string;
  xlm_swept: string;
  xlm_remaining_locked: string;
  destination: string;
  transaction_hash: string;
}

export function sweepFunds(id: string): Promise<SweepResponse> {
  return apiFetch<SweepResponse>(`/v1/admin/api-keys/${id}/sweep`, {
    method: "POST",
  });
}

// --- Transactions ---

export interface TransactionLog {
  id: string;
  api_key_id: string;
  transaction_hash: string;
  operations: string[];
  source_account: string;
  status: string;
  rejection_reason?: string;
  submission_status: "confirmed" | "not_found" | null;
  submission_checked_at?: string;
  ledger_sequence?: number;
  submitted_at?: string;
  reserves_locked?: number;
  created_at: string;
}

export interface ListTransactionsResponse {
  transactions: TransactionLog[];
  total: number;
  page: number;
  per_page: number;
}

export function listTransactions(params: {
  page?: number;
  per_page?: number;
  api_key_id?: string;
  status?: string;
  from?: string;
  to?: string;
}): Promise<ListTransactionsResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.api_key_id) searchParams.set("api_key_id", params.api_key_id);
  if (params.status) searchParams.set("status", params.status);
  if (params.from) searchParams.set("from", params.from);
  if (params.to) searchParams.set("to", params.to);

  return apiFetch<ListTransactionsResponse>(
    `/v1/admin/transactions?${searchParams.toString()}`
  );
}

export interface CheckTransactionResponse {
  id: string;
  submission_status: string;
  ledger_sequence?: number;
  submitted_at?: string;
}

export function checkTransactionSubmission(
  id: string
): Promise<CheckTransactionResponse> {
  return apiFetch<CheckTransactionResponse>(
    `/v1/admin/transactions/${id}/check`,
    { method: "POST" }
  );
}
