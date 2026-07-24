package booking

import (
	"database/sql"
	"testing"
	"time"

	"app-booking/internal/solver"
)

// --- Propose ---

// Recreates the exact design doc worked example: Haircut (30min/15min
// buffer) then Facial (30min/15min buffer) from 11:30. Barber A is free all
// day; Facial Tech B is booked 09:00-12:15 with a 15min buffer (blocked
// until 12:30), so their real free interval only starts at 12:30 — the
// resulting 30-minute gap (12:00-12:30) belongs to no one.
func TestPropose_GapExample_HaircutThenFacial(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barberA := seedEmployeeAt(t, conn, locID, "Barber A")
	facialTechB := seedEmployeeAt(t, conn, locID, "Facial Tech B")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 15, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 15, 40)
	seedAssignment(t, conn, barberA, haircut)
	seedAssignment(t, conn, facialTechB, facial)

	monday := nextWeekday(time.Now().UTC(), time.Monday)
	seedSchedule(t, conn, barberA, int(time.Monday), "09:00:00", "17:00:00")
	seedSchedule(t, conn, facialTechB, int(time.Monday), "09:00:00", "17:00:00")

	// Facial Tech B already has a booking 09:00-12:15, blocked until 12:30.
	priorBooking := seedRawBooking(t, conn, locID, "confirmed")
	seedRawSegment(t, conn, priorBooking, facialTechB, facial, "booked",
		monday.Add(9*time.Hour), monday.Add(12*time.Hour+15*time.Minute), monday.Add(12*time.Hour+30*time.Minute), 40)

	earliestStart := monday.Add(11*time.Hour + 30*time.Minute)
	proposal, err := svc.Propose(locID, ProposeInput{
		Services:      []ServiceRequest{{ServiceID: haircut}, {ServiceID: facial}},
		EarliestStart: earliestStart,
	})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(proposal.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(proposal.Segments))
	}

	hc, fc := proposal.Segments[0], proposal.Segments[1]
	if hc.EmployeeID != barberA || !hc.Start.Equal(monday.Add(11*time.Hour+30*time.Minute)) || !hc.End.Equal(monday.Add(12*time.Hour)) {
		t.Fatalf("unexpected haircut segment: %+v", hc)
	}
	if fc.EmployeeID != facialTechB || !fc.Start.Equal(monday.Add(12*time.Hour+30*time.Minute)) || !fc.End.Equal(monday.Add(13*time.Hour)) {
		t.Fatalf("unexpected facial segment (expected the 30min gap 12:00-12:30): %+v", fc)
	}
	if !proposal.WindowStart.Equal(hc.Start) || !proposal.WindowEnd.Equal(fc.End) {
		t.Fatalf("unexpected window: start=%v end=%v", proposal.WindowStart, proposal.WindowEnd)
	}
	if proposal.TotalPrice != 65 {
		t.Fatalf("expected total_price 65, got %v", proposal.TotalPrice)
	}
}

func TestPropose_LocationTimezoneUnset_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", false) // NOT confirmed

	_, err := svc.Propose(locID, ProposeInput{
		Services:      []ServiceRequest{{ServiceID: "irrelevant"}},
		EarliestStart: time.Now().Add(24 * time.Hour),
	})
	if err != ErrLocationTimezoneUnset {
		t.Fatalf("expected ErrLocationTimezoneUnset, got %v", err)
	}
}

func TestPropose_NoEmployeeAssigned(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 0, 25)
	// No employee assigned to this service at all.

	_, err := svc.Propose(locID, ProposeInput{
		Services:      []ServiceRequest{{ServiceID: haircut}},
		EarliestStart: time.Now().Add(24 * time.Hour),
	})
	var naErr *solver.NoAvailabilityError
	if err == nil {
		t.Fatal("expected an error")
	}
	if !asNoAvailability(err, &naErr) || naErr.Reason != solver.ReasonNoEmployeeAssigned {
		t.Fatalf("expected NO_EMPLOYEE_ASSIGNED, got %v", err)
	}
}

func asNoAvailability(err error, target **solver.NoAvailabilityError) bool {
	if e, ok := err.(*solver.NoAvailabilityError); ok {
		*target = e
		return true
	}
	return false
}

// --- Confirm ---

