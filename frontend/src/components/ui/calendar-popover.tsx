import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Calendar as CalendarIcon, ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/utils";
import { usePopoverAnchor } from "@/hooks/use-popover-anchor";

const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];
const DAYS_SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function toISODate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function parseISODate(iso: string): Date {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
}

function formatDisplayDate(iso: string): string {
  return parseISODate(iso).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

interface DayCell {
  date: Date;
  iso: string;
  inMonth: boolean;
}

function buildMonthGrid(year: number, month: number): DayCell[] {
  const first = new Date(year, month, 1);
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const cells: DayCell[] = [];

  for (let i = first.getDay(); i > 0; i--) {
    const d = new Date(year, month, 1 - i);
    cells.push({ date: d, iso: toISODate(d), inMonth: false });
  }
  for (let day = 1; day <= daysInMonth; day++) {
    const d = new Date(year, month, day);
    cells.push({ date: d, iso: toISODate(d), inMonth: true });
  }
  while (cells.length % 7 !== 0) {
    const prev = cells[cells.length - 1].date;
    const d = new Date(prev.getFullYear(), prev.getMonth(), prev.getDate() + 1);
    cells.push({ date: d, iso: toISODate(d), inMonth: false });
  }
  return cells;
}

const TRIGGER_CLASS = cn(
  "flex w-full items-center gap-2 h-10 rounded px-[15px] whitespace-nowrap",
  "border border-field-border bg-field-bg text-body-2 font-medium text-field-text",
  "hover:bg-field-hover transition-colors duration-100 cursor-pointer"
);

const POPOVER_CLASS = cn(
  "rounded border border-field-border bg-popover shadow-md p-section",
  "animate-in fade-in-0 zoom-in-95"
);

function MonthNav({ label, onPrev, onNext }: { label: string; onPrev: () => void; onNext: () => void }) {
  return (
    <div className="flex items-center justify-between mb-3">
      <button
        type="button"
        onClick={onPrev}
        className="size-7 grid place-items-center rounded hover:bg-muted text-content-secondary transition-colors"
      >
        <ChevronLeft className="size-4" />
      </button>
      <span className="text-body-2 font-medium text-foreground">{label}</span>
      <button
        type="button"
        onClick={onNext}
        className="size-7 grid place-items-center rounded hover:bg-muted text-content-secondary transition-colors"
      >
        <ChevronRight className="size-4" />
      </button>
    </div>
  );
}

function WeekdayRow() {
  return (
    <div className="grid grid-cols-7 gap-0.5 mb-1">
      {DAYS_SHORT.map((d) => (
        <div key={d} className="text-center text-body-3 font-medium text-content-tertiary py-1">
          {d[0]}
        </div>
      ))}
    </div>
  );
}

interface DatePopoverProps {
  value: string | null; // ISO yyyy-mm-dd
  onChange: (iso: string) => void;
  placeholder?: string;
  className?: string;
  /** Dates before this (ISO yyyy-mm-dd) are shown grayed out and unpickable. */
  minISO?: string;
}

/** Single-date field — day-grid popover, no native date input. */
export function DatePopover({ value, onChange, placeholder = "Select date", className, minISO }: DatePopoverProps) {
  const [open, setOpen] = useState(false);
  const [cursor, setCursor] = useState(() => parseISODate(value ?? toISODate(new Date())));
  const { triggerRef, popoverRef, pos, updatePos } = usePopoverAnchor<HTMLButtonElement, HTMLDivElement>(
    open,
    () => setOpen(false),
    280,
    380
  );

  useEffect(() => {
    if (open && value) setCursor(parseISODate(value));
  }, [open, value]);

  const cells = buildMonthGrid(cursor.getFullYear(), cursor.getMonth());

  function shiftMonth(delta: number) {
    setCursor((c) => new Date(c.getFullYear(), c.getMonth() + delta, 1));
  }

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => {
          updatePos();
          setOpen((v) => !v);
        }}
        className={cn(TRIGGER_CLASS, open && "border-field-border-active")}
      >
        <CalendarIcon className="size-4 text-primary shrink-0" />
        <span className={cn("truncate", !value && "text-field-text-placeholder")}>
          {value ? formatDisplayDate(value) : placeholder}
        </span>
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={popoverRef}
            style={{
              position: "fixed", top: pos.top, bottom: pos.bottom, left: pos.left, width: pos.width, zIndex: 9999,
              // See time-popover.tsx: Radix's modal Dialog disables pointer events
              // on body and only re-enables them on its own content node, so this
              // body-portaled popover would otherwise be inert while a Dialog is open.
              pointerEvents: "auto",
            }}
            className={POPOVER_CLASS}
          >
            <MonthNav
              label={`${MONTHS[cursor.getMonth()]} ${cursor.getFullYear()}`}
              onPrev={() => shiftMonth(-1)}
              onNext={() => shiftMonth(1)}
            />
            <WeekdayRow />
            <div className="grid grid-cols-7 gap-0.5">
              {cells.map((c) => {
                const tooEarly = Boolean(minISO && c.iso < minISO);
                return (
                  <button
                    key={c.iso}
                    type="button"
                    disabled={!c.inMonth || tooEarly}
                    onClick={() => {
                      onChange(c.iso);
                      setOpen(false);
                    }}
                    className={cn(
                      "size-9 rounded text-body-3 font-medium grid place-items-center transition-colors",
                      !c.inMonth && "invisible",
                      c.inMonth && tooEarly && "text-content-tertiary/40 cursor-not-allowed hover:bg-transparent",
                      c.inMonth && !tooEarly && c.iso === value
                        ? "bg-primary text-primary-foreground font-medium"
                        : c.inMonth && !tooEarly && "text-foreground hover:bg-muted"
                    )}
                  >
                    {c.date.getDate()}
                  </button>
                );
              })}
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}

