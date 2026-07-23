import { serviceVisual, type ServiceVisual } from "@/constants/service-visuals";
import { formatTime } from "@/utils/format";

export interface TimelineSegmentInput {
  key: string;
  name: string;
  start: string; // ISO datetime
  end: string; // ISO datetime
}

export interface BuiltTimelineSegment extends TimelineSegmentInput {
  left: number; // percent, 0-100
  width: number; // percent, 0-100
  color: ServiceVisual;
}

export interface BuiltTimelineGap {
  left: number;
  width: number;
  minutes: number;
}

export interface BuiltTimeline {
  segments: BuiltTimelineSegment[];
  gaps: BuiltTimelineGap[];
  totalGapMinutes: number;
  startLabel: string;
  endLabel: string;
}

// Lays out a booking's segments on a shared 0-100% axis spanning from the
// earliest start to the latest end, and surfaces idle gaps between
// consecutive (time-sorted) segments — the visual heart of the "gap-aware
// scheduling" pitch. Pure percentage geometry so it works at any container
// width; callers render it with absolutely-positioned blocks.
export function buildTimeline(items: TimelineSegmentInput[]): BuiltTimeline | null {
  if (items.length === 0) return null;

  const sorted = [...items].sort(
    (a, b) => new Date(a.start).getTime() - new Date(b.start).getTime()
  );
  const windowStart = Math.min(...sorted.map((s) => new Date(s.start).getTime()));
  const windowEnd = Math.max(...sorted.map((s) => new Date(s.end).getTime()));
  const span = Math.max(windowEnd - windowStart, 1);

  const segments: BuiltTimelineSegment[] = sorted.map((item) => {
    const start = new Date(item.start).getTime();
    const end = new Date(item.end).getTime();
    return {
      ...item,
      left: ((start - windowStart) / span) * 100,
      width: ((end - start) / span) * 100,
      color: serviceVisual(item.name),
    };
  });

  const gaps: BuiltTimelineGap[] = [];
  for (let i = 0; i < sorted.length - 1; i++) {
    const prevEnd = new Date(sorted[i].end).getTime();
    const nextStart = new Date(sorted[i + 1].start).getTime();
    if (nextStart > prevEnd) {
      gaps.push({
        left: ((prevEnd - windowStart) / span) * 100,
        width: ((nextStart - prevEnd) / span) * 100,
        minutes: Math.round((nextStart - prevEnd) / 60000),
      });
    }
  }

  return {
    segments,
    gaps,
    totalGapMinutes: gaps.reduce((sum, g) => sum + g.minutes, 0),
    startLabel: formatTime(sorted[0].start),
    endLabel: formatTime(sorted[sorted.length - 1].end),
  };
}

export interface TimelineAxisTick {
  key: string;
  left: number;
  label: string;
}

// 30-minute tick marks under the "large" timeline variant, spanning the same
// window buildTimeline used.
export function buildTimelineAxisTicks(items: TimelineSegmentInput[]): TimelineAxisTick[] {
  if (items.length === 0) return [];
  const windowStart = Math.min(...items.map((s) => new Date(s.start).getTime()));
  const windowEnd = Math.max(...items.map((s) => new Date(s.end).getTime()));
  const span = Math.max(windowEnd - windowStart, 1);

  const ticks: TimelineAxisTick[] = [];
  const stepMs = 30 * 60 * 1000;
  const firstTick = Math.ceil(windowStart / stepMs) * stepMs;
  for (let t = firstTick; t <= windowEnd; t += stepMs) {
    ticks.push({
      key: String(t),
      left: ((t - windowStart) / span) * 100,
      label: formatTime(new Date(t).toISOString()),
    });
  }
  return ticks;
}
