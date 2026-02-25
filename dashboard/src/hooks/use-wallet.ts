"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createWalletKit, getStellarNetwork, getNetworkPassphrase } from "@/lib/stellar";
import type { StellarWalletsKit } from "@creit.tech/stellar-wallets-kit";

interface WalletState {
  kit: StellarWalletsKit | null;
  connected: boolean;
  publicKey: string | null;
  connecting: boolean;
  error: string | null;
}

export function useWallet() {
  const kitRef = useRef<StellarWalletsKit | null>(null);
  const [state, setState] = useState<WalletState>({
    kit: null,
    connected: false,
    publicKey: null,
    connecting: false,
    error: null,
  });

  useEffect(() => {
    const network = getStellarNetwork();
    kitRef.current = createWalletKit(network);
    setState((s) => ({ ...s, kit: kitRef.current }));
  }, []);

  const connect = useCallback(async () => {
    if (!kitRef.current) return;
    setState((s) => ({ ...s, connecting: true, error: null }));
    try {
      await kitRef.current.openModal({
        onWalletSelected: async (option) => {
          kitRef.current!.setWallet(option.id);
          const { address } = await kitRef.current!.getAddress();
          setState({
            kit: kitRef.current,
            connected: true,
            publicKey: address,
            connecting: false,
            error: null,
          });
        },
      });
    } catch (err) {
      setState((s) => ({
        ...s,
        connecting: false,
        error: err instanceof Error ? err.message : "Failed to connect wallet",
      }));
    }
  }, []);

  const disconnect = useCallback(() => {
    setState({
      kit: kitRef.current,
      connected: false,
      publicKey: null,
      connecting: false,
      error: null,
    });
  }, []);

  const signTransaction = useCallback(
    async (xdr: string): Promise<string> => {
      if (!kitRef.current) throw new Error("Wallet not initialized");
      const result = await kitRef.current.signTransaction(xdr, {
        networkPassphrase: getNetworkPassphrase(),
        address: state.publicKey || undefined,
      });
      if (typeof result === "string") return result;
      if (result && typeof result === "object" && "signedTxXdr" in result) {
        return (result as { signedTxXdr: string }).signedTxXdr;
      }
      throw new Error("Unexpected response from wallet");
    },
    [state.publicKey]
  );

  return {
    ...state,
    connect,
    disconnect,
    signTransaction,
  };
}
