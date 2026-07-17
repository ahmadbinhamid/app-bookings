// Package solver is the pure booking proposal algorithm from the
// app-bookings design doc — "Sequential greedy solver (MVP)". It is
// deliberately independently-testable: no database, no HTTP, no wall-clock
// reads (time.Now() never appears here — "now" is always a parameter).
// Everything DB/timezone-related (fetching an employee's free intervals,
// converting wall-clock schedules to absolute instants) happens one layer
// up, in internal/modules/booking, which calls Solve with already-resolved
// data.
package solver

import (
	"errors"
	"fmt"
	"time"
)

// ErrNoServicesRequested guards an empty input — the design doc's
// pseudocode loop only ever reaches its "segments.isEmpty()" check by
// exhausting a non-empty services list (every other path returns early via
// NoAvailabilityError), so an empty request is validated here explicitly
// rather than silently producing an empty proposal.
var ErrNoServicesRequested = errors.New("at least one service must be requested")

// Reason codes exactly as named in the design doc's "Solver edge cases"
// table.
const (
	ReasonNoEmployeeAssigned  = "NO_EMPLOYEE_ASSIGNED"
	ReasonNoEmployeeFreeToday = "NO_EMPLOYEE_FREE_THAT_DAY"
)

// NoAvailabilityError is NO_AVAILABILITY(service, reason) from the design
// doc. ServiceID is empty only if the failure isn't attributable to a
// specific service (not currently produced by Solve itself — see this
// package's doc comment on LOCATION_CLOSED_ON_TARGET_DATE, which the
// booking service detects before ever calling Solve).
type NoAvailabilityError struct {
	ServiceID string
	Reason    string
}

func (e *NoAvailabilityError) Error() string {
	return fmt.Sprintf("no availability for service %s: %s", e.ServiceID, e.Reason)
}

// ServiceRequest is one line of a booking request — a service plus the
// candidate employees already resolved as: active, assigned to this
// service (EMPLOYEE_SERVICE), with their FreeIntervals for the target date
// already computed (schedule − time off − existing DB segments). Solve
// itself does no employee-qualification lookups and no schedule math — see
// this package's doc comment.
type ServiceRequest struct {
	ServiceID       string
	DurationMinutes int
	BufferMinutes   int
	Price           float64
	Candidates      []CandidateEmployee
}

type CandidateEmployee struct {
	EmployeeID    string
	FreeIntervals []Interval
}

// Segment is one assigned slot in a Proposal — mirrors BOOKING_SEGMENT's
// client-facing (Start, End) vs employee-facing (BlockedUntil) split from
// the design doc's Buffer semantics section.
type Segment struct {
	ServiceID    string
	EmployeeID   string
	Start        time.Time
	End          time.Time
	BlockedUntil time.Time
	Price        float64
}

type Proposal struct {
	Segments    []Segment
	WindowStart time.Time
	WindowEnd   time.Time
	TotalPrice  float64
}

// Solve runs the sequential greedy solver: for each requested service in
// order, pick the earliest-available qualified employee at or after cursor,
// preferring the earliest start across all candidates. cursor advances to
// each assigned segment's End (client-facing), never BlockedUntil
// (employee-facing) — the client never waits through the next employee's
// buffer. earliestStart is clamped to now before solving starts (design
// doc's "earliestStart in the past" edge case).
func Solve(requests []ServiceRequest, earliestStart, now time.Time) (Proposal, error) {
	if len(requests) == 0 {
		return Proposal{}, ErrNoServicesRequested
	}

	cursor := earliestStart
	if now.After(cursor) {
		cursor = now
	}

	segments := make([]Segment, 0, len(requests))
	// occupied tracks, per employee, the [start, blockedUntil) ranges
	// already claimed by earlier segments IN THIS PROPOSAL — subtracted on
	// top of each candidate's FreeIntervals so the same employee can't be
	// double-booked against themselves within one solve (the same-employee-
	// two-services bug fixed during design review).
	occupied := make(map[string][]Interval)

	for _, req := range requests {
		if len(req.Candidates) == 0 {
			return Proposal{}, &NoAvailabilityError{ServiceID: req.ServiceID, Reason: ReasonNoEmployeeAssigned}
		}

		var best *Segment
		for _, cand := range req.Candidates {
			free := Subtract(cand.FreeIntervals, occupied[cand.EmployeeID])

			for _, interval := range free {
				start := interval.Start
				if cursor.After(start) {
					start = cursor
				}
				end := start.Add(time.Duration(req.DurationMinutes) * time.Minute)
				blockedUntil := end.Add(time.Duration(req.BufferMinutes) * time.Minute)

				// The segment (including its buffer) must fit entirely
				// inside this one free interval — never split across a
				// schedule gap or across midnight (design doc edge case).
				if blockedUntil.After(interval.End) {
					continue
				}
				if best == nil || start.Before(best.Start) {
					best = &Segment{
						ServiceID:    req.ServiceID,
						EmployeeID:   cand.EmployeeID,
						Start:        start,
						End:          end,
						BlockedUntil: blockedUntil,
						Price:        req.Price,
					}
				}
			}
		}

		if best == nil {
			return Proposal{}, &NoAvailabilityError{ServiceID: req.ServiceID, Reason: ReasonNoEmployeeFreeToday}
		}

		segments = append(segments, *best)
		occupied[best.EmployeeID] = append(occupied[best.EmployeeID], Interval{Start: best.Start, End: best.BlockedUntil})
		cursor = best.End // client-facing end, NOT BlockedUntil
	}

	return buildProposal(segments), nil
}

// buildProposal computes window_start/window_end/total_price exactly as
// the design doc specifies: min(segment.start), max(segment.end),
// sum(segment.price) — independent of solve order, since a later service
// in the request list can still end up with an earlier start time (e.g.
// the facial's employee turns out to be free before the haircut's).
func buildProposal(segments []Segment) Proposal {
	windowStart := segments[0].Start
	windowEnd := segments[0].End
	var total float64
	for _, s := range segments {
		if s.Start.Before(windowStart) {
			windowStart = s.Start
		}
		if s.End.After(windowEnd) {
			windowEnd = s.End
		}
		total += s.Price
	}

	return Proposal{Segments: segments, WindowStart: windowStart, WindowEnd: windowEnd, TotalPrice: total}
}
