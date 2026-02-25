"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { APIKeyTable } from "@/components/api-key-table";
import { useAPIKeys } from "@/hooks/use-api-keys";
import { Loader2, Plus } from "lucide-react";
import { useState } from "react";

export default function APIKeysPage() {
  const [page, setPage] = useState(1);
  const perPage = 20;
  const { data, isLoading } = useAPIKeys(page, perPage);

  const totalPages = data ? Math.ceil(data.total / perPage) : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">API Keys</h2>
          <p className="text-muted-foreground">
            Manage API keys and their sponsor accounts.
          </p>
        </div>
        <Button asChild>
          <Link href="/api-keys/new">
            <Plus className="mr-2 h-4 w-4" />
            Create API Key
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>All API Keys</CardTitle>
          <CardDescription>
            {data ? `${data.total} total` : "Loading..."}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <>
              <APIKeyTable apiKeys={data?.api_keys || []} />
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <p className="text-sm text-muted-foreground">
                    Page {page} of {totalPages}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page <= 1}
                      onClick={() => setPage((p) => p - 1)}
                    >
                      Previous
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page >= totalPages}
                      onClick={() => setPage((p) => p + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
