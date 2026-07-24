import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog, DialogContent, DialogHeader, DialogFooter,
  DialogTitle, DialogDescription, Button, Input, Textarea,
} from "@flowposltd/ui";
import { DialogBody } from "@/components/ui/dialog-body";
import { FormField } from "@/components/ui/form-field";
import { type Service, type ServiceInput } from "@/types";
import { createService, updateService } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/utils";

interface ServiceFormProps {
  open: boolean;
  onClose: () => void;
  locationId: string;
  service?: Service;
  page: number;
  limit: number;
  search: string;
}

const EMPTY: ServiceInput = {
  name: "",
  description: "",
  duration_minutes: 30,
  buffer_minutes: 0,
  price: 0,
};

const DURATION_PRESETS = [15, 30, 45, 60];

export function ServiceForm({ open, onClose, locationId, service, page, limit, search }: ServiceFormProps) {
  const qc = useQueryClient();
  const isEdit = Boolean(service);

  const [form, setForm] = useState<ServiceInput>(EMPTY);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (open) {
      setErrors({});
      if (service) {
        setForm({
          name: service.name,
          description: service.description ?? "",
          duration_minutes: service.duration_minutes,
          buffer_minutes: service.buffer_minutes,
          price: service.price,
          active: service.active,
        });
      } else {
        setForm(EMPTY);
      }
    }
  }, [open, service]);

  const invalidate = () =>
    qc.invalidateQueries({ queryKey: queryKeys.services(locationId, page, limit, search || undefined) });

  const createMut = useMutation({
    mutationFn: (input: ServiceInput) => createService(locationId, input),
    onSuccess: () => {
      toast.success("Service created");
      invalidate();
      onClose();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const updateMut = useMutation({
    mutationFn: (input: ServiceInput) => updateService(locationId, service!.id, input),
    onSuccess: () => {
      toast.success("Service updated");
      invalidate();
      onClose();
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const loading = createMut.isPending || updateMut.isPending;

  function validate(): boolean {
    const errs: Record<string, string> = {};
    if (!form.name.trim()) errs.name = "Name is required";
    if (!form.duration_minutes || form.duration_minutes < 1) errs.duration_minutes = "Duration must be at least 1 minute";
    if (form.buffer_minutes < 0) errs.buffer_minutes = "Buffer can't be negative";
    setErrors(errs);
    return Object.keys(errs).length === 0;
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!validate()) return;
    if (isEdit) updateMut.mutate(form);
    else createMut.mutate(form);
  }

  function set<K extends keyof ServiceInput>(key: K, value: ServiceInput[K]) {
    setForm((f) => ({ ...f, [key]: value }));
    setErrors((e) => ({ ...e, [key]: "" }));
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="w-[calc(100%-2rem)] max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit service" : "New service"}</DialogTitle>
          <DialogDescription>
            {isEdit ? "Update the service details." : "Add a service customers can book at this location."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <DialogBody className="flex flex-col gap-4">
            <FormField label="Service name" required>
              <Input
                value={form.name}
                onChange={(e) => set("name", e.target.value)}
                placeholder="e.g. Haircut & style"
                error={errors.name}
              />
            </FormField>

            <FormField label="Duration" hint={errors.duration_minutes}>
              <div className="flex gap-1.5">
                {DURATION_PRESETS.map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => set("duration_minutes", m)}
                    className={cn(
                      "flex-1 h-11 rounded-md text-body-3 font-medium border-2 transition-colors",
                      form.duration_minutes === m
                        ? "border-primary bg-primary/5 text-primary"
                        : "border-border text-content-secondary hover:bg-muted"
                    )}
                  >
                    {m} min
                  </button>
                ))}
                <input
                  type="number"
                  min={1}
                  value={form.duration_minutes || ""}
                  onChange={(e) => set("duration_minutes", parseInt(e.target.value, 10) || 0)}
                  placeholder="Custom"
                  className={cn(
                    "w-20 shrink-0 h-11 rounded-md text-body-3 font-medium border-2 text-center tabular-nums transition-colors",
                    "focus-visible:outline-none focus:border-primary focus:bg-primary/5 focus:text-primary",
                    "placeholder:text-content-tertiary placeholder:font-normal",
                    !DURATION_PRESETS.includes(form.duration_minutes)
                      ? "border-primary bg-primary/5 text-primary"
                      : "border-border text-content-secondary hover:bg-muted"
                  )}
                />
              </div>
            </FormField>

            <div className="flex gap-3.5">
              <div className="flex-1">
                <FormField label="Price" required>
                  <div className="relative">
                    <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-body-2 font-medium text-content-tertiary pointer-events-none">
                      $
                    </span>
                    <Input
                      type="number"
                      min={0}
                      step="0.01"
                      value={form.price ?? ""}
                      onChange={(e) => set("price", parseFloat(e.target.value) || 0)}
                      placeholder="0.00"
                      className="pl-7 tabular-nums"
                    />
                  </div>
                </FormField>
              </div>
              <div className="flex-1">
                <FormField label="Buffer (min)" hint="Employee-only, never charged">
                  <Input
                    type="number"
                    min={0}
                    value={form.buffer_minutes ?? ""}
                    onChange={(e) => set("buffer_minutes", parseInt(e.target.value, 10) || 0)}
                    error={errors.buffer_minutes}
                  />
                </FormField>
              </div>
            </div>

            <FormField label="Description" hint="Optional">
              <Textarea
                value={form.description ?? ""}
                onChange={(e) => set("description", e.target.value)}
                placeholder="What's included, aftercare notes, anything the customer should know…"
                rows={3}
              />
            </FormField>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="secondary" onClick={onClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" loading={loading}>
              {isEdit ? "Save changes" : "Create service"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
