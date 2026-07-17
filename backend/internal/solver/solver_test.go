package solver_test

import (
	"errors"
	"testing"
	"time"

	"app-booking/internal/solver"
)

func at(hh, mm int) time.Time {
	return time.Date(2026, 7, 20, hh, mm, 0, 0, time.UTC)
}

// The worked example from the design doc: Haircut (30min, 15min buffer)
// then Facial (30min, 15min buffer), requested from 11:30. Barber A is free
// 11:30–13:00 solid; Facial Tech B is busy until 12:15, free 12:15–17:00.
// Expected: Haircut 11:30-12:00 (Barber A), Facial 12:30-13:00 (Facial Tech
// B) — the 30-minute gap falls out naturally because Tech B isn't free
// until 12:30 (their own free interval only starts at 12:15, and duration
// runs to 13:00 either way — matches the doc's timeline).
func TestSolve_GapExample_HaircutThenFacial(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, BufferMinutes: 15, Price: 25,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(11, 30), End: at(13, 0)}}},
			},
		},
		{
			ServiceID: "facial", DurationMinutes: 30, BufferMinutes: 15, Price: 40,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "facial-tech-b", FreeIntervals: []solver.Interval{{Start: at(12, 30), End: at(17, 0)}}},
			},
		},
	}

	proposal, err := solver.Solve(requests, at(11, 30), at(9, 0))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if len(proposal.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(proposal.Segments))
	}

	haircut := proposal.Segments[0]
	if haircut.EmployeeID != "barber-a" || !haircut.Start.Equal(at(11, 30)) || !haircut.End.Equal(at(12, 0)) {
		t.Fatalf("unexpected haircut segment: %+v", haircut)
	}
	if !haircut.BlockedUntil.Equal(at(12, 15)) {
		t.Fatalf("expected haircut blocked_until 12:15 (buffer), got %v", haircut.BlockedUntil)
	}

	facial := proposal.Segments[1]
	if facial.EmployeeID != "facial-tech-b" || !facial.Start.Equal(at(12, 30)) || !facial.End.Equal(at(13, 0)) {
		t.Fatalf("unexpected facial segment: %+v", facial)
	}

	// The 30-minute idle gap for Barber A (12:15-13:00) is real but belongs
	// to no one — it must never appear in window_start/window_end.
	if !proposal.WindowStart.Equal(at(11, 30)) {
		t.Fatalf("expected window_start 11:30, got %v", proposal.WindowStart)
	}
	if !proposal.WindowEnd.Equal(at(13, 0)) {
		t.Fatalf("expected window_end 13:00, got %v", proposal.WindowEnd)
	}
	if proposal.TotalPrice != 65 {
		t.Fatalf("expected total_price 65, got %v", proposal.TotalPrice)
	}
}

// The bug fixed during design review: the same employee is qualified for
// BOTH requested services. The solver must subtract the first-assigned
// segment's own [start, blockedUntil) before considering the same employee
// for the second service, or it would double-book them against themselves.
func TestSolve_SameEmployeeTwoServices_NoSelfOverlap(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, BufferMinutes: 10, Price: 25,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(17, 0)}}},
			},
		},
		{
			ServiceID: "beard-trim", DurationMinutes: 20, BufferMinutes: 5, Price: 15,
			Candidates: []solver.CandidateEmployee{
				// Only barber-a is qualified for both — no other option.
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(17, 0)}}},
			},
		},
	}

	proposal, err := solver.Solve(requests, at(9, 0), at(8, 0))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if len(proposal.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(proposal.Segments))
	}

	haircut, trim := proposal.Segments[0], proposal.Segments[1]
	if haircut.EmployeeID != "barber-a" || trim.EmployeeID != "barber-a" {
		t.Fatalf("expected both segments assigned to barber-a, got %+v / %+v", haircut, trim)
	}
	// Haircut: 09:00-09:30, blocked_until 09:40. Trim must start no earlier
	// than 09:30 (cursor = end, not blocked_until) AND not before 09:40
	// (can't overlap barber-a's own buffer) — the free-interval subtraction
	// is what enforces the second half of that, cursor alone only enforces
	// the first.
	if !haircut.Start.Equal(at(9, 0)) || !haircut.End.Equal(at(9, 30)) || !haircut.BlockedUntil.Equal(at(9, 40)) {
		t.Fatalf("unexpected haircut segment: %+v", haircut)
	}
	if trim.Start.Before(haircut.BlockedUntil) {
		t.Fatalf("beard trim (%v) must not start before barber-a's own buffer ends (%v) — self-overlap", trim.Start, haircut.BlockedUntil)
	}
	if !trim.Start.Equal(at(9, 40)) || !trim.End.Equal(at(10, 0)) {
		t.Fatalf("expected beard trim 09:40-10:00 (right after barber-a's buffer), got %+v", trim)
	}
}

