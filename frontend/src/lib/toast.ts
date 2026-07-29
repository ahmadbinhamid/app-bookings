import { toast as uiToast } from "@flowposltd/ui";

// Thin adapter over @flowposltd/ui's toast({ description, variant }) API so
// call sites keep the terser toast.success(...)/toast.error(...) shape —
// mirrors quotes' frontend/src/lib/toast.ts exactly, per "mirror the sibling
// apps' conventions."
export const toast = {
  success: (message: string) => uiToast({ description: message, variant: "success" }),
  error: (message: string) => uiToast({ description: message, variant: "destructive" }),
};
