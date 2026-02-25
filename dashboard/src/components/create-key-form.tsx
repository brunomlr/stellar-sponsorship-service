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
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { SignTransaction } from "@/components/sign-transaction";
import { StellarAccount } from "@/components/stellar-account";
import { useCreateAPIKey, useBuildActivate, useSubmitActivate } from "@/hooks/use-api-keys";
import { Copy, Check, AlertTriangle, Loader2 } from "lucide-react";
import { SPONSORABLE_OPERATIONS } from "@/lib/constants";

type Step = "form" | "api_key" | "sign" | "done";

export function CreateKeyForm() {
  const [step, setStep] = useState<Step>("form");

  // Form state
  const [name, setName] = useState("");
  const [xlmBudget, setXlmBudget] = useState("");
  const [selectedOps, setSelectedOps] = useState<string[]>([]);
  const [expiresAt, setExpiresAt] = useState("");
  const [maxRequests, setMaxRequests] = useState("100");
  const [windowSeconds, setWindowSeconds] = useState("60");
  const [allowedSources, setAllowedSources] = useState("");

  // Result state
  const [apiKeyResult, setApiKeyResult] = useState<{
    id: string;
    api_key: string;
  } | null>(null);
  const [sponsorAccount, setSponsorAccount] = useState("");
  const [unsignedXDR, setUnsignedXDR] = useState("");
  const [copied, setCopied] = useState(false);
  const [activationHash, setActivationHash] = useState("");
  const [error, setError] = useState("");

  const createMutation = useCreateAPIKey();
  const buildActivateMutation = useBuildActivate();
  const submitActivateMutation = useSubmitActivate();

  const toggleOp = (op: string) => {
    setSelectedOps((prev) =>
      prev.includes(op) ? prev.filter((o) => o !== op) : [...prev, op]
    );
  };

  const handleSubmit = async () => {
    setError("");
    try {
      const sources = allowedSources
        .split("\n")
        .map((s) => s.trim())
        .filter(Boolean);

      const result = await createMutation.mutateAsync({
        name,
        xlm_budget: xlmBudget,
        allowed_operations: selectedOps,
        expires_at: new Date(expiresAt).toISOString(),
        rate_limit: {
          max_requests: parseInt(maxRequests) || 100,
          window_seconds: parseInt(windowSeconds) || 60,
        },
        ...(sources.length > 0 && { allowed_source_accounts: sources }),
      });

      setApiKeyResult({
        id: result.id,
        api_key: result.api_key,
      });
      setStep("api_key");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create API key");
    }
  };

  const handleCopy = async () => {
    if (!apiKeyResult) return;
    await navigator.clipboard.writeText(apiKeyResult.api_key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleContinueToSign = async () => {
    if (!apiKeyResult) return;
    setError("");
    try {
      const result = await buildActivateMutation.mutateAsync(apiKeyResult.id);
      setSponsorAccount(result.sponsor_account);
      setUnsignedXDR(result.activate_transaction_xdr);
      setStep("sign");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to build activation transaction"
      );
    }
  };

  const handleSigned = async (signedXDR: string) => {
    if (!apiKeyResult) return;
    setError("");
    try {
      const result = await submitActivateMutation.mutateAsync({
        id: apiKeyResult.id,
        signedXDR,
      });
      setActivationHash(result.transaction_hash);
      setStep("done");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to activate API key"
      );
    }
  };

  if (step === "done") {
    return (
      <Card>
        <CardHeader>
          <CardTitle>API Key Created Successfully</CardTitle>
          <CardDescription>
            The sponsor account has been funded and the API key is now active.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Sponsor Account</Label>
            <StellarAccount address={sponsorAccount} truncateChars={false} />
          </div>
          <div className="space-y-2">
            <Label>Funding Transaction Hash</Label>
            <p className="font-mono text-sm break-all">{activationHash}</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (step === "sign" && apiKeyResult) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Sign Funding Transaction</CardTitle>
          <CardDescription>
            Sign the funding transaction with your master account wallet to
            create and fund the sponsor account.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Sponsor Account</Label>
            <StellarAccount address={sponsorAccount} truncateChars={false} />
          </div>
          <div className="space-y-2">
            <Label>XLM Budget</Label>
            <p>{xlmBudget} XLM</p>
          </div>
          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <SignTransaction
            unsignedXDR={unsignedXDR}
            onSigned={handleSigned}
            onError={(err) => setError(err.message)}
            label="Sign & Fund Sponsor Account"
            disabled={submitActivateMutation.isPending}
          />
          {submitActivateMutation.isPending && (
            <p className="text-sm text-muted-foreground flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              Submitting to Stellar network...
            </p>
          )}
        </CardContent>
      </Card>
    );
  }

  if (step === "api_key" && apiKeyResult) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Save Your API Key</CardTitle>
          <CardDescription>
            This is the only time the full API key will be shown. Copy it now and
            store it securely.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert>
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Important</AlertTitle>
            <AlertDescription>
              This API key will not be shown again. Make sure to copy and store
              it securely before proceeding.
            </AlertDescription>
          </Alert>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-muted px-3 py-2 font-mono text-sm break-all">
              {apiKeyResult.api_key}
            </code>
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? (
                <Check className="h-4 w-4" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
          <Separator />
          <Button
            onClick={handleContinueToSign}
            disabled={buildActivateMutation.isPending}
            className="w-full"
          >
            {buildActivateMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            I have saved my API key — Continue to funding
          </Button>
        </CardContent>
      </Card>
    );
  }

  // step === "form"
  return (
    <Card>
      <CardHeader>
        <CardTitle>Create API Key</CardTitle>
        <CardDescription>
          Provision a new API key with a dedicated sponsor account.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            placeholder="e.g. Wallet Co"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="xlm-budget">XLM Budget</Label>
          <Input
            id="xlm-budget"
            type="text"
            placeholder="e.g. 1000.0000000"
            value={xlmBudget}
            onChange={(e) => setXlmBudget(e.target.value)}
          />
          <p className="text-sm text-muted-foreground">
            Amount of XLM to fund the sponsor account with.
          </p>
        </div>

        <div className="space-y-2">
          <Label>Allowed Operations</Label>
          <div className="flex flex-wrap gap-2">
            {SPONSORABLE_OPERATIONS.map((op) => (
              <Badge
                key={op}
                variant={selectedOps.includes(op) ? "default" : "outline"}
                className="cursor-pointer"
                onClick={() => toggleOp(op)}
              >
                {op}
              </Badge>
            ))}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="expires-at">Expires At</Label>
          <Input
            id="expires-at"
            type="date"
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="max-requests">Rate Limit (max requests)</Label>
            <Input
              id="max-requests"
              type="number"
              value={maxRequests}
              onChange={(e) => setMaxRequests(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="window-seconds">Window (seconds)</Label>
            <Input
              id="window-seconds"
              type="number"
              value={windowSeconds}
              onChange={(e) => setWindowSeconds(e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="allowed-sources">
            Allowed Source Accounts (optional)
          </Label>
          <Textarea
            id="allowed-sources"
            placeholder="One Stellar public key per line (G...)"
            value={allowedSources}
            onChange={(e) => setAllowedSources(e.target.value)}
            rows={3}
          />
          <p className="text-sm text-muted-foreground">
            If set, only these accounts can appear as transaction/operation
            source.
          </p>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Button
          onClick={handleSubmit}
          disabled={
            !name ||
            !xlmBudget ||
            selectedOps.length === 0 ||
            !expiresAt ||
            createMutation.isPending
          }
          className="w-full"
        >
          {createMutation.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          Create API Key
        </Button>
      </CardContent>
    </Card>
  );
}
