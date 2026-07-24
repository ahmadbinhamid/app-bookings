import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button, Input } from "@flowposltd/ui";
import { Trash2, Plus, Palmtree } from "lucide-react";
import { FormField } from "@/components/ui/form-field";
import { DatePopover } from "@/components/ui/calendar-popover";
import { TimePopover } from "@/components/ui/time-popover";
import { listTimeOff, createTimeOff, deleteTimeOff } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { combineDateAndMinutes, toISODateLocal } from "@/utils";
import { useLocationContext } from "@/contexts/location-context";

interface EmployeeTimeOffEditorProps {
  locationId: string;
  employeeId: string;
}

export function EmployeeTimeOffEditor({ locationId, employeeId }: EmployeeTimeOffEditorProps) {
  const qc = useQueryClient();
  const key = queryKeys.timeOff(locationId, employeeId);
  const { selectedLocation } = useLocationContext();
  const timeZone = selectedLocation?.timezone;

  const { data: timeOff = [], isLoading } = useQuery({
    queryKey: key,
    queryFn: () => listTimeOff(locationId, employeeId),
  });

  const today = toISODateLocal(new Date());
  const [startDate, setStartDate] = useState(today);
  const [startMinutes, setStartMinutes] = useState(0);
  const [endDate, setEndDate] = useState(today);
  const [endMinutes, setEndMinutes] = useState(24 * 60 - 15);
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const createMut = useMutation({
    mutationFn: () => createTimeOff(locationId, employeeId, {
      start_datetime: combineDateAndMinutes(startDate, startMinutes),
      end_datetime: combineDateAndMinutes(endDate, endMinutes),
      reason: reason || undefined,
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: key });
      setReason("");
      setError("");
    },
    onError: (err: Error) => setError(err.message),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteTimeOff(locationId, employeeId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
    onError: (err: Error) => toast.error(err.message),
  });

  function handleAdd() {
    setError("");
    createMut.mutate();
  }

  return (
    <div className="flex flex-col gap-4">
      {isLoading ? (
        <p className="text-body-2 text-content-secondary">Loading…</p>
      ) : timeOff.length === 0 ? (
        <p className="text-body-2 text-content-secondary italic">No time off scheduled.</p>
      ) : (
        <div className="flex flex-col gap-2.5 max-h-70 overflow-y-auto">
          {timeOff.map((t) => (
            <div key={t.id} className="flex items-center gap-3.5 rounded-md border border-border p-3.5">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-amber-100 text-amber-700">
                <Palmtree className="size-4.5" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium text-body-2 text-foreground">
                  {new Date(t.start_datetime).toLocaleString([], { dateStyle: "medium", timeStyle: "short", timeZone })}
                  {" – "}
                  {new Date(t.end_datetime).toLocaleString([], { dateStyle: "medium", timeStyle: "short", timeZone })}
                </div>
                {t.reason && <div className="text-body-3 text-content-secondary mt-0.5">{t.reason}</div>}
              </div>
              <button
                onClick={() => deleteMut.mutate(t.id)}
                className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border text-content-tertiary hover:border-destructive/30 hover:bg-destructive/5 hover:text-destructive transition-colors"
              >
                <Trash2 className="size-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-md border border-border p-4 flex flex-col gap-3.5">
        <p className="text-body-3 font-medium text-content-tertiary uppercase tracking-wide">Add time off</p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <FormField label="Start">
            <div className="flex gap-2">
              <DatePopover value={startDate} onChange={setStartDate} />
              <TimePopover value={startMinutes} onChange={setStartMinutes} />
            </div>
          </FormField>
          <FormField label="End">
            <div className="flex gap-2">
              <DatePopover value={endDate} onChange={setEndDate} />
              <TimePopover value={endMinutes} onChange={setEndMinutes} />
            </div>
          </FormField>
        </div>
        <FormField label="Reason" hint="Optional">
          <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. Annual leave" />
        </FormField>
        {error && <p className="text-body-3 text-destructive">{error}</p>}
        <Button size="sm" onClick={handleAdd} loading={createMut.isPending} className="self-start">
          <Plus className="size-3.5" />
          Add time off
        </Button>
      </div>
    </div>
  );
}
