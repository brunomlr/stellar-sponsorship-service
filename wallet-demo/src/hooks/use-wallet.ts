"use client";

import { useCallback, useRef, useState } from "react";
import { Keypair, TransactionBuilder } from "@stellar/stellar-sdk";
import { getNetworkPassphrase } from "@/lib/stellar";

interface WalletState {
  connected: boolean;
  publicKey: string | null;
  error: string | null;
}

export function useWallet() {
  const keypairRef = useRef<Keypair | null>(null);
  const [state, setState] = useState<WalletState>({
    connected: false,
    publicKey: null,
    error: null,
  });

  const connectWithSecret = useCallback((secretKey: string) => {
    try {
      const kp = Keypair.fromSecret(secretKey);
      keypairRef.current = kp;
      setState({
        connected: true,
        publicKey: kp.publicKey(),
        error: null,
      });
    } catch {
      setState((s) => ({
        ...s,
        error: "Invalid secret key",
      }));
    }
  }, []);

  const generateAndConnect = useCallback(() => {
    const kp = Keypair.random();
    keypairRef.current = kp;
    setState({
      connected: true,
      publicKey: kp.publicKey(),
      error: null,
    });
    return { publicKey: kp.publicKey(), secretKey: kp.secret() };
  }, []);

  const disconnect = useCallback(() => {
    keypairRef.current = null;
    setState({
      connected: false,
      publicKey: null,
      error: null,
    });
  }, []);

  const signTransaction = useCallback(
    async (xdr: string): Promise<string> => {
      if (!keypairRef.current) throw new Error("No keypair loaded");
      const tx = TransactionBuilder.fromXDR(xdr, getNetworkPassphrase());
      tx.sign(keypairRef.current);
      return tx.toXDR();
    },
    []
  );

  return {
    ...state,
    connectWithSecret,
    generateAndConnect,
    disconnect,
    signTransaction,
  };
}
