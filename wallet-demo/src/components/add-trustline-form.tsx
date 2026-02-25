"use client";

import { useState } from "react";
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
import { Badge } from "@/components/ui/badge";
import { TransactionStatus } from "@/components/transaction-status";
import { useWalletContext } from "@/components/wallet-provider";
import { useSponsorUsage } from "@/hooks/use-sponsor";
import { signTransaction as sponsorSign } from "@/lib/api";
import {
  buildAddTrustlineTransaction,
  submitTransaction,
} from "@/lib/transaction-builder";
import { getExplorerUrl } from "@/lib/utils";
import { getStellarNetwork } from "@/lib/stellar";
import type { TransactionRecord } from "@/types";
import { ExternalLink, RotateCcw } from "lucide-react";

// Well-known testnet assets for quick selection
const TESTNET_ASSETS = [
  {
    code: "USDC",
    issuer: "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5",
    label: "USDC",
  },
  {
    code: "SRT",
    issuer: "GCDNJUBQSX7AJWLJACMJ7I4BC3Z47BQUTMHEICZLE6MU4KQBRYG5JY6B",
    label: "SRT",
  },
];

type Step =
  | "configure"
  | "building"
  | "signing_sponsor"
  | "signing_wallet"
  | "submitting"
  | "success"
  | "error";

type ErrorAt = "building" | "signing_sponsor" | "signing_wallet" | "submitting";

interface AddTrustlineFormProps {
  onTransaction: (tx: TransactionRecord) => void;
  onTransactionUpdate: (id: string, updates: Partial<TransactionRecord>) => void;
}

export function AddTrustlineForm({
  onTransaction,
  onTransactionUpdate,
}: AddTrustlineFormProps) {
  const { publicKey, signTransaction: walletSign } = useWalletContext();
  const { data: usage } = useSponsorUsage();

  const [assetCode, setAssetCode] = useState("");
  const [assetIssuer, setAssetIssuer] = useState("");
  const [step, setStep] = useState<Step>("configure");
  const [errorAt, setErrorAt] = useState<ErrorAt | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [txHash, setTxHash] = useState<string | null>(null);

  const reset = () => {
    setStep("configure");
    setErrorAt(null);
    setError(null);
    setTxHash(null);
    setAssetCode("");
    setAssetIssuer("");
  };

  const selectPreset = (code: string, issuer: string) => {
    setAssetCode(code);
    setAssetIssuer(issuer);
  };

  const fail = (at: ErrorAt, id: string, err: unknown) => {
    const message = err instanceof Error ? err.message : "Transaction failed";
    setError(message);
    setErrorAt(at);
    setStep("error");
    onTransactionUpdate(id, { status: "failed", error: message });
  };

  const handleSubmit = async () => {
    if (!publicKey || !usage) return;

    const id = crypto.randomUUID();
    setError(null);
    setErrorAt(null);

    onTransaction({
      id,
      type: "add_trustline",
      description: `Add ${assetCode} trustline`,
      status: "building",
      timestamp: new Date(),
    });

    try {
      // Step 1: Build unsigned transaction
      setStep("building");
      const unsignedXdr = await buildAddTrustlineTransaction(
        publicKey,
        usage.sponsor_account,
        assetCode,
        assetIssuer
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

      // Step 3: Get user wallet signature
      setStep("signing_wallet");
      onTransactionUpdate(id, { status: "signing_wallet" });
      try {
        signedXdr = await walletSign(signedXdr);
      } catch (err) {
        return fail("signing_wallet", id, err);
      }

      // Step 4: Submit to Horizon
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

  type StepState = "pending" | "active" | "done" | "error";
  const allSteps: Step[] = ["building", "signing_sponsor", "signing_wallet", "submitting"];

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
    { label: "Submit to network", state: getState("submitting") },
  ];

  const network = getStellarNetwork();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Add Trustline</CardTitle>
        <CardDescription>
          Add a trustline to your account. The sponsor covers the 0.5 XLM
          reserve.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {step === "configure" && (
          <>
            <div>
              <p className="text-xs text-muted-foreground mb-2">
                Quick select (testnet)
              </p>
              <div className="flex gap-2">
                {TESTNET_ASSETS.map((asset) => (
                  <Badge
                    key={asset.code}
                    variant={assetCode === asset.code ? "default" : "outline"}
                    className="cursor-pointer"
                    onClick={() => selectPreset(asset.code, asset.issuer)}
                  >
                    {asset.label}
                  </Badge>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="asset-code">Asset Code</Label>
              <Input
                id="asset-code"
                placeholder="e.g. USDC"
                value={assetCode}
                onChange={(e) => setAssetCode(e.target.value.toUpperCase())}
                maxLength={12}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="asset-issuer">Asset Issuer</Label>
              <Input
                id="asset-issuer"
                placeholder="G..."
                value={assetIssuer}
                onChange={(e) => setAssetIssuer(e.target.value)}
                className="font-mono text-sm"
              />
            </div>
            <Button
              onClick={handleSubmit}
              disabled={
                !assetCode ||
                !assetIssuer ||
                assetIssuer.length !== 56 ||
                !publicKey ||
                !usage
              }
              className="w-full"
            >
              Add Trustline
            </Button>
          </>
        )}

        {step !== "configure" && (
          <TransactionStatus steps={steps} error={error} />
        )}

        {step === "success" && txHash && (
          <div className="space-y-3">
            <div className="p-3 rounded-md bg-green-50 text-green-800 text-sm">
              Trustline added successfully!
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
              Add Another Trustline
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