func TestConfirm_HappyPath(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	seedAssignment(t, conn, barber, haircut)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	end := start.Add(30 * time.Minute)
	blockedUntil := end.Add(10 * time.Minute)

	phone := "555-1234"
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName:  "Jane Doe",
		CustomerPhone: &phone,
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: end, BlockedUntil: blockedUntil, Price: 25},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if b.Status != "confirmed" {
		t.Fatalf("expected status confirmed, got %q", b.Status)
	}
	if len(b.Segments) != 1 || b.Segments[0].Status != "booked" {
		t.Fatalf("unexpected segments: %+v", b.Segments)
	}
	if b.Segments[0].OriginalPrice != 25 || b.Segments[0].Price != 25 {
		t.Fatalf("expected original_price=price=25, got %+v", b.Segments[0])
	}
	if b.TotalPrice != 25 {
		t.Fatalf("expected total_price 25, got %v", b.TotalPrice)
	}
}

func TestConfirm_InvalidProposal_PairwiseOverlap_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	trim := seedServiceAt(t, conn, locID, "Beard Trim", 20, 5, 15)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, trim)

	start := safeFutureTime(2)

	_, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane Doe",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			// Overlaps the haircut's own blocked_until — same employee.
			{ServiceID: trim, EmployeeID: barber, Start: start.Add(20 * time.Minute), End: start.Add(40 * time.Minute), BlockedUntil: start.Add(45 * time.Minute), Price: 15},
		},
	})
	if err != ErrInvalidProposal {
		t.Fatalf("expected ErrInvalidProposal, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ?`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted after an invalid proposal, found %d", count)
	}
}

func TestConfirm_SlotNoLongerAvailable_ExistingSegmentConflicts(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	seedAssignment(t, conn, barber, haircut)

	start := safeFutureTime(2)
	priorBooking := seedRawBooking(t, conn, locID, "confirmed")
	seedRawSegment(t, conn, priorBooking, barber, haircut, "booked", start, start.Add(30*time.Minute), start.Add(40*time.Minute), 25)

	// Overlaps the pre-existing segment.
	_, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "New Customer",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start.Add(15 * time.Minute), End: start.Add(45 * time.Minute), BlockedUntil: start.Add(55 * time.Minute), Price: 25},
		},
	})
	if err != ErrSlotNoLongerAvailable {
		t.Fatalf("expected ErrSlotNoLongerAvailable, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'New Customer'`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no new booking to be persisted, found %d", count)
	}
}

// TestConfirm_PriceTampering_ServerPriceWins covers the "fully optimistic"
// model's sharpest edge: nothing stops a client from calling /propose, then
// POSTing /confirm with the same segments but an arbitrary price. Confirm
// must never trust seg.Price — the persisted price must always be the
// service's own current price, regardless of what the client submitted.
func TestConfirm_PriceTampering_ServerPriceWins(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn) // Haircut costs 25

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Bargain Hunter",
		Segments: []ProposedSegment{
			// Real price is 25 — try to confirm it for a cent.
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 0.01},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if b.Segments[0].OriginalPrice != 25 || b.Segments[0].Price != 25 {
		t.Fatalf("expected the server's real price (25) to win over the tampered client price (0.01), got %+v", b.Segments[0])
	}
	if b.TotalPrice != 25 {
		t.Fatalf("expected total_price 25 (server price), got %v", b.TotalPrice)
	}
}

// TestConfirm_InvalidSegment_ServiceBelongsToDifferentLocation covers the
// cross-location gap: Propose already checks this, but Confirm never
// repeated it, so a client authorized for locID A submitting a service that
// actually belongs to locID B went through untouched before this fix.
func TestConfirm_InvalidSegment_ServiceBelongsToDifferentLocation(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locA, barber, _ := oneEmployeeOneService(t, conn)
	locB := seedLocationTZ(t, conn, "UTC", true)
	otherLocationsHaircut := seedServiceAt(t, conn, locB, "Haircut", 30, 10, 25)

	start := safeFutureTime(2)
	_, err := svc.Confirm(locA, nil, ConfirmInput{
		CustomerName: "Cross Location Customer",
		Segments: []ProposedSegment{
			{ServiceID: otherLocationsHaircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
		},
	})
	if err != ErrInvalidSegment {
		t.Fatalf("expected ErrInvalidSegment, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'Cross Location Customer'`, locA).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted, found %d", count)
	}
}

// TestConfirm_InvalidSegment_EmployeeDeactivatedAfterPropose is the same
// gap as the schedule/time-off one, for a different table: Propose filters
// to active + qualified employees via QualifiedActiveEmployeeIDs, but that
// was never re-checked at Confirm time.
func TestConfirm_InvalidSegment_EmployeeDeactivatedAfterPropose(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)

	// Another admin deactivates the employee (e.g. they left) between
	// Propose and Confirm.
	if _, err := conn.Exec(`UPDATE employees SET active = FALSE WHERE id = ?`, barber); err != nil {
		t.Fatalf("deactivate employee: %v", err)
	}

	_, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Stale Proposal Customer 3",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
		},
	})
	if err != ErrInvalidSegment {
		t.Fatalf("expected ErrInvalidSegment, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'Stale Proposal Customer 3'`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted, found %d", count)
	}
}

