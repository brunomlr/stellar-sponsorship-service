"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { SignTransaction } from "@/components/sign-transaction";
import { buildFundTransaction, submitFundTransaction } from "@/lib/api";
import { AlertTriangle, Check, Loader2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";

interface FundFormProps {
  apiKeyId: string;
  sponsorAccount: string;
}

type Step = "amount" | "sign" | "done";

export function FundForm({ apiKeyId, sponsorAccount }: FundFormProps) {
  const queryClient = useQueryClient();
  const [step, setStep] = useState<Step>("amount");
  const [amount, setAmount] = useState("");
  const [fundingXDR, setFundingXDR] = useState("");
  const [result, setResult] = useState<{
    xlm_available: string;
    transaction_hash: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleBuildFund = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await buildFundTransaction(apiKeyId, amount);
      setFundingXDR(res.funding_transaction_xdr);
      setStep("sign");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to build transaction");
    } finally {
      setLoading(false);
    }
  };

  const handleSigned = async (signedXDR: string) => {
    setError("");
    setLoading(true);
    try {
      const res = await submitFundTransaction(apiKeyId, signedXDR);
      setResult({
        xlm_available: res.xlm_available,
        transaction_hash: res.transaction_hash,
      });
      setStep("done");
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit transaction");
    } finally {
      setLoading(false);
    }
  };

  if (step === "done" && result) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Check className="h-5 w-5 text-green-600" />
            Funds Added
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm">
            New available balance:{" "}
            <span className="font-mono font-medium">
              {result.xlm_available} XLM
            </span>
          </p>
          <p className="text-sm text-muted-foreground font-mono break-all">
            TX: {result.transaction_hash}
          </p>
        </CardContent>
      </Card>
    );
  }

  if (step === "sign") {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Sign Funding Transaction</CardTitle>
          <CardDescription>
            Adding {amount} XLM to {sponsorAccount.slice(0, 8)}...
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <SignTransaction
            unsignedXDR={fundingXDR}
            onSigned={handleSigned}
            onError={(err) => setError(err.message)}
            label={`Sign & Send ${amount} XLM`}
            disabled={loading}
          />
          {loading && (
            <p className="text-sm text-muted-foreground flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Submitting...
            </p>
          )}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Add Funds</CardTitle>
        <CardDescription>
          Send additional XLM from the master account to this sponsor account.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="fund-amount">Amount (XLM)</Label>
          <Input
            id="fund-amount"
            type="text"
            placeholder="e.g. 500.0000000"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </div>
        {error && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <Button
          onClick={handleBuildFund}
          disabled={!amount || loading}
          className="w-full"
        >
          {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Build Fund Transaction
        </Button>
      </CardContent>
    </Card>
  );
}
