"use client";

import Link from "next/link";
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
import { StellarAccount } from "@/components/stellar-account";
import { formatXLM, truncateKey, formatDate } from "@/lib/utils";
import type { APIKey } from "@/lib/api";
import { ExternalLink } from "lucide-react";

interface APIKeyTableProps {
  apiKeys: APIKey[];
}

function statusVariant(status: string) {
  switch (status) {
    case "active":
      return "default" as const;
    case "pending_funding":
      return "secondary" as const;
    case "revoked":
      return "destructive" as const;
    default:
      return "outline" as const;
  }
}

export function APIKeyTable({ apiKeys }: APIKeyTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Sponsor Account</TableHead>
          <TableHead className="text-right">XLM Available</TableHead>
          <TableHead>Operations</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Expires</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {apiKeys.map((key) => (
          <TableRow key={key.id}>
            <TableCell className="font-medium">{key.name}</TableCell>
            <TableCell>
              {key.sponsor_account ? (
                <StellarAccount address={key.sponsor_account} />
              ) : (
                <span className="text-muted-foreground">—</span>
              )}
            </TableCell>
            <TableCell className="text-right">
              {formatXLM(key.xlm_available)} XLM
            </TableCell>
            <TableCell>
              <div className="flex flex-wrap gap-1">
                {key.allowed_operations.map((op) => (
                  <Badge key={op} variant="outline" className="text-xs">
                    {op}
                  </Badge>
                ))}
              </div>
            </TableCell>
            <TableCell>
              <Badge variant={statusVariant(key.status)}>
                {key.status.replace("_", " ")}
              </Badge>
            </TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {formatDate(key.expires_at)}
            </TableCell>
            <TableCell className="text-right">
              <Button variant="ghost" size="sm" asChild>
                <Link href={`/api-keys/${key.id}`}>
                  <ExternalLink className="h-4 w-4" />
                </Link>
              </Button>
            </TableCell>
          </TableRow>
        ))}
        {apiKeys.length === 0 && (
          <TableRow>
            <TableCell
              colSpan={7}
              className="text-center text-muted-foreground py-8"
            >
              No API keys found. Create one to get started.
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