export interface SimpleDateRange {
  from: string | null;
  to: string | null;
}

interface DateRangePopoverProps {
  value: SimpleDateRange;
  onChange: (range: SimpleDateRange) => void;
  placeholder?: string;
  className?: string;
}

/** Date-range field — two-click start→end day-grid popover with a Clear action. */
export function DateRangePopover({ value, onChange, placeholder = "All dates", className }: DateRangePopoverProps) {
  const [open, setOpen] = useState(false);
  const [cursor, setCursor] = useState(() => parseISODate(value.from ?? toISODate(new Date())));
  const { triggerRef, popoverRef, pos, updatePos } = usePopoverAnchor<HTMLButtonElement, HTMLDivElement>(
    open,
    () => setOpen(false),
    300,
    420
  );

  const cells = buildMonthGrid(cursor.getFullYear(), cursor.getMonth());

  function shiftMonth(delta: number) {
    setCursor((c) => new Date(c.getFullYear(), c.getMonth() + delta, 1));
  }

  function pick(iso: string) {
    const { from, to } = value;
    if (!from || (from && to)) {
      onChange({ from: iso, to: null });
    } else if (iso < from) {
      onChange({ from: iso, to: null });
    } else {
      onChange({ from, to: iso });
      setOpen(false);
    }
  }

  function cellClass(c: DayCell) {
    if (!c.inMonth) return "invisible";
    const { from, to } = value;
    if (c.iso === from || c.iso === to) return "bg-primary text-primary-foreground font-medium rounded";
    if (from && to && c.iso > from && c.iso < to) return "bg-primary/10 text-primary rounded-none";
    return "text-foreground hover:bg-muted rounded";
  }

  const label = value.from
    ? value.to
      ? `${formatDisplayDate(value.from)}  –  ${formatDisplayDate(value.to)}`
      : `${formatDisplayDate(value.from)}  –  …`
    : null;

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => {
          updatePos();
          setOpen((v) => !v);
        }}
        className={cn(TRIGGER_CLASS, open && "border-field-border-active")}
      >
        <CalendarIcon className="size-4 text-primary shrink-0" />
        <span className={cn("truncate", !label && "text-field-text-placeholder")}>{label ?? placeholder}</span>
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={popoverRef}
            style={{
              position: "fixed", top: pos.top, bottom: pos.bottom, left: pos.left, width: pos.width, zIndex: 9999,
              // See time-popover.tsx: Radix's modal Dialog disables pointer events
              // on body and only re-enables them on its own content node, so this
              // body-portaled popover would otherwise be inert while a Dialog is open.
              pointerEvents: "auto",
            }}
            className={POPOVER_CLASS}
          >
            <MonthNav
              label={`${MONTHS[cursor.getMonth()]} ${cursor.getFullYear()}`}
              onPrev={() => shiftMonth(-1)}
              onNext={() => shiftMonth(1)}
            />
            <WeekdayRow />
            <div className="grid grid-cols-7 gap-0.5">
              {cells.map((c) => (
                <button
                  key={c.iso}
                  type="button"
                  disabled={!c.inMonth}
                  onClick={() => pick(c.iso)}
                  className={cn("size-9 text-body-3 font-medium grid place-items-center transition-colors", cellClass(c))}
                >
                  {c.date.getDate()}
                </button>
              ))}
            </div>
            {value.from && (
              <div className="flex justify-end mt-3 pt-3 border-t border-border">
                <button
                  type="button"
                  onClick={() => {
                    onChange({ from: null, to: null });
                    setOpen(false);
                  }}
                  className="h-8 px-3 rounded text-body-3 font-medium text-content-secondary hover:bg-muted transition-colors"
                >
                  Clear
                </button>
              </div>
            )}
          </div>,
          document.body
        )}
    </div>
  );
}
