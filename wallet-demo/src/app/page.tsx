"use client";

import { useWalletContext } from "@/components/wallet-provider";
import { ConnectWalletButton } from "@/components/connect-wallet-button";
import { AccountInfo } from "@/components/account-info";
import { SponsorInfo } from "@/components/sponsor-info";
import { AddTrustlineForm } from "@/components/add-trustline-form";
import { CreateAccountForm } from "@/components/create-account-form";
import { TransactionHistory } from "@/components/transaction-history";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useState } from "react";
import type { TransactionRecord } from "@/types";

export default function WalletDemoPage() {
  const { connected } = useWalletContext();
  const [transactions, setTransactions] = useState<TransactionRecord[]>([]);

  const addTransaction = (tx: TransactionRecord) => {
    setTransactions((prev) => [tx, ...prev]);
  };

  const updateTransaction = (
    id: string,
    updates: Partial<TransactionRecord>
  ) => {
    setTransactions((prev) =>
      prev.map((tx) => (tx.id === id ? { ...tx, ...updates } : tx))
    );
  };

  if (!connected) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] space-y-6">
        <div className="text-center space-y-2">
          <h2 className="text-3xl font-bold tracking-tight">
            Sponsorship Demo Wallet
          </h2>
          <p className="text-muted-foreground max-w-md">
            Generate a new keypair or import an existing secret key to try
            sponsored transactions. The sponsor covers network reserves so you
            don&apos;t need to hold extra XLM.
          </p>
        </div>
        <ConnectWalletButton />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2">
        <AccountInfo />
        <SponsorInfo />
      </div>

      <Tabs defaultValue="trustline">
        <TabsList>
          <TabsTrigger value="trustline">Add Trustline</TabsTrigger>
          <TabsTrigger value="create-account">Create Account</TabsTrigger>
        </TabsList>
        <TabsContent value="trustline">
          <AddTrustlineForm
            onTransaction={addTransaction}
            onTransactionUpdate={updateTransaction}
          />
        </TabsContent>
        <TabsContent value="create-account">
          <CreateAccountForm
            onTransaction={addTransaction}
            onTransactionUpdate={updateTransaction}
          />
        </TabsContent>
      </Tabs>

      {transactions.length > 0 && (
        <TransactionHistory transactions={transactions} />
      )}
    </div>
  );
}
