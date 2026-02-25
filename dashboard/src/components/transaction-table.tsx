"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { StellarAccount } from "@/components/stellar-account";
import { formatDate, truncateKey } from "@/lib/utils";
import type { TransactionLog } from "@/lib/api";
import { useCheckTransaction } from "@/hooks/use-api-keys";
import { Loader2, RefreshCw } from "lucide-react";

interface TransactionTableProps {
  transactions: TransactionLog[];
}

export function TransactionTable({ transactions }: TransactionTableProps) {
  const checkTx = useCheckTransaction();

  return (
    <TooltipProvider>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>TX Hash</TableHead>
            <TableHead>Source</TableHead>
            <TableHead>Operations</TableHead>
            <TableHead>Reserves</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Submitted</TableHead>
            <TableHead>Reason</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {transactions.map((tx) => (
            <TableRow key={tx.id}>
              <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                {formatDate(tx.created_at)}
              </TableCell>
              <TableCell className="font-mono text-sm">
                {tx.transaction_hash ? (
                  <Tooltip>
                    <TooltipTrigger>
                      {truncateKey(tx.transaction_hash, 6)}
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="font-mono">{tx.transaction_hash}</p>
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <span className="text-muted-foreground">-</span>
                )}
              </TableCell>
              <TableCell>
                <StellarAccount address={tx.source_account} truncateChars={6} />
              </TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-1">
                  {tx.operations?.map((op, i) => (
                    <Badge key={i} variant="outline" className="text-xs">
                      {op}
                    </Badge>
                  ))}
                </div>
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {tx.reserves_locked != null ? tx.reserves_locked : "-"}
              </TableCell>
              <TableCell>
                <Badge
                  variant={
                    tx.status === "signed" ? "default" : "destructive"
                  }
                >
                  {tx.status}
                </Badge>
              </TableCell>
              <TableCell>
                <SubmissionCell tx={tx} onCheck={(id) => checkTx.mutate(id)} isChecking={checkTx.isPending} checkingId={checkTx.variables} />
              </TableCell>
              <TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
                {tx.rejection_reason || "-"}
              </TableCell>
            </TableRow>
          ))}
          {transactions.length === 0 && (
            <TableRow>
              <TableCell
                colSpan={8}
                className="text-center text-muted-foreground py-8"
              >
                No transactions found.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TooltipProvider>
  );
}

function SubmissionCell({
  tx,
  onCheck,
  isChecking,
  checkingId,
}: {
  tx: TransactionLog;
  onCheck: (id: string) => void;
  isChecking: boolean;
  checkingId?: string;
}) {
  if (tx.status === "rejected") {
    return <span className="text-muted-foreground">-</span>;
  }

  if (tx.submission_status === "confirmed") {
    return (
      <Tooltip>
        <TooltipTrigger>
          <Badge className="bg-green-600 hover:bg-green-600/80 text-white border-transparent">
            Confirmed
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          {tx.ledger_sequence && <p>Ledger #{tx.ledger_sequence}</p>}
          {tx.submitted_at && <p>{formatDate(tx.submitted_at)}</p>}
        </TooltipContent>
      </Tooltip>
    );
  }

  const isThisChecking = isChecking && checkingId === tx.id;

  return (
    <div className="flex items-center gap-1">
      <Badge variant={tx.submission_status === "not_found" ? "secondary" : "outline"}>
        {tx.submission_status === "not_found" ? "Not Found" : "Unknown"}
      </Badge>
      {tx.transaction_hash && (
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={() => onCheck(tx.id)}
          disabled={isThisChecking}
        >
          {isThisChecking ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCw className="h-3 w-3" />
          )}
        </Button>
      )}
    </div>
  );
}
