import { X } from "lucide-react";
import { cn } from "@/utils";
import { type Schedule } from "@/types";

const DAYS_SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const GRID_HEIGHT = 220;

function parseTime(t: string): number {
  const [h, m] = t.split(":").map(Number);
  return h * 60 + m;
}

function formatHourLabel(hourOfDay: number): string {
  const h = hourOfDay % 24;
  const period = h < 12 ? "AM" : "PM";
  const hour12 = h % 12 === 0 ? 12 : h % 12;
  return `${hour12} ${period}`;
}

function formatCompactTime(t: string): string {
  const minutes = parseTime(t);
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  const period = h < 12 ? "AM" : "PM";
  const hour12 = h % 12 === 0 ? 12 : h % 12;
  return m === 0 ? `${hour12}${period}` : `${hour12}:${String(m).padStart(2, "0")}${period}`;
}

interface WeeklyHoursGridProps {
  schedules: Schedule[];
  className?: string;
  /** When set, each shift block gets a hover-revealed remove button — this is
   *  the grid's own single source of truth for shifts, so removal lives here
   *  instead of a separate duplicate list. */
  onRemove?: (scheduleId: string) => void;
}

// The 7-day weekly-hours grid — absolutely-positioned shift blocks over an
// hour-axis background, replacing the old per-day text list. Multiple rows
// on the same day (split shifts) simply render as separate blocks in that
// day's column, since the underlying data already supports it.
export function WeeklyHoursGrid({ schedules, className, onRemove }: WeeklyHoursGridProps) {
  const byDay: Schedule[][] = Array.from({ length: 7 }, (_, day) =>
    schedules
      .filter((s) => s.day_of_week === day)
      .sort((a, b) => parseTime(a.start_time) - parseTime(b.start_time))
  );

  const allMinutes = schedules.flatMap((s) => [parseTime(s.start_time), parseTime(s.end_time)]);
  const rawMin = allMinutes.length ? Math.min(...allMinutes) : 8 * 60;
  const rawMax = allMinutes.length ? Math.max(...allMinutes) : 18 * 60;
  const gridStart = Math.max(0, Math.floor(rawMin / 60) * 60 - 60);
  const gridEnd = Math.min(24 * 60, Math.ceil(rawMax / 60) * 60 + 60);
  const span = Math.max(gridEnd - gridStart, 60);

  const hourMarks: { minutes: number; label: string }[] = [];
  for (let h = Math.ceil(gridStart / 60); h <= Math.floor(gridEnd / 60); h += 2) {
    hourMarks.push({ minutes: h * 60, label: formatHourLabel(h) });
  }

  function topFor(minutes: number): number {
    return ((minutes - gridStart) / span) * GRID_HEIGHT;
  }

  return (
    <div className={cn("rounded-md border border-border overflow-x-auto", className)}>
      <div className="min-w-140">
      <div className="grid" style={{ gridTemplateColumns: "56px repeat(7, 1fr)" }}>
        <div className="border-b border-border" />
        {DAYS_SHORT.map((label, day) => {
          const on = byDay[day].length > 0;
          const split = byDay[day].length > 1;
          return (
            <div
              key={day}
              className={cn(
                "flex flex-col items-center gap-0.5 py-2 border-b border-l border-border",
                !on && "bg-muted/50"
              )}
            >
              <span className="text-body-3 font-medium text-foreground">{label}</span>
              <span
                className={cn(
                  "text-body-3 font-medium",
                  on ? "text-primary" : "text-content-tertiary"
                )}
              >
                {on ? (split ? "Split" : "On") : "Off"}
              </span>
            </div>
          );
        })}
      </div>

      <div className="relative grid" style={{ gridTemplateColumns: "56px repeat(7, 1fr)", height: GRID_HEIGHT }}>
        <div className="relative">
          {hourMarks.map((h) => (
            <div
              key={h.minutes}
              className="absolute right-1.5 -translate-y-1/2 whitespace-nowrap text-body-3 text-content-tertiary tabular-nums"
              style={{ top: topFor(h.minutes) }}
            >
              {h.label}
            </div>
          ))}
        </div>
        {byDay.map((shifts, day) => (
          <div key={day} className="relative border-l border-border">
            {hourMarks.map((h) => (
              <div
                key={h.minutes}
                className="absolute inset-x-0 h-px bg-border"
                style={{ top: topFor(h.minutes) }}
              />
            ))}
            {shifts.map((s) => {
              const top = topFor(parseTime(s.start_time));
              const bottom = topFor(parseTime(s.end_time));
              return (
                <div
                  key={s.id}
                  className="group/shift absolute left-1 right-1 rounded-md border border-primary bg-primary/10 px-1.5 py-1 overflow-hidden"
                  style={{ top, height: Math.max(bottom - top, 6) }}
                >
                  <span className="text-body-3 font-semibold text-primary leading-tight">
                    {formatCompactTime(s.start_time)}–{formatCompactTime(s.end_time)}
                  </span>
                  {onRemove && (
                    <button
                      type="button"
                      title="Remove shift"
                      onClick={() => onRemove(s.id)}
                      className="absolute top-0.5 right-0.5 flex size-4 items-center justify-center rounded-sm bg-card/90 text-primary opacity-0 transition-opacity hover:bg-card group-hover/shift:opacity-100"
                    >
                      <X className="size-2.5" />
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        ))}
      </div>
      </div>
    </div>
  );
}