func TestSolve_NoEmployeeAssigned(t *testing.T) {
	requests := []solver.ServiceRequest{
		{ServiceID: "haircut", DurationMinutes: 30, Price: 25, Candidates: nil},
	}
	_, err := solver.Solve(requests, at(9, 0), at(8, 0))
	var naErr *solver.NoAvailabilityError
	if !errors.As(err, &naErr) || naErr.Reason != solver.ReasonNoEmployeeAssigned {
		t.Fatalf("expected NoAvailabilityError(NO_EMPLOYEE_ASSIGNED), got %v", err)
	}
	if naErr.ServiceID != "haircut" {
		t.Fatalf("expected error attributed to haircut, got %q", naErr.ServiceID)
	}
}

func TestSolve_NoEmployeeFreeThatDay(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, Price: 25,
			Candidates: []solver.CandidateEmployee{
				// Free interval exists but is too short for the service.
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(9, 15)}}},
			},
		},
	}
	_, err := solver.Solve(requests, at(9, 0), at(8, 0))
	var naErr *solver.NoAvailabilityError
	if !errors.As(err, &naErr) || naErr.Reason != solver.ReasonNoEmployeeFreeToday {
		t.Fatalf("expected NoAvailabilityError(NO_EMPLOYEE_FREE_THAT_DAY), got %v", err)
	}
}

func TestSolve_EmptyServices_Rejected(t *testing.T) {
	_, err := solver.Solve(nil, at(9, 0), at(8, 0))
	if err != solver.ErrNoServicesRequested {
		t.Fatalf("expected ErrNoServicesRequested, got %v", err)
	}
}

// "A segment must fit entirely within one free interval — never split a
// service across a schedule gap or across midnight."
func TestSolve_SegmentCannotSpanAGapEvenIfBufferWouldFit(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, BufferMinutes: 15, Price: 25,
			Candidates: []solver.CandidateEmployee{
				// Free 09:00-09:40 (a lunch break starts at 09:40), then
				// free again 10:00-17:00. 30+15=45 min doesn't fit in the
				// first 40-minute window.
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{
					{Start: at(9, 0), End: at(9, 40)},
					{Start: at(10, 0), End: at(17, 0)},
				}},
			},
		},
	}

	proposal, err := solver.Solve(requests, at(9, 0), at(8, 0))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	seg := proposal.Segments[0]
	if !seg.Start.Equal(at(10, 0)) {
		t.Fatalf("expected the solver to skip the too-short first window and use 10:00, got start=%v", seg.Start)
	}
}

func TestSolve_PastEarliestStart_ClampedToNow(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, Price: 25,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(17, 0)}}},
			},
		},
	}

	// earliestStart (09:00) is in the past relative to now (10:30) — the
	// solver must clamp to now, not silently propose a slot in the past.
	proposal, err := solver.Solve(requests, at(9, 0), at(10, 30))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if !proposal.Segments[0].Start.Equal(at(10, 30)) {
		t.Fatalf("expected the segment to start at now (10:30), got %v", proposal.Segments[0].Start)
	}
}

func TestSolve_EarliestStartInFuture_NotClamped(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, Price: 25,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(17, 0)}}},
			},
		},
	}
	proposal, err := solver.Solve(requests, at(14, 0), at(9, 0))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if !proposal.Segments[0].Start.Equal(at(14, 0)) {
		t.Fatalf("expected earliestStart (14:00) to be honored since it's after now, got %v", proposal.Segments[0].Start)
	}
}

// The solver always picks the EARLIEST candidate across employees, not the
// first one listed — proves candidate order doesn't matter.
func TestSolve_PicksEarliestAcrossMultipleCandidates(t *testing.T) {
	requests := []solver.ServiceRequest{
		{
			ServiceID: "haircut", DurationMinutes: 30, Price: 25,
			Candidates: []solver.CandidateEmployee{
				{EmployeeID: "barber-a", FreeIntervals: []solver.Interval{{Start: at(11, 0), End: at(17, 0)}}},
				{EmployeeID: "barber-b", FreeIntervals: []solver.Interval{{Start: at(9, 0), End: at(17, 0)}}},
			},
		},
	}
	proposal, err := solver.Solve(requests, at(9, 0), at(8, 0))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	seg := proposal.Segments[0]
	if seg.EmployeeID != "barber-b" || !seg.Start.Equal(at(9, 0)) {
		t.Fatalf("expected barber-b at 09:00 (earliest across candidates), got %+v", seg)
	}
}
