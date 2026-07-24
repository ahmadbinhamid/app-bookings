package booking

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// These are the tests that actually matter most in this project (per
// explicit instruction): MySQL has no range-exclusion constraint, so
// Repository.LockEmployees + the FOR UPDATE re-check in Confirm/Reschedule
// IS the entire double-booking guarantee. Everything else in this package
// could be logically correct and this could still be broken under real
// concurrent load if the locking doesn't actually serialize the way the
// reasoning in repository.go's LockEmployees doc comment claims. These
// tests run REAL goroutines against REAL MySQL, synchronized on a barrier
// channel to maximize the actual race window, not mocks.

// TestConcurrentConfirm_ExactlyOneWins is run 10 times with a fresh slot
// each iteration — a single pass could pass by luck even with broken
// locking, since MySQL might happen to serialize favorably once.
func TestConcurrentConfirm_ExactlyOneWins(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	for iter := 0; iter < 10; iter++ {
		start := safeFutureTime(2).Add(time.Duration(iter) * time.Hour)
		end := start.Add(30 * time.Minute)
		blockedUntil := end.Add(10 * time.Minute)

		var wg sync.WaitGroup
		results := make([]error, 2)
		barrier := make(chan struct{})
		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func(i int) {
				defer wg.Done()
				<-barrier // release both goroutines as close to simultaneously as possible
				_, err := svc.Confirm(locID, nil, ConfirmInput{
					CustomerName: fmt.Sprintf("Customer %d iter %d", i, iter),
					Segments: []ProposedSegment{
						{ServiceID: haircut, EmployeeID: barber, Start: start, End: end, BlockedUntil: blockedUntil, Price: 25},
					},
				})
				results[i] = err
			}(i)
		}
		close(barrier)
		wg.Wait()

		successCount, conflictCount := 0, 0
		for _, err := range results {
			switch err {
			case nil:
				successCount++
			case ErrSlotNoLongerAvailable:
				conflictCount++
			}
		}
		if successCount != 1 || conflictCount != 1 {
			t.Fatalf("iter %d: expected exactly 1 success and 1 conflict racing the same slot, got successes=%d conflicts=%d errors=%v",
				iter, successCount, conflictCount, results)
		}

		// The actual ground truth: the database must contain exactly one
		// non-cancelled segment for this employee overlapping this slot —
		// not "the Go code returned the right errors" but "the data is
		// actually not double-booked."
		var count int
		if err := conn.QueryRow(`
			SELECT COUNT(*) FROM booking_segments
			WHERE employee_id = ? AND status <> 'cancelled' AND start_time < ? AND blocked_until > ?
		`, barber, blockedUntil, start).Scan(&count); err != nil {
			t.Fatalf("iter %d: count: %v", iter, err)
		}
		if count != 1 {
			t.Fatalf("iter %d: expected exactly 1 non-cancelled segment in the database, got %d — DOUBLE BOOKING OCCURRED", iter, count)
		}
	}
}

