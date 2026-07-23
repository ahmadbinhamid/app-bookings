import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button, ToggleGroup, ToggleGroupItem } from "@flowposltd/ui";
import { Plus } from "lucide-react";
import { TimePopover, formatMinutesOfDay } from "@/components/ui/time-popover";
import { WeeklyHoursGrid } from "@/components/ui/weekly-hours-grid";
import { listSchedules, createSchedule, deleteSchedule } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/utils";

const DAY_LABELS = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];
const ALL_DAYS = ["0", "1", "2", "3", "4", "5", "6"];
const WEEKDAYS = ["1", "2", "3", "4", "5"];

function toTimeString(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

interface EmployeeScheduleEditorProps {
  locationId: string;
  employeeId: string;
}

// Weekly recurring working hours (design doc EMPLOYEE_SCHEDULE). Multiple
// rows on the same day are valid on purpose — split shifts — the backend
// only rejects an actually-overlapping range, not a second row per day; the
// grid renders each as its own block in that day's column, and is the single
// source of truth for existing shifts (remove lives on the block itself, on
// hover, rather than a second duplicate list of the same data).
export function EmployeeScheduleEditor({ locationId, employeeId }: EmployeeScheduleEditorProps) {
  const qc = useQueryClient();
  const key = queryKeys.schedules(locationId, employeeId);

  const { data: schedules = [], isLoading } = useQuery({
    queryKey: key,
    queryFn: () => listSchedules(locationId, employeeId),
  });

  const [days, setDays] = useState<string[]>(["1"]);
  const [startMinutes, setStartMinutes] = useState(9 * 60);
  const [endMinutes, setEndMinutes] = useState(17 * 60);
  const [error, setError] = useState("");

  const createMut = useMutation({
    mutationFn: () =>
      Promise.all(
        days.map((day) =>
          createSchedule(locationId, employeeId, {
            day_of_week: Number(day), start_time: toTimeString(startMinutes), end_time: toTimeString(endMinutes),
          })
        )
      ),
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

  return (
    <div className="flex flex-col gap-5">
      {isLoading ? (
        <p className="text-body-2 text-content-secondary">Loading…</p>
      ) : (
        <WeeklyHoursGrid schedules={schedules} onRemove={(id) => deleteMut.mutate(id)} />
      )}

      <div className="rounded-md border border-border p-4 flex flex-col gap-3.5">
        <p className="text-body-3 font-medium text-content-tertiary uppercase tracking-wide">Add a shift</p>

        <div>
          <div className="flex items-center justify-between mb-1.5">
            <span className="text-body-3 font-medium text-content-secondary">Days</span>
            <div className="flex items-center gap-1.5 text-body-3">
              <button type="button" onClick={() => setDays(ALL_DAYS)} className="font-medium text-primary hover:underline">
                Every day
              </button>
              <span className="text-content-tertiary">·</span>
              <button type="button" onClick={() => setDays(WEEKDAYS)} className="font-medium text-primary hover:underline">
                Weekdays
              </button>
            </div>
          </div>
          <ToggleGroup
            type="multiple"
            value={days}
            onValueChange={(v) => v.length > 0 && setDays(v)}
            className="justify-start gap-1 flex-wrap"
          >
            {DAY_LABELS.map((label, day) => (
              <ToggleGroupItem
                key={day}
                value={String(day)}
                className={cn(
                  "rounded-md px-2.5 py-1.5 text-body-3 font-medium text-content-secondary",
                  "data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
                )}
              >
                {label}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:flex sm:items-end">
          <div className="flex-1">
            <span className="text-body-3 font-medium text-content-secondary block mb-1.5">Start</span>
            <TimePopover value={startMinutes} onChange={setStartMinutes} />
          </div>
          <div className="flex-1">
            <span className="text-body-3 font-medium text-content-secondary block mb-1.5">End</span>
            <TimePopover value={endMinutes} onChange={setEndMinutes} />
          </div>
          <Button onClick={() => createMut.mutate()} loading={createMut.isPending} className="col-span-2 sm:col-span-1">
            <Plus className="size-3.5" />
            Add
          </Button>
        </div>

        {error && <p className="text-body-3 text-destructive">{error}</p>}
        {endMinutes <= startMinutes && (
          <p className="text-body-3 text-content-tertiary">
            End ({formatMinutesOfDay(endMinutes)}) should be after start ({formatMinutesOfDay(startMinutes)}).
          </p>
        )}
      </div>
    </div>
  );
}
