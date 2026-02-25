"use client";

import { useState } from "react";
import * as StellarSdk from "@stellar/stellar-sdk";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { TransactionStatus } from "@/components/transaction-status";
import { useWalletContext } from "@/components/wallet-provider";
import { useSponsorUsage } from "@/hooks/use-sponsor";
import { signTransaction as sponsorSign } from "@/lib/api";
import {
  buildCreateAccountTransaction,
  submitTransaction,
} from "@/lib/transaction-builder";
import {
  getExplorerUrl,
  getExplorerAccountUrl,
  truncateKey,
} from "@/lib/utils";
import { getNetworkPassphrase, getStellarNetwork } from "@/lib/stellar";
import type { TransactionRecord } from "@/types";
import { AlertTriangle, Copy, ExternalLink, Key, RotateCcw } from "lucide-react";

type Step =
  | "configure"
  | "building"
  | "signing_sponsor"
  | "signing_wallet"
  | "signing_new_account"
  | "submitting"
  | "success"
  | "error";

type ErrorAt = "building" | "signing_sponsor" | "signing_wallet" | "signing_new_account" | "submitting";

interface CreateAccountFormProps {
  onTransaction: (tx: TransactionRecord) => void;
  onTransactionUpdate: (
    id: string,
    updates: Partial<TransactionRecord>
  ) => void;
}

