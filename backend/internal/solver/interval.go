package solver

import "time"

// Interval is a half-open absolute-time range [Start, End). No timezone
// concept lives here — by the time an Interval reaches this package, it's
// already an absolute instant (see internal/modules/booking/availability.go
// for where the wall-clock-to-instant, DST-safe conversion happens).
type Interval struct {
	Start time.Time
	End   time.Time
}

func (i Interval) overlaps(other Interval) bool {
	return i.Start.Before(other.End) && other.Start.Before(i.End)
}

// Subtract removes every `remove` range from every `base` range, returning
// whatever free slices remain. A single base interval can split into zero,
// one, or two resulting intervals depending on where a removed range falls.
// Exported so internal/modules/booking's availability computation (schedule
// − time off − existing segments) can reuse the same interval algebra Solve
// itself uses for in-proposal subtraction — one implementation, not two.
func Subtract(base []Interval, remove []Interval) []Interval {
	result := base
	for _, r := range remove {
		result = subtractOne(result, r)
	}
	return result
}

func subtractOne(base []Interval, remove Interval) []Interval {
	out := make([]Interval, 0, len(base))
	for _, b := range base {
		if !b.overlaps(remove) {
			out = append(out, b)
			continue
		}
		// Left remainder: base starts before remove does.
		if b.Start.Before(remove.Start) {
			out = append(out, Interval{Start: b.Start, End: remove.Start})
		}
		// Right remainder: base ends after remove does.
		if remove.End.Before(b.End) {
			out = append(out, Interval{Start: remove.End, End: b.End})
		}
	}
	return out
}
