import { useCallback, useEffect, useRef, useState } from "react";

export interface AnchorPosition {
  left: number;
  width: number;
  /** Exactly one of top/bottom is set — whichever side has room. */
  top?: number;
  bottom?: number;
}

// Shared fixed-position + outside-click wiring for the app's custom
// popovers (calendar, time list) — every one of them anchors to a trigger
// button, portals into document.body, and closes on an outside click, so
// that logic lives here once instead of in each popover component.
export function usePopoverAnchor<T extends HTMLElement, P extends HTMLElement>(
  open: boolean,
  onClose: () => void,
  minWidth = 0,
  // Rough height of the popover content — used only to decide whether it
  // fits below the trigger before it's mounted (so we can't measure the
  // real height yet). Doesn't need to be exact, just enough to catch the
  // common case: a date/time field near the bottom of a modal, where
  // opening downward would spill the popover past the dialog/viewport.
  estimatedHeight = 320
) {
  const triggerRef = useRef<T>(null);
  const popoverRef = useRef<P>(null);
  const [pos, setPos] = useState<AnchorPosition | null>(null);

  const updatePos = useCallback(() => {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    const gap = 6;
    const spaceBelow = window.innerHeight - rect.bottom;
    const spaceAbove = rect.top;
    const width = Math.max(rect.width, minWidth);
    if (spaceBelow < estimatedHeight + gap && spaceAbove > spaceBelow) {
      setPos({ bottom: window.innerHeight - rect.top + gap, left: rect.left, width });
    } else {
      setPos({ top: rect.bottom + gap, left: rect.left, width });
    }
  }, [minWidth, estimatedHeight]);

  useEffect(() => {
    if (!open) return;
    updatePos();

    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (popoverRef.current?.contains(target) || triggerRef.current?.contains(target)) return;
      onClose();
    };

    // Escape should close just this popover, not fall through to a parent
    // Radix Dialog's own Escape handling (which would dismiss the whole
    // modal underneath). Listening on window in the capture phase runs
    // before Radix's document-level capture listener, so stopPropagation()
    // here reliably wins the race.
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      e.stopPropagation();
      e.preventDefault();
      onClose();
    };

    document.addEventListener("mousedown", handleClick);
    window.addEventListener("keydown", handleKeyDown, true);
    window.addEventListener("scroll", updatePos, true);
    window.addEventListener("resize", updatePos);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      window.removeEventListener("keydown", handleKeyDown, true);
      window.removeEventListener("scroll", updatePos, true);
      window.removeEventListener("resize", updatePos);
    };
    // onClose is expected to be stable enough for this listener's lifetime;
    // re-running the effect on every render would thrash the listeners.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, updatePos]);

  return { triggerRef, popoverRef, pos, updatePos };
}
