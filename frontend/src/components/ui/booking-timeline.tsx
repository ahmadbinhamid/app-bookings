import { cn, formatTimeRange } from "@/utils";
import { type BuiltTimeline, type TimelineSegmentInput, buildTimelineAxisTicks } from "@/utils/timeline";

interface BookingTimelineProps {
  timeline: BuiltTimeline;
  /** The same segments passed to buildTimeline() — only needed for the "large" axis. */
  segments: TimelineSegmentInput[];
  variant?: "mini" | "large";
  className?: string;
}

// Visualizes a booking's segments on a shared time axis, with idle gaps
// rendered as dashed regions — the redesign's core "gap-aware scheduling"
// visual, reused by the bookings table, the new-booking proposal step, and
// the booking detail dialog.
export function BookingTimeline({ timeline, segments, variant = "mini", className }: BookingTimelineProps) {
  const isLarge = variant === "large";
  const ticks = isLarge ? buildTimelineAxisTicks(segments) : [];

  return (
    <div className={className}>
      <div className={cn("relative overflow-hidden rounded", isLarge ? "h-14" : "h-3 bg-muted")}>
        {timeline.gaps.map((gap, i) => (
          <div
            key={i}
            className={cn(
              "absolute inset-y-0 rounded border border-dashed border-border",
              // The diagonal texture reads fine at the large size (wizard/detail
              // timelines) but is just noise at the table's compact mini size —
              // a plain muted fill stays legible there.
              isLarge
                ? "bg-[repeating-linear-gradient(45deg,transparent,transparent_5px,hsl(var(--border))_5px,hsl(var(--border))_7px)]"
                : "bg-muted"
            )}
            style={{ left: `${gap.left}%`, width: `${gap.width}%` }}
          >
            {isLarge && gap.minutes > 0 && (
              <span className="absolute inset-0 grid place-items-center text-body-3 font-medium text-amber-700 whitespace-nowrap px-1 overflow-hidden">
                {gap.minutes} min idle
              </span>
            )}
          </div>
        ))}
        {timeline.segments.map((seg, i) => (
          <div
            key={seg.key ?? i}
            title={`${seg.name} · ${formatTimeRange(seg.start, seg.end)}`}
            className={cn(
              "absolute inset-y-0 rounded flex items-center overflow-hidden",
              isLarge && "px-2.5"
            )}
            style={{ left: `${seg.left}%`, width: `${seg.width}%`, backgroundColor: seg.color.solid }}
          >
            {isLarge && (
              <span className="text-body-3 font-medium text-white capitalize truncate">{seg.name}</span>
            )}
          </div>
        ))}
      </div>

      {isLarge && (
        <div className="relative h-4 mt-1.5">
          {ticks.map((tick) => (
            <span
              key={tick.key}
              className="absolute -translate-x-1/2 text-body-3 text-content-tertiary tabular-nums whitespace-nowrap"
              style={{ left: `${tick.left}%` }}
            >
              {tick.label}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
