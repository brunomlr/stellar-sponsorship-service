"use client";

import { useSession, signOut } from "next-auth/react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useHealth } from "@/hooks/use-api-keys";
import { useWalletContext } from "@/components/wallet-provider";
import { StellarAccount } from "@/components/stellar-account";
import { formatXLM } from "@/lib/utils";
import { Loader2, LogOut } from "lucide-react";

export default function SettingsPage() {
  const health = useHealth();
  const { data: session } = useSession();
  const { connected, publicKey } = useWalletContext();

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Settings</h2>
        <p className="text-muted-foreground">
          Service configuration and admin account.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Admin Account</CardTitle>
          <CardDescription>
            Signed in via Google Workspace.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label className="text-muted-foreground">Email</Label>
            <p>{session?.user?.email}</p>
          </div>
          {session?.user?.name && (
            <div className="space-y-2">
              <Label className="text-muted-foreground">Name</Label>
              <p>{session.user.name}</p>
            </div>
          )}
          <Button variant="outline" onClick={() => signOut()} className="gap-2">
            <LogOut className="h-4 w-4" />
            Sign Out
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Service Info</CardTitle>
          <CardDescription>
            Current service configuration and status.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {health.isLoading ? (
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          ) : health.data ? (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-muted-foreground">Status</Label>
                  <p>
                    <Badge variant="default">{health.data.status}</Badge>
                  </p>
                </div>
                <div>
                  <Label className="text-muted-foreground">Version</Label>
                  <p className="font-mono">{health.data.version}</p>
                </div>
                <div>
                  <Label className="text-muted-foreground">
                    Stellar Network
                  </Label>
                  <p>
                    <Badge variant="outline">
                      {health.data.stellar_network}
                    </Badge>
                  </p>
                </div>
                <div>
                  <Label className="text-muted-foreground">
                    Sponsor Accounts
                  </Label>
                  <p>{health.data.total_sponsor_accounts}</p>
                </div>
              </div>
              <Separator />
              <div>
                <Label className="text-muted-foreground">
                  Master Account Balance
                </Label>
                <p className="text-lg font-mono font-medium">
                  {formatXLM(health.data.master_account_balance)} XLM
                </p>
              </div>
            </>
          ) : (
            <p className="text-muted-foreground">
              Unable to connect to the service.
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Connected Wallet</CardTitle>
          <CardDescription>
            The wallet used to sign funding transactions via Freighter.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {connected && publicKey ? (
            <div className="space-y-2">
              <Label className="text-muted-foreground">Public Key</Label>
              <StellarAccount address={publicKey} truncateChars={false} />
            </div>
          ) : (
            <p className="text-muted-foreground">
              No wallet connected. Use the sidebar to connect.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
