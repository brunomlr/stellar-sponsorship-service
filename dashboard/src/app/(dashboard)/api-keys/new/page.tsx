"use client";

import { CreateKeyForm } from "@/components/create-key-form";

export default function NewAPIKeyPage() {
  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Create API Key</h2>
        <p className="text-muted-foreground">
          Provision a new API key with a dedicated sponsor account.
        </p>
      </div>
      <CreateKeyForm />
    </div>
  );
}
