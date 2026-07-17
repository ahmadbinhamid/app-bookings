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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)

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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

// --- Cancel ---

func TestCancelBooking_CascadesToAllSegments(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

func TestCancelBooking_WithCompletedSegment_StaysConfirmed(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID := seedLocationTZ(t, conn, "UTC", true)
	barber := seedEmployeeAt(t, conn, locID, "Barber A")
	haircut := seedServiceAt(t, conn, locID, "Haircut", 30, 10, 25)
	facial := seedServiceAt(t, conn, locID, "Facial", 30, 10, 40)
	seedAssignment(t, conn, barber, haircut)
	seedAssignment(t, conn, barber, facial)

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

	// Simulate the haircut having already been completed (no dedicated
	// "complete" endpoint exists yet — see final report's flagged gap).
	if _, err := conn.Exec(`UPDATE booking_segments SET status = 'completed' WHERE id = ?`, b.Segments[0].ID); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	if err := svc.CancelBooking(b.ID); err != nil {
		t.Fatalf("CancelBooking: %v", err)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// The rule: booking becomes 'cancelled' only when ALL segments are
	// cancelled. One is 'completed' (never touched by cancel), so the
	// booking must stay 'confirmed' even though the admin just clicked
	// "cancel booking."
	if after.Status != "confirmed" {
		t.Fatalf("expected booking to stay confirmed (one segment completed), got %q", after.Status)
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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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
	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

// --- Reschedule ---

func TestReschedule_PriceCarryover_UnchangedService(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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
	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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

	start := time.Now().Add(48 * time.Hour).Truncate(time.Minute)
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
	return locationID, employeeID, serviceID
}
