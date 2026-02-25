"use client";

import { Badge } from "@/components/ui/badge";
import { ConnectWalletButton } from "@/components/connect-wallet-button";
import { useWalletContext } from "@/components/wallet-provider";
import { getStellarNetwork } from "@/lib/stellar";

export function Header() {
  const network = getStellarNetwork();
  const { connected } = useWalletContext();

  return (
    <header className="border-b">
      <div className="container mx-auto px-4 max-w-4xl flex h-14 items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold">Wallet Demo</h1>
          <Badge variant={network === "mainnet" ? "default" : "secondary"}>
            {network}
          </Badge>
        </div>
        {connected && <ConnectWalletButton />}
      </div>
    </header>
  );
}
