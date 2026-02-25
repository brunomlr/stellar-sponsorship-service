import type { InfoResponse, UsageResponse, SignResponse } from "@/types";

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

async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

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

export function getInfo(): Promise<InfoResponse> {
  return apiFetch<InfoResponse>("/info");
}

export function getUsage(): Promise<UsageResponse> {
  return apiFetch<UsageResponse>("/usage");
}

export function signTransaction(
  transactionXdr: string
): Promise<SignResponse> {
  return apiFetch<SignResponse>("/sign", {
    method: "POST",
    body: JSON.stringify({ transaction_xdr: transactionXdr }),
  });
}
