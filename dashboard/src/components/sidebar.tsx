"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useWalletContext } from "@/components/wallet-provider";
import { StellarAccount } from "@/components/stellar-account";
import { truncateKey } from "@/lib/utils";
import { useHealth } from "@/hooks/use-api-keys";
import {
  LayoutDashboard,
  Key,
  ArrowLeftRight,
  Settings,
  Wallet,
  LogOut,
} from "lucide-react";

const navItems = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/api-keys", label: "API Keys", icon: Key },
  { href: "/transactions", label: "Transactions", icon: ArrowLeftRight },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const { data: session } = useSession();
  const { connected, publicKey, connect, disconnect, connecting } =
    useWalletContext();
  const health = useHealth();

  return (
    <div className="flex h-full w-64 flex-col border-r bg-card">
      <div className="p-6">
        <h1 className="text-lg font-semibold">Sponsorship Service</h1>
        <p className="text-sm text-muted-foreground">Admin Dashboard</p>
      </div>

      <nav className="flex-1 space-y-1 px-3">
        {navItems.map((item) => {
          const isActive =
            pathname === item.href ||
            (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link key={item.href} href={item.href}>
              <Button
                variant={isActive ? "secondary" : "ghost"}
                className={cn("w-full justify-start gap-2")}
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </Button>
            </Link>
          );
        })}
      </nav>

      <div className="border-t p-4 space-y-2">
        {connected && publicKey ? (
          <>
            <div className="text-xs text-muted-foreground">
              Connected Wallet
            </div>
            <StellarAccount address={publicKey} truncateChars={6} />
            <Button
              variant="outline"
              size="sm"
              className="w-full gap-2"
              onClick={disconnect}
            >
              <LogOut className="h-3 w-3" />
              Disconnect
            </Button>
          </>
        ) : (
          <Button
            variant="outline"
            className="w-full gap-2"
            onClick={connect}
            disabled={connecting}
          >
            <Wallet className="h-4 w-4" />
            {connecting
              ? "Connecting..."
              : health.data?.master_public_key
                ? `Connect Wallet (${health.data.master_public_key.slice(0, 5)}...)`
                : "Connect Wallet"}
          </Button>
        )}
      </div>

      {session?.user && (
        <div className="border-t p-4 space-y-2">
          <div className="text-xs text-muted-foreground">Signed in as</div>
          <div className="text-sm truncate">{session.user.email}</div>
          <Button
            variant="ghost"
            size="sm"
            className="w-full gap-2"
            onClick={() => signOut()}
          >
            <LogOut className="h-3 w-3" />
            Sign Out
          </Button>
        </div>
      )}
    </div>
  );
}