// TestConfirm_InvalidSegment_EmployeeUnassignedFromService covers the other
// half: the employee is still active, but the EMPLOYEE_SERVICE assignment
// that qualified them for this service was removed between Propose and
// Confirm.
func TestConfirm_InvalidSegment_EmployeeUnassignedFromService(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)

	if _, err := conn.Exec(`DELETE FROM employee_services WHERE employee_id = ? AND service_id = ?`, barber, haircut); err != nil {
		t.Fatalf("remove assignment: %v", err)
	}

	_, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Stale Proposal Customer 4",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
		},
	})
	if err != ErrInvalidSegment {
		t.Fatalf("expected ErrInvalidSegment, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'Stale Proposal Customer 4'`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted, found %d", count)
	}
}

// TestReschedule_PriceTampering_ServerPriceWins is the Reschedule-side
// equivalent of TestConfirm_PriceTampering_ServerPriceWins, specifically for
// a NEWLY-ADDED service (the price-carryover path for unchanged services
// already only ever uses DB-sourced values, never client input).
func TestReschedule_PriceTampering_ServerPriceWins(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Reschedule keeps the haircut AND adds a facial (real price 40) but
	// tries to confirm it for a cent.
	after, err := svc.Reschedule(b.ID, RescheduleInput{
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			{ServiceID: facial, EmployeeID: barber, Start: start.Add(40 * time.Minute), End: start.Add(70 * time.Minute), BlockedUntil: start.Add(80 * time.Minute), Price: 0.01},
		},
	})
	if err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	surviving := nonCancelled(after.Segments)
	if len(surviving) != 2 {
		t.Fatalf("expected 2 surviving segments, got %d: %+v", len(surviving), after.Segments)
	}
	facialSeg := surviving[1]
	if facialSeg.OriginalPrice != 40 || facialSeg.Price != 40 {
		t.Fatalf("expected the server's real price (40) to win over the tampered client price (0.01), got %+v", facialSeg)
	}
}

// TestConfirm_EmployeeNoLongerAvailable_ScheduleRemovedAfterPropose covers
// the gap FindConflicts alone doesn't close: between Propose and Confirm,
// another admin edits the SAME employee's EMPLOYEE_SCHEDULE (here, removes
// it entirely for that day) rather than booking a conflicting segment.
// FindConflicts would see no other booking and let this through; the new
// schedule/time-off re-check must catch it instead.
func TestConfirm_EmployeeNoLongerAvailable_ScheduleRemovedAfterPropose(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	seedAssignment(t, conn, barber, haircut)

	monday := nextWeekday(time.Now().UTC(), time.Monday)
	seedSchedule(t, conn, barber, int(time.Monday), "09:00:00", "17:00:00")

	proposal, err := svc.Propose(locID, ProposeInput{
		Services:      []ServiceRequest{{ServiceID: haircut}},
		EarliestStart: monday.Add(10 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	if len(proposal.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(proposal.Segments))
	}

	// Another admin removes the employee's working hours for that day
	// between Propose and Confirm — no other booking conflicts, so
	// FindConflicts alone would let this through.
	if _, err := conn.Exec(`DELETE FROM employee_schedules WHERE employee_id = ?`, barber); err != nil {
		t.Fatalf("remove schedule: %v", err)
	}

	_, err = svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Stale Proposal Customer",
		Segments:     proposal.Segments,
	})
	if err != ErrEmployeeNoLongerAvailable {
		t.Fatalf("expected ErrEmployeeNoLongerAvailable, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'Stale Proposal Customer'`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted after a stale proposal, found %d", count)
	}
}