export function CreateAccountForm({
  onTransaction,
  onTransactionUpdate,
}: CreateAccountFormProps) {
  const { publicKey, signTransaction: walletSign } = useWalletContext();
  const { data: usage } = useSponsorUsage();

  const [newPublicKey, setNewPublicKey] = useState("");
  const [newSecretKey, setNewSecretKey] = useState("");
  const [keypairGenerated, setKeypairGenerated] = useState(false);
  const [secretAcknowledged, setSecretAcknowledged] = useState(false);
  const [step, setStep] = useState<Step>("configure");
  const [errorAt, setErrorAt] = useState<ErrorAt | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [txHash, setTxHash] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const reset = () => {
    setStep("configure");
    setErrorAt(null);
    setError(null);
    setTxHash(null);
    setNewPublicKey("");
    setNewSecretKey("");
    setKeypairGenerated(false);
    setSecretAcknowledged(false);
    setCopied(false);
  };

  const generateKeypair = () => {
    const pair = StellarSdk.Keypair.random();
    setNewPublicKey(pair.publicKey());
    setNewSecretKey(pair.secret());
    setKeypairGenerated(true);
    setSecretAcknowledged(false);
  };

  const copySecret = async () => {
    await navigator.clipboard.writeText(newSecretKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const fail = (at: ErrorAt, id: string, err: unknown) => {
    const message = err instanceof Error ? err.message : "Transaction failed";
    setError(message);
    setErrorAt(at);
    setStep("error");
    onTransactionUpdate(id, { status: "failed", error: message });
  };

  const handleSubmit = async () => {
    if (!publicKey || !usage || !newPublicKey) return;

    const id = crypto.randomUUID();
    setError(null);
    setErrorAt(null);

    onTransaction({
      id,
      type: "create_account",
      description: `Create account ${truncateKey(newPublicKey)}`,
      status: "building",
      timestamp: new Date(),
    });

    try {
      // Step 1: Build unsigned transaction
      setStep("building");
      const unsignedXdr = await buildCreateAccountTransaction(
        publicKey,
        usage.sponsor_account,
        newPublicKey
      );

      // Step 2: Get sponsor signature
      setStep("signing_sponsor");
      onTransactionUpdate(id, { status: "signing_sponsor" });
      let signedXdr: string;
      try {
        const result = await sponsorSign(unsignedXdr);
        signedXdr = result.signed_transaction_xdr;
      } catch (err) {
        return fail("signing_sponsor", id, err);
      }

      // Step 3: Sign with the source account (wallet)
      setStep("signing_wallet");
      onTransactionUpdate(id, { status: "signing_wallet" });
      try {
        signedXdr = await walletSign(signedXdr);
      } catch (err) {
        return fail("signing_wallet", id, err);
      }

      // Step 4: Sign with the new account keypair (for END_SPONSORING)
      setStep("signing_new_account");
      try {
        if (newSecretKey) {
          const newKp = StellarSdk.Keypair.fromSecret(newSecretKey);
          const tx = StellarSdk.TransactionBuilder.fromXDR(
            signedXdr,
            getNetworkPassphrase()
          );
          tx.sign(newKp);
          signedXdr = tx.toXDR();
        }
      } catch (err) {
        return fail("signing_new_account", id, err);
      }

      // Step 5: Submit to Horizon
      setStep("submitting");
      onTransactionUpdate(id, { status: "submitting" });
      try {
        const result = await submitTransaction(signedXdr);
        setTxHash(result.hash);
        setStep("success");
        onTransactionUpdate(id, { status: "success", txHash: result.hash });
      } catch (err) {
        return fail("submitting", id, err);
      }
    } catch (err) {
      fail("building", id, err);
    }
  };

  const sponsorPrefix = usage?.sponsor_account?.slice(0, 5) ?? "...";
  const walletPrefix = publicKey?.slice(0, 5) ?? "...";
  const newAccPrefix = newPublicKey?.slice(0, 5) ?? "...";

  type StepState = "pending" | "active" | "done" | "error";
  const allSteps: Step[] = ["building", "signing_sponsor", "signing_wallet", "signing_new_account", "submitting"];

  function getState(s: Step): StepState {
    if (step === "error" && errorAt === s) return "error";
    if (step === s) return "active";
    const curIdx = allSteps.indexOf(step === "error" ? errorAt! : step === "success" ? "submitting" : step);
    const sIdx = allSteps.indexOf(s);
    if (sIdx < curIdx) return "done";
    if (step === "success" && s === "submitting") return "done";
    return "pending";
  }

  const steps: { label: string; state: StepState }[] = [
    { label: "Build transaction", state: getState("building") },
    { label: `Sponsor signature (${sponsorPrefix}...)`, state: getState("signing_sponsor") },
    { label: `Source signature (${walletPrefix}...)`, state: getState("signing_wallet") },
    { label: `New account signature (${newAccPrefix}...)`, state: getState("signing_new_account") },
    { label: "Submit to network", state: getState("submitting") },
  ];

  const network = getStellarNetwork();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Create Account</CardTitle>
        <CardDescription>
          Create a new Stellar account. The sponsor covers the 1 XLM base
          reserve.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {step === "configure" && (
          <>
            {!keypairGenerated ? (
              <div className="space-y-3">
                <p className="text-sm text-muted-foreground">
                  Generate a new Stellar keypair or enter an existing public key
                  for the new account.
                </p>
                <Button
                  variant="outline"
                  onClick={generateKeypair}
                  className="w-full"
                >
                  <Key className="mr-2 h-4 w-4" />
                  Generate New Keypair
                </Button>
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-card px-2 text-muted-foreground">
                      or
                    </span>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="new-public-key">
                    Existing Public Key
                  </Label>
                  <Input
                    id="new-public-key"
                    placeholder="G..."
                    value={newPublicKey}
                    onChange={(e) => {
                      setNewPublicKey(e.target.value);
                      setSecretAcknowledged(true);
                    }}
                    className="font-mono text-sm"
                  />
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                <div className="space-y-2">
                  <Label>New Public Key</Label>
                  <p className="text-sm font-mono break-all bg-muted p-2 rounded">
                    {newPublicKey}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>Secret Key</Label>
                  <Alert variant="destructive" className="border-orange-300 bg-orange-50 text-orange-800">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>
                      Save this secret key now. It will not be shown again.
                    </AlertDescription>
                  </Alert>
                  <div className="flex gap-2">
                    <Input
                      value={newSecretKey}
                      readOnly
                      className="font-mono text-xs"
                    />
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={copySecret}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                  {copied && (
                    <p className="text-xs text-green-600">
                      Copied to clipboard
                    </p>
                  )}
                </div>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    checked={secretAcknowledged}
                    onChange={(e) => setSecretAcknowledged(e.target.checked)}
                    className="rounded"
                  />
                  I have saved the secret key
                </label>
              </div>
            )}

            <Button
              onClick={handleSubmit}
              disabled={
                !newPublicKey ||
                newPublicKey.length !== 56 ||
                !secretAcknowledged ||
                !publicKey ||
                !usage
              }
              className="w-full"
            >
              Create Account
            </Button>
          </>
        )}

        {step !== "configure" && (
          <TransactionStatus steps={steps} error={error} />
        )}

        {step === "success" && txHash && (
          <div className="space-y-3">
            <div className="p-3 rounded-md bg-green-50 text-green-800 text-sm">
              Account created successfully!
            </div>
            <div>
              <p className="text-xs text-muted-foreground">New Account</p>
              <a
                href={getExplorerAccountUrl(network, newPublicKey)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm font-mono text-primary hover:underline inline-flex items-center gap-1"
              >
                {truncateKey(newPublicKey, 12)}
                <ExternalLink className="h-3 w-3" />
              </a>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Transaction Hash</p>
              <a
                href={getExplorerUrl(network, txHash)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm font-mono text-primary hover:underline inline-flex items-center gap-1"
              >
                {txHash.slice(0, 16)}...
                <ExternalLink className="h-3 w-3" />
              </a>
            </div>
            <Button variant="outline" onClick={reset} className="w-full">
              <RotateCcw className="mr-2 h-4 w-4" />
              Create Another Account
            </Button>
          </div>
        )}

        {step === "error" && (
          <Button variant="outline" onClick={reset} className="w-full">
            <RotateCcw className="mr-2 h-4 w-4" />
            Try Again
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
