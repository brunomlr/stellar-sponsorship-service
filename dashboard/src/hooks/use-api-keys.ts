"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listAPIKeys,
  getAPIKey,
  createAPIKey,
  updateAPIKey,
  regenerateAPIKey,
  revokeAPIKey,
  buildActivateTransaction,
  submitActivateTransaction,
  sweepFunds,
  listTransactions,
  checkTransactionSubmission,
  getHealth,
  type CreateAPIKeyRequest,
  type UpdateAPIKeyRequest,
} from "@/lib/api";

export function useAPIKeys(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["api-keys", page, perPage],
    queryFn: () => listAPIKeys(page, perPage),
    refetchInterval: 30000,
  });
}

export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: 30000,
  });
}

export function useTransactions(params: {
  page?: number;
  per_page?: number;
  api_key_id?: string;
  status?: string;
}) {
  return useQuery({
    queryKey: ["transactions", params],
    queryFn: () => listTransactions(params),
    refetchInterval: 15000,
  });
}

export function useCreateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateAPIKeyRequest) => createAPIKey(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
  });
}

export function useAPIKey(id: string) {
  return useQuery({
    queryKey: ["api-key", id],
    queryFn: () => getAPIKey(id),
    enabled: Boolean(id),
    refetchInterval: 30000,
  });
}

export function useUpdateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...req }: UpdateAPIKeyRequest & { id: string }) =>
      updateAPIKey(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      queryClient.invalidateQueries({ queryKey: ["api-key"] });
    },
  });
}

export function useRegenerateAPIKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => regenerateAPIKey(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      queryClient.invalidateQueries({ queryKey: ["api-key"] });
    },
  });
}

export function useRevokeAPIKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => revokeAPIKey(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
  });
}

export function useBuildActivate() {
  return useMutation({
    mutationFn: (id: string) => buildActivateTransaction(id),
  });
}

export function useSubmitActivate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      signedXDR,
    }: {
      id: string;
      signedXDR: string;
    }) => submitActivateTransaction(id, signedXDR),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      queryClient.invalidateQueries({ queryKey: ["api-key"] });
    },
  });
}

export function useSweepFunds() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => sweepFunds(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
  });
}

export function useCheckTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => checkTransactionSubmission(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
    },
  });
}
