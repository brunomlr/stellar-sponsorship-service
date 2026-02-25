"use client";

import { useState } from "react";
import { Copy, Check, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { truncateKey } from "@/lib/utils";

interface StellarAccountProps {
  address: string;
  truncateChars?: number | false;
  className?: string;
}

function getStellarExpertUrl(address: string): string {
  const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK || "testnet";
  const path = network === "mainnet" ? "public" : "testnet";
  return `https://stellar.expert/explorer/${path}/account/${address}`;
}

export function StellarAccount({
  address,
  truncateChars = 8,
  className,
}: StellarAccountProps) {
  const [copied, setCopied] = useState(false);

  const displayAddress =
    truncateChars === false ? address : truncateKey(address, truncateChars);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(address);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <TooltipProvider>
      <span className={`inline-flex items-center gap-1 ${className ?? ""}`}>
        {truncateChars !== false ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono text-sm">{displayAddress}</span>
            </TooltipTrigger>
            <TooltipContent>
              <p className="font-mono">{address}</p>
            </TooltipContent>
          </Tooltip>
        ) : (
          <span className="font-mono text-sm break-all">{address}</span>
        )}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 shrink-0"
              onClick={handleCopy}
            >
              {copied ? (
                <Check className="h-3 w-3" />
              ) : (
                <Copy className="h-3 w-3" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>Copy address</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 shrink-0"
              asChild
            >
              <a
                href={getStellarExpertUrl(address)}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink className="h-3 w-3" />
              </a>
            </Button>
          </TooltipTrigger>
          <TooltipContent>View on Stellar Expert</TooltipContent>
        </Tooltip>
      </span>
    </TooltipProvider>
  );
}
