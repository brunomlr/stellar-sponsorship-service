"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { StellarAccount } from "@/components/stellar-account";
import { useHealth, useAPIKeys } from "@/hooks/use-api-keys";
import { formatXLM } from "@/lib/utils";
import { Loader2, Activity, Key, Wallet, Clock } from "lucide-react";

export default function OverviewPage() {
  const health = useHealth();
  const apiKeys = useAPIKeys(1, 100);

  const activeKeys =
    apiKeys.data?.api_keys.filter((k) => k.status === "active") || [];
  const totalAvailable = activeKeys.reduce(
    (sum, k) => sum + parseFloat(k.xlm_available || "0"),
    0
  );

  if (health.isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Overview</h2>
        <p className="text-muted-foreground">
          Stellar Sponsorship Service Dashboard
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Master Account Balance
            </CardTitle>
            <Wallet className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {health.data
                ? formatXLM(health.data.master_account_balance)
                : "-"}{" "}
              XLM
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Sponsor Accounts
            </CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {health.data?.total_sponsor_accounts ?? "-"}
            </div>
            <p className="text-xs text-muted-foreground">
              {activeKeys.length} active
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              Total Available XLM
            </CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatXLM(totalAvailable.toFixed(7))} XLM
            </div>
            <p className="text-xs text-muted-foreground">
              Across all sponsor accounts
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Uptime</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {health.data
                ? formatUptime(health.data.uptime_seconds)
                : "-"}
            </div>
            <p className="text-xs text-muted-foreground">
              {health.data?.stellar_network ?? ""}{" "}
              <Badge variant="outline" className="ml-1">
                v{health.data?.version}
              </Badge>
            </p>
          </CardContent>
        </Card>
      </div>

      <Separator />

      <Card>
        <CardHeader>
          <CardTitle>Active API Keys</CardTitle>
          <CardDescription>
            Quick view of active sponsor accounts and their balances.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {activeKeys.length === 0 ? (
            <p className="text-muted-foreground text-sm">
              No active API keys. Create one from the API Keys page.
            </p>
          ) : (
            <div className="space-y-3">
              {activeKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div>
                    <p className="font-medium">{key.name}</p>
                    {key.sponsor_account ? (
                      <StellarAccount address={key.sponsor_account} truncateChars={6} />
                    ) : (
                      <p className="text-sm text-muted-foreground">—</p>
                    )}
                  </div>
                  <div className="text-right">
                    <p className="font-mono font-medium">
                      {formatXLM(key.xlm_available)} XLM
                    </p>
                    <p className="text-xs text-muted-foreground">available</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${mins}m`;
  return `${mins}m`;
}
