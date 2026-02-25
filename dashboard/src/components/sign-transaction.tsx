"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useWalletContext } from "@/components/wallet-provider";
import { useHealth } from "@/hooks/use-api-keys";
import { Loader2, Wallet } from "lucide-react";

interface SignTransactionProps {
  unsignedXDR: string;
  onSigned: (signedXDR: string) => void;
  onError: (error: Error) => void;
  label: string;
  disabled?: boolean;
}

export function SignTransaction({
  unsignedXDR,
  onSigned,
  onError,
  label,
  disabled,
}: SignTransactionProps) {
  const { connected, connect, signTransaction, connecting } =
    useWalletContext();
  const health = useHealth();
  const [signing, setSigning] = useState(false);

  const handleSign = async () => {
    setSigning(true);
    try {
      const signedXDR = await signTransaction(unsignedXDR);
      onSigned(signedXDR);
    } catch (err: unknown) {
      if (err instanceof Error) {
        onError(err);
      } else if (err && typeof err === "object" && "message" in err) {
        onError(new Error(String((err as { message: unknown }).message)));
      } else {
        onError(new Error(typeof err === "string" ? err : "Wallet signing failed"));
      }
    } finally {
      setSigning(false);
    }
  };

  if (!connected) {
    return (
      <Button onClick={connect} disabled={connecting} variant="outline">
        {connecting ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Wallet className="mr-2 h-4 w-4" />
        )}
        {connecting
          ? "Connecting..."
          : health.data?.master_public_key
            ? `Connect Wallet (${health.data.master_public_key.slice(0, 5)}...)`
            : "Connect Wallet"}
      </Button>
    );
  }

  return (
    <Button onClick={handleSign} disabled={signing || disabled}>
      {signing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
      {signing ? "Signing..." : label}
    </Button>
  );
}