// TestConfirm_EmployeeNoLongerAvailable_TimeOffAddedAfterPropose is the
// other half of the same gap: time off added (rather than the schedule
// itself changing) between Propose and Confirm.
func TestConfirm_EmployeeNoLongerAvailable_TimeOffAddedAfterPropose(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)

	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	seedAssignment(t, conn, barber, haircut)

	monday := nextWeekday(time.Now().UTC(), time.Monday)
	seedSchedule(t, conn, barber, int(time.Monday), "09:00:00", "17:00:00")

	proposal, err := svc.Propose(locID, ProposeInput{
		Services:      []ServiceRequest{{ServiceID: haircut}},
		EarliestStart: monday.Add(10 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}
	seg := proposal.Segments[0]

	// Another admin adds time off overlapping the proposed segment.
	if _, err := conn.Exec(`
		INSERT INTO employee_time_off (id, employee_id, start_datetime, end_datetime, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`, newUUID(), barber, seg.Start.Add(-time.Hour), seg.End.Add(time.Hour)); err != nil {
		t.Fatalf("seed time off: %v", err)
	}

	_, err = svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Stale Proposal Customer 2",
		Segments:     proposal.Segments,
	})
	if err != ErrEmployeeNoLongerAvailable {
		t.Fatalf("expected ErrEmployeeNoLongerAvailable, got %v", err)
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM bookings WHERE location_id = ? AND customer_name = 'Stale Proposal Customer 2'`, locID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no booking to be persisted after a stale proposal, found %d", count)
	}
}

// TestReschedule_EmployeeNoLongerAvailable_ScheduleRemovedAfterPropose is
// the same gap on the Reschedule path — the shared helper both Confirm and
// Reschedule call must be wired into both, not just Confirm.
func TestReschedule_EmployeeNoLongerAvailable_ScheduleRemovedAfterPropose(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// oneEmployeeOneService seeds a wide-open all-day schedule — remove it
	// entirely before attempting the reschedule.
	if _, err := conn.Exec(`DELETE FROM employee_schedules WHERE employee_id = ?`, barber); err != nil {
		t.Fatalf("remove schedule: %v", err)
	}

	newStart := start.Add(4 * time.Hour)
	_, err = svc.Reschedule(b.ID, RescheduleInput{
		Segments: []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: newStart, End: newStart.Add(30 * time.Minute), BlockedUntil: newStart.Add(40 * time.Minute), Price: 25}},
	})
	if err != ErrEmployeeNoLongerAvailable {
		t.Fatalf("expected ErrEmployeeNoLongerAvailable, got %v", err)
	}

	// The original segment must be untouched — same ROLLBACK guarantee as
	// TestReschedule_ConflictWithAnotherBooking_OriginalUntouched.
	afterA, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(afterA.Segments) != 1 {
		t.Fatalf("expected booking to still have exactly 1 segment, got %d", len(afterA.Segments))
	}
	if afterA.Segments[0].Status != "booked" || !afterA.Segments[0].StartTime.Equal(start) {
		t.Fatalf("expected the original segment untouched (status=booked, original start), got %+v", afterA.Segments[0])
	}
}

// --- Cancel ---

func TestCancelBooking_CascadesToAllSegments(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if err := svc.CancelBooking(b.ID); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.Status != "cancelled" {
		t.Fatalf("expected booking status cancelled, got %q", after.Status)
	}
	if after.Segments[0].Status != "cancelled" {
		t.Fatalf("expected segment cancelled, got %q", after.Segments[0].Status)
	}
}

func TestCancelBooking_WithCompletedSegment_BookingBecomesCompleted(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			{ServiceID: facial, EmployeeID: barber, Start: start.Add(40 * time.Minute), End: start.Add(70 * time.Minute), BlockedUntil: start.Add(80 * time.Minute), Price: 40},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// The haircut has already been delivered — mark it completed through
	// the real endpoint, not a raw-SQL shim.
	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("CompleteSegment: %v", err)
	}

	if err := svc.CancelBooking(b.ID); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// The rule: every segment is now completed-or-cancelled, with at least
	// one actually completed (the haircut) — the booking auto-transitions
	// to 'completed', not 'cancelled' and not 'confirmed'.
	if after.Status != "completed" {
		t.Fatalf("expected booking to become completed (one segment completed, one cancelled), got %q", after.Status)
	}
	var completedCount, cancelledCount int
	for _, s := range after.Segments {
		if s.Status == "completed" {
			completedCount++
		}
		if s.Status == "cancelled" {
			cancelledCount++
		}
	}
	if completedCount != 1 || cancelledCount != 1 {
		t.Fatalf("expected 1 completed + 1 cancelled, got segments=%+v", after.Segments)
	}
}

func TestCancelSegment_LeavesRestIntact(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			{ServiceID: facial, EmployeeID: barber, Start: start.Add(40 * time.Minute), End: start.Add(70 * time.Minute), BlockedUntil: start.Add(80 * time.Minute), Price: 40},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if err := svc.CancelSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("CancelSegment: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.Status != "confirmed" {
		t.Fatalf("expected booking to stay confirmed, got %q", after.Status)
	}
	if after.Segments[0].Status != "cancelled" || after.Segments[1].Status != "booked" {
		t.Fatalf("unexpected segment statuses: %+v", after.Segments)
	}
	// window/total should now reflect only the surviving facial segment.
	if after.TotalPrice != 40 {
		t.Fatalf("expected total_price 40 (facial only), got %v", after.TotalPrice)
	}
	if after.WindowStart == nil || !after.WindowStart.Equal(start.Add(40*time.Minute)) {
		t.Fatalf("expected window_start to move to the facial segment, got %v", after.WindowStart)
	}
}

func TestCancelSegment_AlreadyCancelled_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := svc.CancelSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	if err := svc.CancelSegment(b.ID, b.Segments[0].ID); err != ErrAlreadyCancelled {
		t.Fatalf("expected ErrAlreadyCancelled, got %v", err)
	}
}

// --- Complete ---

func TestCompleteSegment_HappyPath_OtherSegmentStillBooked_BookingStaysConfirmed(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			{ServiceID: facial, EmployeeID: barber, Start: start.Add(40 * time.Minute), End: start.Add(70 * time.Minute), BlockedUntil: start.Add(80 * time.Minute), Price: 40},
		},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("CompleteSegment: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.Segments[0].Status != "completed" {
		t.Fatalf("expected haircut segment completed, got %q", after.Segments[0].Status)
	}
	if after.Segments[1].Status != "booked" {
		t.Fatalf("expected facial segment to remain booked, got %q", after.Segments[1].Status)
	}
	// One segment is still just 'booked' (not completed, not cancelled) —
	// the booking must stay 'confirmed', not jump to 'completed' early.
	if after.Status != "confirmed" {
		t.Fatalf("expected booking to stay confirmed while a segment is still booked, got %q", after.Status)
	}
}

func TestCompleteSegment_LastRemainingSegment_BookingBecomesCompleted(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("CompleteSegment: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.Status != "completed" {
		t.Fatalf("expected a single-segment booking to become completed once its only segment is, got %q", after.Status)
	}
}

func TestCompleteSegment_AlreadyCompleted_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("first complete: %v", err)
	}
	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != ErrAlreadyCompleted {
		t.Fatalf("expected ErrAlreadyCompleted, got %v", err)
	}
}

func TestCompleteSegment_AlreadyCancelled_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := svc.CancelSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != ErrAlreadyCancelled {
		t.Fatalf("expected ErrAlreadyCancelled, got %v", err)
	}
}

func TestCancelBooking_AlreadyCompleted_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := svc.CompleteSegment(b.ID, b.Segments[0].ID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if err := svc.CancelBooking(b.ID); err != ErrAlreadyCompleted {
		t.Fatalf("expected ErrAlreadyCompleted, got %v", err)
	}
}

// --- Reschedule ---

func TestReschedule_PriceCarryover_UnchangedService(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Admin applies a discount to the confirmed segment.
	if _, err := conn.Exec(`UPDATE booking_segments SET price = 15 WHERE id = ?`, b.Segments[0].ID); err != nil {
		t.Fatalf("apply discount: %v", err)
	}

	newStart := start.Add(4 * time.Hour)
	after, err := svc.Reschedule(b.ID, RescheduleInput{
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: newStart, End: newStart.Add(30 * time.Minute), BlockedUntil: newStart.Add(40 * time.Minute), Price: 25},
		},
	})
	if err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	// GetByID returns full history (old cancelled segment + new one) — not
	// just survivors, which is intentional (audit trail). Filter to what's
	// actually still booked.
	surviving := nonCancelled(after.Segments)
	if len(surviving) != 1 {
		t.Fatalf("expected 1 surviving (non-cancelled) segment, got %d: %+v", len(surviving), after.Segments)
	}
	newSeg := surviving[0]
	if newSeg.OriginalPrice != 25 || newSeg.Price != 15 {
		t.Fatalf("expected original_price=25 price=15 (discount preserved), got %+v", newSeg)
	}
	if !newSeg.StartTime.Equal(newStart) {
		t.Fatalf("expected the new segment at the rescheduled time, got %v", newSeg.StartTime)
	}
}

func TestReschedule_NewlyAddedService_FreshPriceSnapshot(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)
	seedAllDaySchedule(t, conn, barber)

	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Reschedule keeps the haircut AND adds a facial that wasn't there before.
	after, err := svc.Reschedule(b.ID, RescheduleInput{
		Segments: []ProposedSegment{
			{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25},
			{ServiceID: facial, EmployeeID: barber, Start: start.Add(40 * time.Minute), End: start.Add(70 * time.Minute), BlockedUntil: start.Add(80 * time.Minute), Price: 40},
		},
	})
	if err != nil {
		t.Fatalf("Reschedule: %v", err)
	}
	surviving := nonCancelled(after.Segments)
	if len(surviving) != 2 {
		t.Fatalf("expected 2 surviving (non-cancelled) segments, got %d: %+v", len(surviving), after.Segments)
	}
	facialSeg := surviving[1]
	if facialSeg.OriginalPrice != 40 || facialSeg.Price != 40 {
		t.Fatalf("expected the newly-added facial to get a fresh price snapshot (40/40), got %+v", facialSeg)
	}
}

func TestReschedule_AlreadyCancelled_Rejected(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)
	start := safeFutureTime(2)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := svc.CancelBooking(b.ID); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}

	newStart := start.Add(4 * time.Hour)
	_, err = svc.Reschedule(b.ID, RescheduleInput{
		Segments: []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: newStart, End: newStart.Add(30 * time.Minute), BlockedUntil: newStart.Add(40 * time.Minute), Price: 25}},
	})
	if err != ErrAlreadyCancelled {
		t.Fatalf("expected ErrAlreadyCancelled, got %v", err)
	}
}

func TestReschedule_ConflictWithAnotherBooking_OriginalUntouched(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(2)
	a, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Booking A",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm A: %v", err)
	}

	otherStart := start.Add(4 * time.Hour)
	_, err = svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Booking B",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: otherStart, End: otherStart.Add(30 * time.Minute), BlockedUntil: otherStart.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm B: %v", err)
	}

	// Try to reschedule A into a time that collides with B.
	_, err = svc.Reschedule(a.ID, RescheduleInput{
		Segments: []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: otherStart.Add(10 * time.Minute), End: otherStart.Add(40 * time.Minute), BlockedUntil: otherStart.Add(50 * time.Minute), Price: 25}},
	})
	if err != ErrSlotNoLongerAvailable {
		t.Fatalf("expected ErrSlotNoLongerAvailable, got %v", err)
	}

	afterA, err := svc.GetByID(a.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(afterA.Segments) != 1 {
		t.Fatalf("expected booking A to still have exactly 1 segment, got %d", len(afterA.Segments))
	}
	if afterA.Segments[0].Status != "booked" || !afterA.Segments[0].StartTime.Equal(start) {
		t.Fatalf("expected booking A's original segment untouched (status=booked, original start), got %+v", afterA.Segments[0])
	}
}

func nonCancelled(segments []Segment) []Segment {
	out := make([]Segment, 0, len(segments))
	for _, s := range segments {
		if s.Status != "cancelled" {
			out = append(out, s)
		}
	}
	return out
}

// oneEmployeeOneService is the smallest fixture shared by most Confirm/
// Cancel/Reschedule tests that don't care about the solver itself.
func oneEmployeeOneService(t *testing.T, conn *sql.DB) (locationID, employeeID, serviceID string) {
	t.Helper()
	locationID = seedLocationTZ(t, conn, "UTC", true)
	employeeID = seedEmployeeAt(t, conn, locationID, "Barber A")
	serviceID = seedServiceAt(t, conn, locationID, "Haircut", 30, 10, 25)
	seedAssignment(t, conn, employeeID, serviceID)
	seedAllDaySchedule(t, conn, employeeID)
	return locationID, employeeID, serviceID
}
