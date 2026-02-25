"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useSponsorUsage } from "@/hooks/use-sponsor";
import { formatXLM, truncateKey } from "@/lib/utils";

export function SponsorInfo() {
  const { data, isLoading, error } = useSponsorUsage();

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">Sponsor</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && (
          <div className="space-y-2">
            <Skeleton className="h-4 w-32" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-4 w-40" />
          </div>
        )}
        {error && (
          <p className="text-sm text-destructive">
            {error instanceof Error ? error.message : "Failed to load sponsor info"}
          </p>
        )}
        {data && (
          <>
            <div>
              <p className="text-xs text-muted-foreground">Sponsor Account</p>
              <p className="text-sm font-mono">
                {truncateKey(data.sponsor_account, 12)}
              </p>
            </div>
            <div className="flex gap-4">
              <div>
                <p className="text-xs text-muted-foreground">Available</p>
                <p className="text-lg font-semibold">
                  {formatXLM(data.xlm_available)} XLM
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Locked</p>
                <p className="text-sm text-muted-foreground">
                  {formatXLM(data.xlm_locked_in_reserves)} XLM
                </p>
              </div>
            </div>
            <div>
              <p className="text-xs text-muted-foreground mb-1">
                Allowed Operations
              </p>
              <div className="flex flex-wrap gap-1">
                {data.allowed_operations.map((op) => (
                  <Badge key={op} variant="secondary" className="text-xs">
                    {op}
                  </Badge>
                ))}
              </div>
            </div>
            <div className="flex gap-4 text-sm">
              <div>
                <p className="text-xs text-muted-foreground">Rate Limit</p>
                <p>
                  {data.rate_limit.remaining}/{data.rate_limit.max_requests}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">TX Signed</p>
                <p>{data.transactions_signed}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Status</p>
                <Badge variant={data.is_active ? "default" : "destructive"}>
                  {data.is_active ? "Active" : "Inactive"}
                </Badge>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
