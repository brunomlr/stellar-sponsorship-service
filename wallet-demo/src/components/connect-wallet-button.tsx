"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useWalletContext } from "@/components/wallet-provider";
import { truncateKey } from "@/lib/utils";
import { AlertTriangle, Copy, Key, LogOut } from "lucide-react";

export function ConnectWalletButton() {
  const {
    connected,
    publicKey,
    error,
    connectWithSecret,
    generateAndConnect,
    disconnect,
  } = useWalletContext();

  const [secretInput, setSecretInput] = useState("");
  const [generatedSecret, setGeneratedSecret] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [showInput, setShowInput] = useState(false);

  if (connected && publicKey) {
    return (
      <div className="flex items-center gap-2">
        <span className="text-sm font-mono text-muted-foreground">
          {truncateKey(publicKey)}
        </span>
        <Button variant="ghost" size="icon" onClick={disconnect}>
          <LogOut className="h-4 w-4" />
        </Button>
      </div>
    );
  }

  const handleGenerate = () => {
    const { secretKey } = generateAndConnect();
    setGeneratedSecret(secretKey);
  };

  const handleImport = () => {
    connectWithSecret(secretInput);
    setSecretInput("");
  };

  const copySecret = async () => {
    if (!generatedSecret) return;
    await navigator.clipboard.writeText(generatedSecret);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (generatedSecret) {
    return (
      <div className="w-full max-w-md space-y-3">
        <Alert variant="destructive" className="border-orange-300 bg-orange-50 text-orange-800">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            Save this secret key. It will not be shown again after you dismiss.
          </AlertDescription>
        </Alert>
        <div className="flex gap-2">
          <Input
            value={generatedSecret}
            readOnly
            className="font-mono text-xs"
          />
          <Button variant="outline" size="icon" onClick={copySecret}>
            <Copy className="h-4 w-4" />
          </Button>
        </div>
        {copied && (
          <p className="text-xs text-green-600">Copied to clipboard</p>
        )}
        <Button
          className="w-full"
          onClick={() => setGeneratedSecret(null)}
        >
          Continue
        </Button>
      </div>
    );
  }

  return (
    <div className="w-full max-w-md space-y-3">
      <Button onClick={handleGenerate} className="w-full">
        <Key className="mr-2 h-4 w-4" />
        Generate New Keypair
      </Button>

      {!showInput ? (
        <Button
          variant="outline"
          className="w-full"
          onClick={() => setShowInput(true)}
        >
          Import Existing Secret Key
        </Button>
      ) : (
        <div className="space-y-2">
          <Input
            type="password"
            placeholder="S..."
            value={secretInput}
            onChange={(e) => setSecretInput(e.target.value)}
            className="font-mono text-sm"
          />
          <div className="flex gap-2">
            <Button
              onClick={handleImport}
              disabled={!secretInput || !secretInput.startsWith("S")}
              className="flex-1"
            >
              Import
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setShowInput(false);
                setSecretInput("");
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {error && (
        <p className="text-sm text-destructive text-center">{error}</p>
      )}
    </div>
  );
}
