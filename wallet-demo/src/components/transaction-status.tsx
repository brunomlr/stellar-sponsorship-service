"use client";

import { Check, Loader2, AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

type StepState = "pending" | "active" | "done" | "error";

interface Step {
  label: string;
  state: StepState;
}

interface TransactionStatusProps {
  steps: Step[];
  error?: string | null;
}

export function TransactionStatus({ steps, error }: TransactionStatusProps) {
  return (
    <div className="space-y-2">
      {steps.map((step, i) => (
        <div key={i} className="flex items-center gap-2">
          <StepIcon state={step.state} />
          <span
            className={cn(
              "text-sm",
              step.state === "pending" && "text-muted-foreground",
              step.state === "active" && "font-medium",
              step.state === "done" && "text-muted-foreground line-through",
              step.state === "error" && "text-destructive"
            )}
          >
            {step.label}
          </span>
        </div>
      ))}
      {error && (
        <div className="flex items-start gap-2 mt-2 p-3 rounded-md bg-destructive/10 text-destructive text-sm">
          <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
          <span>{error}</span>
        </div>
      )}
    </div>
  );
}

function StepIcon({ state }: { state: StepState }) {
  switch (state) {
    case "done":
      return <Check className="h-4 w-4 text-green-600" />;
    case "active":
      return <Loader2 className="h-4 w-4 animate-spin text-primary" />;
    case "error":
      return <AlertTriangle className="h-4 w-4 text-destructive" />;
    default:
      return <div className="h-4 w-4 rounded-full border border-muted-foreground/30" />;
  }
}
