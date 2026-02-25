"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { FundForm } from "@/components/fund-form";
import { TransactionTable } from "@/components/transaction-table";
import { SignTransaction } from "@/components/sign-transaction";
import {
  useAPIKey,
  useUpdateAPIKey,
  useRegenerateAPIKey,
  useRevokeAPIKey,
  useBuildActivate,
  useSubmitActivate,
  useSweepFunds,
  useTransactions,
} from "@/hooks/use-api-keys";
import { StellarAccount } from "@/components/stellar-account";
import { formatXLM, formatDate } from "@/lib/utils";
import { SPONSORABLE_OPERATIONS } from "@/lib/constants";
import {
  Loader2,
  AlertTriangle,
  Trash2,
  ArrowDownToLine,
  Check,
  Pencil,
  Copy,
  RefreshCw,
} from "lucide-react";

export default function APIKeyDetailPage() {
  const params = useParams();
  const id = params.id as string;

  const { data: apiKey, isLoading } = useAPIKey(id);
  const txs = useTransactions({ api_key_id: id, per_page: 20 });
  const updateMutation = useUpdateAPIKey();
  const regenerateMutation = useRegenerateAPIKey();
  const revokeMutation = useRevokeAPIKey();
  const buildActivateMutation = useBuildActivate();
  const submitActivateMutation = useSubmitActivate();
  const sweepMutation = useSweepFunds();

  const [editOpen, setEditOpen] = useState(false);
  const [revokeOpen, setRevokeOpen] = useState(false);
  const [regenerateOpen, setRegenerateOpen] = useState(false);
  const [regeneratedKey, setRegeneratedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [sweepResult, setSweepResult] = useState<string | null>(null);
  const [unsignedXDR, setUnsignedXDR] = useState("");
  const [error, setError] = useState("");

  // Edit form state
  const [editName, setEditName] = useState("");
  const [editOps, setEditOps] = useState<string[]>([]);
  const [editExpiresAt, setEditExpiresAt] = useState("");
  const [editMaxRequests, setEditMaxRequests] = useState("");
  const [editWindowSeconds, setEditWindowSeconds] = useState("");
  const [editAllowedSources, setEditAllowedSources] = useState("");

  const openEditDialog = () => {
    if (!apiKey) return;
    setEditName(apiKey.name);
    setEditOps([...apiKey.allowed_operations]);
    setEditExpiresAt(apiKey.expires_at.split("T")[0]);
    setEditMaxRequests(String(apiKey.rate_limit_max));
    setEditWindowSeconds(String(apiKey.rate_limit_window));
    setEditAllowedSources(apiKey.allowed_source_accounts?.join("\n") || "");
    setEditOpen(true);
  };

  const toggleOp = (op: string) => {
    setEditOps((prev) =>
      prev.includes(op) ? prev.filter((o) => o !== op) : [...prev, op]
    );
  };

  const handleSave = async () => {
    if (!apiKey) return;
    setError("");
    try {
      const sources = editAllowedSources
        .split("\n")
        .map((s) => s.trim())
        .filter(Boolean);

      await updateMutation.mutateAsync({
        id,
        name: editName,
        allowed_operations: editOps,
        expires_at: new Date(editExpiresAt).toISOString(),
        rate_limit_max: parseInt(editMaxRequests) || 100,
        rate_limit_window: parseInt(editWindowSeconds) || 60,
        allowed_source_accounts: sources,
      });
      setEditOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update");
    }
  };

  const handleRegenerate = async () => {
    setError("");
    try {
      const result = await regenerateMutation.mutateAsync(id);
      setRegeneratedKey(result.api_key);
      setRegenerateOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to regenerate");
    }
  };

  const handleCopyKey = async () => {
    if (!regeneratedKey) return;
    await navigator.clipboard.writeText(regeneratedKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRevoke = async () => {
    try {
      await revokeMutation.mutateAsync(id);
      setRevokeOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke");
    }
  };

  const handleSweep = async () => {
    setError("");
    try {
      const result = await sweepMutation.mutateAsync(id);
      setSweepResult(
        `Swept ${result.xlm_swept} XLM. TX: ${result.transaction_hash}`
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to sweep");
    }
  };

  const handleBuildActivate = async () => {
    setError("");
    try {
      const result = await buildActivateMutation.mutateAsync(id);
      setUnsignedXDR(result.activate_transaction_xdr);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to build activation transaction"
      );
    }
  };

  const handleActivateSigned = async (signedXDR: string) => {
    setError("");
    try {
      await submitActivateMutation.mutateAsync({ id, signedXDR });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to activate");
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!apiKey) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertTitle>Not Found</AlertTitle>
        <AlertDescription>API key not found.</AlertDescription>
      </Alert>
    );
  }

  const canEdit = apiKey.status !== "revoked";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">{apiKey.name}</h2>
          <p className="text-muted-foreground font-mono text-sm">
            {apiKey.key_prefix}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {canEdit && (
            <Button variant="outline" size="sm" onClick={openEditDialog}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </Button>
          )}
          <Badge
            variant={
              apiKey.status === "active"
                ? "default"
                : apiKey.status === "revoked"
                  ? "destructive"
                  : "secondary"
            }
            className="text-sm"
          >
            {apiKey.status.replace("_", " ")}
          </Badge>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Regenerated key display */}
      {regeneratedKey && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">
              New API Key Generated
            </CardTitle>
            <CardDescription>
              This is the only time the new API key will be shown. Copy it now
              and store it securely. The previous key is now invalid.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded-md border bg-muted px-3 py-2 font-mono text-sm break-all">
                {regeneratedKey}
              </code>
              <Button variant="outline" size="icon" onClick={handleCopyKey}>
                {copied ? (
                  <Check className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {sweepResult && (
        <Alert>
          <Check className="h-4 w-4" />
          <AlertTitle>Sweep Complete</AlertTitle>
          <AlertDescription>{sweepResult}</AlertDescription>
        </Alert>
      )}

      {/* Info cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Sponsor Account
            </CardTitle>
          </CardHeader>
          <CardContent>
            {apiKey.sponsor_account ? (
              <StellarAccount
                address={apiKey.sponsor_account}
                truncateChars={false}
              />
            ) : (
              <p className="text-sm text-muted-foreground">Not yet created</p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">XLM Available</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {formatXLM(apiKey.xlm_available)} XLM
            </p>
            <p className="text-xs text-muted-foreground">
              Budget: {formatXLM(apiKey.xlm_budget)} XLM
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Expires</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-lg font-medium">
              {formatDate(apiKey.expires_at)}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Settings card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label className="text-muted-foreground">Allowed Operations</Label>
            <div className="flex flex-wrap gap-2">
              {apiKey.allowed_operations.map((op) => (
                <Badge key={op} variant="outline">
                  {op}
                </Badge>
              ))}
            </div>
          </div>
          <Separator />
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1">
              <Label className="text-muted-foreground">Rate Limit</Label>
              <p className="text-sm">
                {apiKey.rate_limit_max} requests / {apiKey.rate_limit_window}s
              </p>
            </div>
            <div className="space-y-1">
              <Label className="text-muted-foreground">
                Allowed Source Accounts
              </Label>
              {apiKey.allowed_source_accounts &&
              apiKey.allowed_source_accounts.length > 0 ? (
                <div className="space-y-1">
                  {apiKey.allowed_source_accounts.map((account) => (
                    <StellarAccount
                      key={account}
                      address={account}
                      truncateChars={false}
                    />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">Any</p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Activate (pending_funding) */}
      {apiKey.status === "pending_funding" && (
        <Card>
          <CardHeader>
            <CardTitle>Activate API Key</CardTitle>
            <CardDescription>
              Build and sign the activation transaction with your master account
              wallet.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {!unsignedXDR ? (
              <Button
                onClick={handleBuildActivate}
                disabled={buildActivateMutation.isPending}
              >
                {buildActivateMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Build Activation Transaction
              </Button>
            ) : (
              <>
                <SignTransaction
                  unsignedXDR={unsignedXDR}
                  onSigned={handleActivateSigned}
                  onError={(err) => setError(err.message)}
                  label="Sign & Activate"
                  disabled={submitActivateMutation.isPending}
                />
                {submitActivateMutation.isPending && (
                  <p className="text-sm text-muted-foreground flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Submitting to Stellar network...
                  </p>
                )}
              </>
            )}
          </CardContent>
        </Card>
      )}

      {/* Fund (active) */}
      {apiKey.status === "active" && (
        <FundForm
          apiKeyId={id}
          sponsorAccount={apiKey.sponsor_account}
        />
      )}

      {/* Danger zone: regenerate + revoke */}
      {canEdit && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Danger Zone</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-3">
              <Dialog
                open={regenerateOpen}
                onOpenChange={setRegenerateOpen}
              >
                <DialogTrigger asChild>
                  <Button variant="outline">
                    <RefreshCw className="mr-2 h-4 w-4" />
                    Regenerate API Key
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Regenerate API Key</DialogTitle>
                    <DialogDescription>
                      This will generate a new API key and invalidate the
                      current one. Any applications using the current key will
                      stop working immediately.
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setRegenerateOpen(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={handleRegenerate}
                      disabled={regenerateMutation.isPending}
                    >
                      {regenerateMutation.isPending && (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      )}
                      Regenerate
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>

              {apiKey.status === "active" && (
                <Dialog open={revokeOpen} onOpenChange={setRevokeOpen}>
                  <DialogTrigger asChild>
                    <Button variant="destructive">
                      <Trash2 className="mr-2 h-4 w-4" />
                      Revoke API Key
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Revoke API Key</DialogTitle>
                      <DialogDescription>
                        This will permanently deactivate the API key. The
                        sponsor account and its funds will remain on-chain. You
                        can sweep available funds afterward.
                      </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                      <Button
                        variant="outline"
                        onClick={() => setRevokeOpen(false)}
                      >
                        Cancel
                      </Button>
                      <Button
                        variant="destructive"
                        onClick={handleRevoke}
                        disabled={revokeMutation.isPending}
                      >
                        {revokeMutation.isPending && (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        )}
                        Revoke
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Sweep (revoked) */}
      {apiKey.status === "revoked" && (
        <Card>
          <CardHeader>
            <CardTitle>Sweep Funds</CardTitle>
            <CardDescription>
              Transfer available (unlocked) XLM from the revoked sponsor account
              back to the master funding account. Signed automatically by the
              service.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleSweep}
              disabled={sweepMutation.isPending}
            >
              {sweepMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              <ArrowDownToLine className="mr-2 h-4 w-4" />
              Sweep Funds to Master
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Transactions */}
      <Separator />
      <Card>
        <CardHeader>
          <CardTitle>Recent Transactions</CardTitle>
          <CardDescription>
            Transaction signing history for this API key.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {txs.isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <TransactionTable
              transactions={txs.data?.transactions || []}
            />
          )}
        </CardContent>
      </Card>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit API Key</DialogTitle>
            <DialogDescription>
              Update the settings for this API key.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>Allowed Operations</Label>
              <div className="flex flex-wrap gap-2">
                {SPONSORABLE_OPERATIONS.map((op) => (
                  <Badge
                    key={op}
                    variant={editOps.includes(op) ? "default" : "outline"}
                    className="cursor-pointer"
                    onClick={() => toggleOp(op)}
                  >
                    {op}
                  </Badge>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-expires">Expires At</Label>
              <Input
                id="edit-expires"
                type="date"
                value={editExpiresAt}
                onChange={(e) => setEditExpiresAt(e.target.value)}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="edit-max-requests">
                  Rate Limit (max requests)
                </Label>
                <Input
                  id="edit-max-requests"
                  type="number"
                  value={editMaxRequests}
                  onChange={(e) => setEditMaxRequests(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-window">Window (seconds)</Label>
                <Input
                  id="edit-window"
                  type="number"
                  value={editWindowSeconds}
                  onChange={(e) => setEditWindowSeconds(e.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-sources">
                Allowed Source Accounts (optional)
              </Label>
              <Textarea
                id="edit-sources"
                placeholder="One Stellar public key per line (G...)"
                value={editAllowedSources}
                onChange={(e) => setEditAllowedSources(e.target.value)}
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                If set, only these accounts can appear as transaction/operation
                source.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSave}
              disabled={
                updateMutation.isPending ||
                editOps.length === 0 ||
                !editName ||
                !editExpiresAt
              }
            >
              {updateMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
