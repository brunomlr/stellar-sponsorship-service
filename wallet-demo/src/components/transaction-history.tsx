"use client";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { getExplorerUrl } from "@/lib/utils";
import { getStellarNetwork } from "@/lib/stellar";
import type { TransactionRecord } from "@/types";
import { ExternalLink } from "lucide-react";

interface TransactionHistoryProps {
  transactions: TransactionRecord[];
}

const STATUS_VARIANT: Record<
  TransactionRecord["status"],
  "default" | "secondary" | "destructive" | "outline"
> = {
  building: "secondary",
  signing_sponsor: "secondary",
  signing_wallet: "secondary",
  submitting: "secondary",
  success: "default",
  failed: "destructive",
};

const STATUS_LABEL: Record<TransactionRecord["status"], string> = {
  building: "Building",
  signing_sponsor: "Signing (Sponsor)",
  signing_wallet: "Signing (Wallet)",
  submitting: "Submitting",
  success: "Success",
  failed: "Failed",
};

const TYPE_LABEL: Record<TransactionRecord["type"], string> = {
  add_trustline: "Add Trustline",
  create_account: "Create Account",
};

export function TransactionHistory({
  transactions,
}: TransactionHistoryProps) {
  const network = getStellarNetwork();

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">
          Transaction History
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {transactions.map((tx) => (
            <div
              key={tx.id}
              className="flex items-center justify-between text-sm border-b pb-2 last:border-0"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    {TYPE_LABEL[tx.type]}
                  </span>
                  <Badge variant={STATUS_VARIANT[tx.status]} className="text-xs">
                    {STATUS_LABEL[tx.status]}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  {tx.description}
                </p>
                {tx.error && (
                  <p className="text-xs text-destructive">{tx.error}</p>
                )}
              </div>
              <div className="text-right shrink-0">
                <p className="text-xs text-muted-foreground">
                  {tx.timestamp.toLocaleTimeString()}
                </p>
                {tx.txHash && (
                  <a
                    href={getExplorerUrl(network, tx.txHash)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-primary hover:underline inline-flex items-center gap-1"
                  >
                    {tx.txHash.slice(0, 8)}...
                    <ExternalLink className="h-3 w-3" />
                  </a>
                )}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