// TestConcurrentRescheduleVsConfirm_ExactlyOneWins: booking A exists at t1.
// One goroutine reschedules A to t2; another concurrently confirms a brand
// new booking B at t2 for the same employee. Exactly one must win — and
// whichever loses must leave no trace: if reschedule loses, A's original
// segment at t1 must be completely untouched (design doc's specific
// requirement); if confirm loses, no booking B exists at all.
func TestConcurrentRescheduleVsConfirm_ExactlyOneWins(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	t1 := safeFutureTime(3)
	a, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Booking A",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: t1, End: t1.Add(30 * time.Minute), BlockedUntil: t1.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm A: %v", err)
	}

	t2 := t1.Add(4 * time.Hour)
	var rescheduleErr, confirmErr error
	var confirmedB Booking

	var wg sync.WaitGroup
	barrier := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-barrier
		_, rescheduleErr = svc.Reschedule(a.ID, RescheduleInput{
			Segments: []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: t2, End: t2.Add(30 * time.Minute), BlockedUntil: t2.Add(40 * time.Minute), Price: 25}},
		})
	}()
	go func() {
		defer wg.Done()
		<-barrier
		confirmedB, confirmErr = svc.Confirm(locID, nil, ConfirmInput{
			CustomerName: "Booking B",
			Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: t2, End: t2.Add(30 * time.Minute), BlockedUntil: t2.Add(40 * time.Minute), Price: 25}},
		})
	}()
	close(barrier)
	wg.Wait()

	successCount := 0
	if rescheduleErr == nil {
		successCount++
	}
	if confirmErr == nil {
		successCount++
	}
	if successCount != 1 {
		t.Fatalf("expected exactly one of reschedule/confirm to succeed, got rescheduleErr=%v confirmErr=%v", rescheduleErr, confirmErr)
	}

	afterA, err := svc.GetByID(a.ID)
	if err != nil {
		t.Fatalf("GetByID A: %v", err)
	}
	survivingA := nonCancelled(afterA.Segments)

	if rescheduleErr == nil {
		// Reschedule won the race.
		if confirmErr != ErrSlotNoLongerAvailable {
			t.Fatalf("expected confirm to fail with ErrSlotNoLongerAvailable when reschedule won, got %v", confirmErr)
		}
		if len(survivingA) != 1 || !survivingA[0].StartTime.Equal(t2) {
			t.Fatalf("expected A's surviving segment to have moved to t2, got %+v", survivingA)
		}
	} else {
		// Confirm won the race — A's original segment must be untouched.
		if rescheduleErr != ErrSlotNoLongerAvailable {
			t.Fatalf("expected reschedule to fail with ErrSlotNoLongerAvailable when confirm won, got %v", rescheduleErr)
		}
		if len(survivingA) != 1 || !survivingA[0].StartTime.Equal(t1) || survivingA[0].Status != "booked" {
			t.Fatalf("expected A's ORIGINAL segment at t1 to be completely untouched, got %+v", survivingA)
		}
		if confirmedB.ID == "" {
			t.Fatal("expected booking B to have actually been created")
		}
	}

	var count int
	if err := conn.QueryRow(`
		SELECT COUNT(*) FROM booking_segments
		WHERE employee_id = ? AND status <> 'cancelled' AND start_time < ? AND blocked_until > ?
	`, barber, t2.Add(40*time.Minute), t2).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 non-cancelled segment at t2, got %d — DOUBLE BOOKING OCCURRED", count)
	}
}

// TestConcurrentRescheduleAndCancel_SameBooking_NoHalfUpdatedState: the
// design doc's specific requirement — "Serialize operations per booking:
// rescheduleBooking must read the old segments INSIDE the transaction,
// after locking the parent booking row ... so concurrent reschedule/cancel
// on the same booking can't interleave." Both operations lock the booking
// row first, so one must fully complete before the other starts — there is
// no valid outcome where the booking ends up with a mix of stale old
// segments and new ones all sitting 'booked' at once.
func TestConcurrentRescheduleAndCancel_SameBooking_NoHalfUpdatedState(t *testing.T) {
	conn := connectOrSkip(t)
	svc := newBookingService(conn)
	locID, barber, haircut := oneEmployeeOneService(t, conn)

	start := safeFutureTime(4)
	b, err := svc.Confirm(locID, nil, ConfirmInput{
		CustomerName: "Jane",
		Segments:     []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: start, End: start.Add(30 * time.Minute), BlockedUntil: start.Add(40 * time.Minute), Price: 25}},
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	newStart := start.Add(6 * time.Hour)
	var rescheduleErr, cancelErr error

	var wg sync.WaitGroup
	barrier := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-barrier
		_, rescheduleErr = svc.Reschedule(b.ID, RescheduleInput{
			Segments: []ProposedSegment{{ServiceID: haircut, EmployeeID: barber, Start: newStart, End: newStart.Add(30 * time.Minute), BlockedUntil: newStart.Add(40 * time.Minute), Price: 25}},
		})
	}()
	go func() {
		defer wg.Done()
		<-barrier
		cancelErr = svc.CancelBooking(b.ID)
	}()
	close(barrier)
	wg.Wait()

	// Cancel always eventually succeeds in both valid orderings (whether it
	// runs first against the original segment, or second against whatever
	// reschedule left behind) — the only thing that can legitimately fail
	// is reschedule, and only with ErrAlreadyCancelled (cancel got there
	// first and flipped the booking's status before reschedule's own lock
	// acquisition went through).
	if cancelErr != nil {
		t.Fatalf("expected cancel to always succeed eventually, got %v", cancelErr)
	}
	if rescheduleErr != nil && rescheduleErr != ErrAlreadyCancelled {
		t.Fatalf("expected reschedule to either succeed or fail with ErrAlreadyCancelled, got %v", rescheduleErr)
	}

	after, err := svc.GetByID(b.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.Status != "cancelled" {
		t.Fatalf("expected the booking to end up fully cancelled either way, got status=%q", after.Status)
	}
	// The actual "no half-updated state" check: NOT ONE segment may still
	// be 'booked' — whichever ordering occurred, every segment (whether
	// it's the original or a rescheduled replacement) must be cancelled.
	for _, seg := range after.Segments {
		if seg.Status == "booked" {
			t.Fatalf("found a segment still 'booked' after both operations completed — half-updated state: %+v", after.Segments)
		}
	}
}
