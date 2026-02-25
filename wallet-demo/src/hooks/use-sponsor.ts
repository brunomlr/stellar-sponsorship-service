"use client";

import { useQuery } from "@tanstack/react-query";
import { getInfo, getUsage } from "@/lib/api";

export function useSponsorInfo() {
  return useQuery({
    queryKey: ["sponsor-info"],
    queryFn: getInfo,
    staleTime: 60_000,
  });
}

export function useSponsorUsage() {
  return useQuery({
    queryKey: ["sponsor-usage"],
    queryFn: getUsage,
    refetchInterval: 15_000,
  });
}
