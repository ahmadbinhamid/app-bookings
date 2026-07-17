import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button, Input } from "@flowposltd/ui";
import { Trash2, Plus } from "lucide-react";
import { FormField } from "@/components/ui/form-field";
import { listTimeOff, createTimeOff, deleteTimeOff } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

interface EmployeeTimeOffEditorProps {
  locationId: string;
  employeeId: string;
}

export function EmployeeTimeOffEditor({ locationId, employeeId }: EmployeeTimeOffEditorProps) {
  const qc = useQueryClient();
  const key = queryKeys.timeOff(locationId, employeeId);

  const { data: timeOff = [], isLoading } = useQuery({
    queryKey: key,
    queryFn: () => listTimeOff(locationId, employeeId),
  });

  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const createMut = useMutation({
    mutationFn: () => createTimeOff(locationId, employeeId, {
      start_datetime: new Date(start).toISOString(),
      end_datetime: new Date(end).toISOString(),
      reason: reason || undefined,
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: key });
      setStart(""); setEnd(""); setReason(""); setError("");
    },
    onError: (err: Error) => setError(err.message),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteTimeOff(locationId, employeeId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
    onError: (err: Error) => toast.error(err.message),
  });

  function handleAdd() {
    if (!start || !end) {
      setError("Both a start and end are required");
      return;
    }
    createMut.mutate();
  }

  return (
    <div className="flex flex-col gap-4">
      {isLoading ? (
        <p className="text-sm text-content-secondary">Loading…</p>
      ) : timeOff.length === 0 ? (
        <p className="text-sm text-content-secondary italic">No time off scheduled.</p>
      ) : (
        <div className="flex flex-col gap-1.5">
          {timeOff.map((t) => (
            <div key={t.id} className="flex items-center justify-between rounded-sm bg-muted px-3 py-2 text-sm">
              <div>
                <span className="tabular-nums">
                  {new Date(t.start_datetime).toLocaleString()} – {new Date(t.end_datetime).toLocaleString()}
                </span>
                {t.reason && <span className="text-content-secondary ml-2">({t.reason})</span>}
              </div>
              <button
                onClick={() => deleteMut.mutate(t.id)}
                className="text-content-secondary hover:text-destructive transition-colors"
              >
                <Trash2 className="size-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-sm border border-border p-3 flex flex-col gap-3">
        <p className="text-xs font-medium text-content-secondary uppercase tracking-wide">Add time off</p>
        <div className="grid grid-cols-2 gap-2">
          <FormField label="Start">
            <Input type="datetime-local" value={start} onChange={(e) => setStart(e.target.value)} />
          </FormField>
          <FormField label="End">
            <Input type="datetime-local" value={end} onChange={(e) => setEnd(e.target.value)} />
          </FormField>
        </div>
        <FormField label="Reason" hint="Optional">
          <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. Annual leave" />
        </FormField>
        {error && <p className="text-xs text-destructive">{error}</p>}
        <Button size="sm" onClick={handleAdd} loading={createMut.isPending}>
          <Plus className="size-3.5" />
          Add time off
        </Button>
      </div>
    </div>
  );
}
