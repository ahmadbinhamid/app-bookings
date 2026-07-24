import { cn, formatTimeRange } from "@/utils";
import { type BuiltTimeline, type TimelineSegmentInput, buildTimelineAxisTicks } from "@/utils/timeline";

interface BookingTimelineProps {
  timeline: BuiltTimeline;
  /** The same segments passed to buildTimeline() — only needed for the "large" axis. */
  segments: TimelineSegmentInput[];
  variant?: "mini" | "large";
  className?: string;
  /** The booking location's IANA zone — appointment times render in it, not the viewer's own. */
  timeZone?: string;
}

// Visualizes a booking's segments on a shared time axis, with idle gaps
// rendered as dashed regions — the redesign's core "gap-aware scheduling"
// visual, reused by the bookings table, the new-booking proposal step, and
// the booking detail dialog.
export function BookingTimeline({ timeline, segments, variant = "mini", className, timeZone }: BookingTimelineProps) {
  const isLarge = variant === "large";
  const ticks = isLarge ? buildTimelineAxisTicks(segments, timeZone) : [];

  return (
    <div className={className}>
      <div className={cn("relative overflow-hidden rounded", isLarge ? "h-10" : "h-3 bg-muted")}>
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
            title={`${seg.name} · ${formatTimeRange(seg.start, seg.end, timeZone)}`}
            className={cn(
              "absolute inset-y-0 rounded flex items-center overflow-hidden",
              isLarge && "px-2"
            )}
            style={{
              left: `${seg.left}%`, width: `${seg.width}%`, backgroundColor: seg.color.solid,
              // A hairline seam between back-to-back segments — without it,
              // two adjacent services paint as one undifferentiated block.
              boxShadow: isLarge && i < timeline.segments.length - 1 ? "inset -2px 0 0 0 hsl(var(--card))" : undefined,
            }}
          >
            {isLarge && (
              <span className="text-body-3 font-medium text-white capitalize truncate">{seg.name}</span>
            )}
          </div>
        ))}
      </div>

      {isLarge && (
        <div className="relative h-4 mt-1.5">
          {ticks.map((tick, i) => {
            // Centering every label on its % mark (the default) pushes the
            // first/last labels half-off the edge, since 0% and 100% are the
            // box's own boundaries with no room to center into. Anchor those
            // two inward from the edge instead; only interior ticks center.
            const isFirst = i === 0;
            const isLast = i === ticks.length - 1;
            return (
              <span
                key={tick.key}
                className={cn(
                  "absolute text-[11px] leading-none text-content-tertiary tabular-nums whitespace-nowrap",
                  isFirst ? "left-0" : isLast ? "right-0" : "-translate-x-1/2"
                )}
                style={isFirst || isLast ? undefined : { left: `${tick.left}%` }}
              >
                {tick.label}
              </span>
            );
          })}
        </div>
      )}
    </div>
  );
}
