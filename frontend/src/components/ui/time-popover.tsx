import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Clock } from "lucide-react";
import { cn } from "@/utils";
import { usePopoverAnchor } from "@/hooks/use-popover-anchor";

export function formatMinutesOfDay(totalMinutes: number): string {
  const h = Math.floor(totalMinutes / 60);
  const m = totalMinutes % 60;
  const period = h < 12 ? "AM" : "PM";
  const hour12 = h % 12 === 0 ? 12 : h % 12;
  return `${hour12}:${String(m).padStart(2, "0")} ${period}`;
}

interface TimePopoverProps {
  /** Minutes since midnight, e.g. 540 = 9:00 AM. */
  value: number;
  onChange: (minutes: number) => void;
  stepMinutes?: number;
  className?: string;
}

/** Scrollable time-of-day list popover — no native time input. */
export function TimePopover({ value, onChange, stepMinutes = 15, className }: TimePopoverProps) {
  const [open, setOpen] = useState(false);
  const { triggerRef, popoverRef, pos, updatePos } = usePopoverAnchor<HTMLButtonElement, HTMLDivElement>(
    open,
    () => setOpen(false),
    150,
    250
  );
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const frame = requestAnimationFrame(() => {
      listRef.current?.querySelector<HTMLElement>(`[data-minutes="${value}"]`)?.scrollIntoView({ block: "center" });
    });
    return () => cancelAnimationFrame(frame);
  }, [open, value]);

  const times: number[] = [];
  for (let t = 0; t < 24 * 60; t += stepMinutes) times.push(t);

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => {
          updatePos();
          setOpen((v) => !v);
        }}
        className={cn(
          "flex w-full items-center gap-2 h-10 rounded px-[15px] whitespace-nowrap",
          "border border-field-border bg-field-bg text-body-2 font-medium text-field-text",
          "hover:bg-field-hover transition-colors duration-100 cursor-pointer",
          open && "border-field-border-active"
        )}
      >
        <Clock className="size-4 text-primary shrink-0" />
        <span>{formatMinutesOfDay(value)}</span>
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={popoverRef}
            style={{ position: "fixed", top: pos.top, bottom: pos.bottom, left: pos.left, width: pos.width, zIndex: 9999 }}
            className="rounded border border-field-border bg-popover shadow-md p-1.5 max-h-60 overflow-y-auto animate-in fade-in-0 zoom-in-95"
          >
            <div ref={listRef}>
              {times.map((t) => (
                <button
                  key={t}
                  type="button"
                  data-minutes={t}
                  onClick={() => {
                    onChange(t);
                    setOpen(false);
                  }}
                  className={cn(
                    "block w-full text-left px-3 py-2 rounded text-body-3 font-medium tabular-nums transition-colors",
                    t === value ? "bg-primary/10 text-primary font-medium" : "text-foreground hover:bg-muted"
                  )}
                >
                  {formatMinutesOfDay(t)}
                </button>
              ))}
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}
