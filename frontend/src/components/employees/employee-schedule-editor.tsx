import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button, Input, Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@flowposltd/ui";
import { Trash2, Plus } from "lucide-react";
import { FormField } from "@/components/ui/form-field";
import { listSchedules, createSchedule, deleteSchedule } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

interface EmployeeScheduleEditorProps {
  locationId: string;
  employeeId: string;
}

// Weekly recurring working hours (design doc EMPLOYEE_SCHEDULE). Multiple
// rows on the same day are valid on purpose — split shifts — the backend
// only rejects an actually-overlapping range, not a second row per day.
export function EmployeeScheduleEditor({ locationId, employeeId }: EmployeeScheduleEditorProps) {
  const qc = useQueryClient();
  const key = queryKeys.schedules(locationId, employeeId);

  const { data: schedules = [], isLoading } = useQuery({
    queryKey: key,
    queryFn: () => listSchedules(locationId, employeeId),
  });

  const [dayOfWeek, setDayOfWeek] = useState("1");
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("17:00");
  const [error, setError] = useState("");

  const createMut = useMutation({
    mutationFn: () => createSchedule(locationId, employeeId, {
      day_of_week: Number(dayOfWeek), start_time: startTime, end_time: endTime,
    }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: key });
      setError("");
    },
    onError: (err: Error) => setError(err.message),
  });

  const deleteMut = useMutation({
    mutationFn: (scheduleId: string) => deleteSchedule(locationId, employeeId, scheduleId),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
    onError: (err: Error) => toast.error(err.message),
  });

  const byDay = DAYS.map((_, day) => schedules.filter((s) => s.day_of_week === day));

  return (
    <div className="flex flex-col gap-4">
      {isLoading ? (
        <p className="text-sm text-content-secondary">Loading…</p>
      ) : (
        <div className="flex flex-col gap-2">
          {DAYS.map((label, day) => (
            <div key={day} className="flex items-start gap-3 text-sm">
              <span className="w-24 shrink-0 font-medium text-foreground pt-1">{label}</span>
              <div className="flex flex-col gap-1 flex-1">
                {byDay[day].length === 0 && <span className="text-content-secondary italic">Off</span>}
                {byDay[day].map((s) => (
                  <div key={s.id} className="flex items-center gap-2 rounded-sm bg-muted px-2 py-1 w-fit">
                    <span className="tabular-nums">{s.start_time.slice(0, 5)}–{s.end_time.slice(0, 5)}</span>
                    <button
                      onClick={() => deleteMut.mutate(s.id)}
                      className="text-content-secondary hover:text-destructive transition-colors"
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-sm border border-border p-3 flex flex-col gap-3">
        <p className="text-xs font-medium text-content-secondary uppercase tracking-wide">Add a shift</p>
        <div className="grid grid-cols-3 gap-2">
          <FormField label="Day">
            <Select value={dayOfWeek} onValueChange={setDayOfWeek}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                {DAYS.map((label, day) => (
                  <SelectItem key={day} value={String(day)}>{label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Start">
            <Input type="time" value={startTime} onChange={(e) => setStartTime(e.target.value)} />
          </FormField>
          <FormField label="End">
            <Input type="time" value={endTime} onChange={(e) => setEndTime(e.target.value)} />
          </FormField>
        </div>
        {error && <p className="text-xs text-destructive">{error}</p>}
        <Button size="sm" onClick={() => createMut.mutate()} loading={createMut.isPending}>
          <Plus className="size-3.5" />
          Add shift
        </Button>
      </div>
    </div>
  );
}
