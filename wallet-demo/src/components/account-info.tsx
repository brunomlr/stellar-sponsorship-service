"use client";

import { useCallback, useEffect, useState } from "react";
import * as StellarSdk from "@stellar/stellar-sdk";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useWalletContext } from "@/components/wallet-provider";
import { getStellarNetwork } from "@/lib/stellar";
import { formatXLM, truncateKey } from "@/lib/utils";
import { Loader2, RefreshCw } from "lucide-react";

interface AccountData {
  xlmBalance: string;
  subentryCount: number;
  numSponsoring: number;
  numSponsored: number;
  trustlines: { code: string; issuer: string }[];
}

function getHorizonUrl(): string {
  const custom = process.env.NEXT_PUBLIC_HORIZON_URL;
  if (custom) return custom;
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  return network === "mainnet"
    ? "https://horizon.stellar.org"
    : "https://horizon-testnet.stellar.org";
}

export function AccountInfo() {
  const { publicKey } = useWalletContext();
  const [data, setData] = useState<AccountData | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [funding, setFunding] = useState(false);

  const network = getStellarNetwork();

  const loadAccount = useCallback(() => {
    if (!publicKey) return;

    setLoading(true);
    setError(null);
    setNotFound(false);

    const server = new StellarSdk.Horizon.Server(getHorizonUrl());
    server
      .loadAccount(publicKey)
      .then((account) => {
        const native = account.balances.find(
          (b) => b.asset_type === "native"
        );
        const trustlines = account.balances
          .filter((b) => b.asset_type !== "native" && "asset_code" in b)
          .map((b) => ({
            code: (b as { asset_code: string }).asset_code,
            issuer: (b as { asset_issuer: string }).asset_issuer,
          }));

        const raw = account as unknown as Record<string, unknown>;
        setData({
          xlmBalance: native ? native.balance : "0",
          subentryCount: account.subentry_count,
          numSponsoring: (raw.num_sponsoring as number) ?? 0,
          numSponsored: (raw.num_sponsored as number) ?? 0,
          trustlines,
        });
        setNotFound(false);
      })
      .catch((err) => {
        if (err?.response?.status === 404) {
          setNotFound(true);
        } else {
          setError(
            err instanceof Error ? err.message : "Failed to load account"
          );
        }
      })
      .finally(() => setLoading(false));
  }, [publicKey]);

  useEffect(() => {
    loadAccount();
  }, [loadAccount]);

  const handleFundWithFriendbot = async () => {
    if (!publicKey) return;
    setFunding(true);
    setError(null);
    try {
      const res = await fetch(
        `https://friendbot.stellar.org?addr=${encodeURIComponent(publicKey)}`
      );
      if (!res.ok) {
        const body = await res.text();
        if (!body.includes("createAccountAlreadyExist")) {
          throw new Error(`Friendbot failed: ${res.status}`);
        }
      }
      loadAccount();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fund via Friendbot"
      );
    } finally {
      setFunding(false);
    }
  };

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-medium">Your Account</CardTitle>
        {data && (
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={loadAccount}
          >
            <RefreshCw className="h-3 w-3" />
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {publicKey && (
          <div>
            <p className="text-xs text-muted-foreground">Address</p>
            <p className="text-sm font-mono">{truncateKey(publicKey, 12)}</p>
          </div>
        )}
        {loading && (
          <div className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-32" />
          </div>
        )}
        {error && <p className="text-sm text-destructive">{error}</p>}
        {notFound && !loading && (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Account not found on the network.
            </p>
            {network === "testnet" && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleFundWithFriendbot}
                disabled={funding}
              >
                {funding ? (
                  <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                ) : null}
                {funding ? "Funding..." : "Fund with Friendbot"}
              </Button>
            )}
          </div>
        )}
        {data && (
          <>
            <div>
              <p className="text-xs text-muted-foreground">XLM Balance</p>
              <p className="text-lg font-semibold">
                {formatXLM(data.xlmBalance)} XLM
              </p>
            </div>
            <div className="flex gap-4 text-sm">
              <div>
                <p className="text-xs text-muted-foreground">Subentries</p>
                <p>{data.subentryCount}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Sponsored</p>
                <p>{data.numSponsored}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Sponsoring</p>
                <p>{data.numSponsoring}</p>
              </div>
            </div>
            {data.trustlines.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">
                  Trustlines
                </p>
                <div className="flex flex-wrap gap-1">
                  {data.trustlines.map((tl) => (
                    <Badge key={`${tl.code}-${tl.issuer}`} variant="outline">
                      {tl.code}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
