package booking

import (
	"fmt"
	"time"

	"app-booking/internal/modules/location"
	"app-booking/internal/modules/services"
	"app-booking/internal/solver"
)

type Service struct {
	repo      *Repository
	services  *services.MyService
	locations *location.Service
}

func NewService(repo *Repository, servicesSvc *services.MyService, locationsSvc *location.Service) *Service {
	return &Service{repo: repo, services: servicesSvc, locations: locationsSvc}
}

// Propose runs the design doc's solveBooking end to end: resolves the
// location's timezone, gathers qualified-and-free candidates per service,
// and calls the pure solver. Nothing is persisted — this is the fully-
// optimistic "proposal" step (design doc, Proposal holds decision).
func (s *Service) Propose(locationID string, in ProposeInput) (Proposal, error) {
	loc, err := s.locations.GetByID(locationID)
	if err != nil {
		return Proposal{}, err
	}
	if !loc.TimezoneConfirmed {
		return Proposal{}, ErrLocationTimezoneUnset
	}
	tz, err := time.LoadLocation(loc.Timezone)
	if err != nil {
		return Proposal{}, fmt.Errorf("invalid location timezone %q: %w", loc.Timezone, err)
	}

	now := time.Now().UTC()
	earliest := in.EarliestStart
	if earliest.Before(now) {
		// Solve() itself also clamps, but targetDate below is derived from
		// earliest — if it were left in the past, targetDate could resolve
		// to a day that's already over instead of "today."
		earliest = now
	}
	localEarliest := earliest.In(tz)
	targetDate := time.Date(localEarliest.Year(), localEarliest.Month(), localEarliest.Day(), 0, 0, 0, 0, time.UTC)

	requests := make([]solver.ServiceRequest, 0, len(in.Services))
	for _, sv := range in.Services {
		svc, err := s.services.Get(sv.ServiceID)
		if err != nil {
			return Proposal{}, err
		}
		if svc.LocationID != locationID {
			return Proposal{}, services.ErrNotFound
		}

		employeeIDs, err := s.repo.QualifiedActiveEmployeeIDs(sv.ServiceID)
		if err != nil {
			return Proposal{}, err
		}

		candidates := make([]solver.CandidateEmployee, 0, len(employeeIDs))
		for _, empID := range employeeIDs {
			free, err := freeIntervalsFor(s.repo.db, empID, targetDate, tz)
			if err != nil {
				return Proposal{}, err
			}
			candidates = append(candidates, solver.CandidateEmployee{EmployeeID: empID, FreeIntervals: free})
		}

		requests = append(requests, solver.ServiceRequest{
			ServiceID:       sv.ServiceID,
			DurationMinutes: svc.DurationMinutes,
			BufferMinutes:   svc.BufferMinutes,
			Price:           svc.Price,
			Candidates:      candidates,
		})
	}

	result, err := solver.Solve(requests, in.EarliestStart, now)
	if err != nil {
		return Proposal{}, err
	}
	return toProposal(result), nil
}

func toProposal(p solver.Proposal) Proposal {
	segments := make([]ProposedSegment, len(p.Segments))
	for i, s := range p.Segments {
		segments[i] = ProposedSegment{
			ServiceID: s.ServiceID, EmployeeID: s.EmployeeID,
			Start: s.Start, End: s.End, BlockedUntil: s.BlockedUntil, Price: s.Price,
		}
	}
	return Proposal{Segments: segments, WindowStart: p.WindowStart, WindowEnd: p.WindowEnd, TotalPrice: p.TotalPrice}
}

// validatePairwise is the design doc's INVALID_PROPOSAL check: within one
// proposal, no employee's own segments may overlap each other. Runs before
// any DB work in both Confirm and Reschedule.
func validatePairwise(segments []ProposedSegment) error {
	byEmployee := map[string][]ProposedSegment{}
	for _, s := range segments {
		byEmployee[s.EmployeeID] = append(byEmployee[s.EmployeeID], s)
	}
	for _, segs := range byEmployee {
		for i := 0; i < len(segs); i++ {
			for j := i + 1; j < len(segs); j++ {
				a, b := segs[i], segs[j]
				if a.Start.Before(b.BlockedUntil) && b.Start.Before(a.BlockedUntil) {
					return ErrInvalidProposal
				}
			}
		}
	}
	return nil
}

func computeWindow(segments []ProposedSegment) (start, end time.Time, total float64) {
	start, end = segments[0].Start, segments[0].End
	for _, s := range segments {
		if s.Start.Before(start) {
			start = s.Start
		}
		if s.End.After(end) {
			end = s.End
		}
		total += s.Price
	}
	return start, end, total
}

// recomputeStatus is the shared rule behind both CancelBooking and
// CancelSegment: "the booking becomes 'cancelled' only when ALL its
// segments are cancelled — it stays 'confirmed' for any mixed state in
// between" (design doc, Status rules). Notably this means "cancel the
// whole booking" does NOT force status to 'cancelled' if any segment was
// already 'completed' — that segment isn't cancelled, so not all segments
// are, so the rule correctly keeps the booking 'confirmed'. That's the
// literal, intended consequence of the rule, not a bug.
func recomputeStatus(segments []Segment) string {
	for _, s := range segments {
		if s.Status != "cancelled" {
			return "confirmed"
		}
	}
	return "cancelled"
}

// Confirm is confirmBooking from the design doc: pairwise self-check, then
// lock every referenced employee (sorted — see Repository.LockEmployees),
// then re-validate each segment against the DB before inserting anything.
func (s *Service) Confirm(locationID string, adminID *uint64, in ConfirmInput) (Booking, error) {
	if err := validatePairwise(in.Segments); err != nil {
		return Booking{}, err
	}

	tx, err := s.repo.BeginTx()
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	employeeIDs := make([]string, 0, len(in.Segments))
	for _, seg := range in.Segments {
		employeeIDs = append(employeeIDs, seg.EmployeeID)
	}
	if err := s.repo.LockEmployees(tx, employeeIDs); err != nil {
		return Booking{}, err
	}

	for _, seg := range in.Segments {
		conflicts, err := s.repo.FindConflicts(tx, seg.EmployeeID, seg.Start, seg.BlockedUntil)
		if err != nil {
			return Booking{}, err
		}
		if len(conflicts) > 0 {
			return Booking{}, ErrSlotNoLongerAvailable
		}
	}

	windowStart, windowEnd, total := computeWindow(in.Segments)
	bookingID, err := s.repo.InsertBooking(tx, locationID, adminID, in.CustomerName, in.CustomerPhone, in.CustomerEmail, windowStart, windowEnd, total)
	if err != nil {
		return Booking{}, err
	}

	for i, seg := range in.Segments {
		if err := s.repo.InsertSegment(tx, bookingID, seg.EmployeeID, seg.ServiceID, i, seg.Start, seg.End, seg.BlockedUntil, seg.Price, seg.Price); err != nil {
			return Booking{}, err
		}
	}
	if err := s.repo.UpdateBookingStatus(tx, bookingID, "confirmed"); err != nil {
		return Booking{}, err
	}

	if err := tx.Commit(); err != nil {
		return Booking{}, err
	}
	return s.repo.GetByID(bookingID)
}

// CancelBooking cascades cancellation to every non-completed segment in
// one transaction (design doc, Status rules) — see recomputeStatus's doc
// comment for what happens to booking.status when a completed segment
// survives.
func (s *Service) CancelBooking(bookingID string) error {
	tx, err := s.repo.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	b, err := s.repo.LockBookingForUpdate(tx, bookingID)
	if err != nil {
		return err
	}
	if b.Status == "cancelled" {
		return ErrAlreadyCancelled
	}
	if b.Status == "completed" {
		return ErrAlreadyCompleted
	}

	segments, err := s.repo.SegmentsForBooking(tx, bookingID, false)
	if err != nil {
		return err
	}
	for _, seg := range segments {
		if seg.Status == "booked" {
			if err := s.repo.UpdateSegmentStatus(tx, seg.ID, "cancelled"); err != nil {
				return err
			}
		}
	}

	updated, err := s.repo.SegmentsForBooking(tx, bookingID, false)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateBookingStatus(tx, bookingID, recomputeStatus(updated)); err != nil {
		return err
	}
	if err := s.repo.RecomputeBookingWindowAndTotal(tx, bookingID); err != nil {
		return err
	}

	return tx.Commit()
}

// CancelSegment cancels one segment of a multi-service booking, leaving
// the rest alone — design doc: "Individual segments can be cancelled on
// their own." Locks the parent booking row first, same as CancelBooking,
// so this can't interleave with a concurrent reschedule/whole-cancel on
// the same booking.
func (s *Service) CancelSegment(bookingID, segmentID string) error {
	tx, err := s.repo.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := s.repo.LockBookingForUpdate(tx, bookingID); err != nil {
		return err
	}

	seg, err := s.repo.GetSegment(tx, segmentID)
	if err != nil {
		return err
	}
	if seg.BookingID != bookingID {
		return ErrSegmentNotFound
	}
	if seg.Status == "cancelled" {
		return ErrAlreadyCancelled
	}
	if seg.Status == "completed" {
		return ErrAlreadyCompleted
	}

	if err := s.repo.UpdateSegmentStatus(tx, segmentID, "cancelled"); err != nil {
		return err
	}

	allSegments, err := s.repo.SegmentsForBooking(tx, bookingID, false)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateBookingStatus(tx, bookingID, recomputeStatus(allSegments)); err != nil {
		return err
	}
	if err := s.repo.RecomputeBookingWindowAndTotal(tx, bookingID); err != nil {
		return err
	}

	return tx.Commit()
}

// Reschedule is rescheduleBooking from the design doc: lock the booking
// row first, read old (non-cancelled) segments inside the transaction,
// cancel them BEFORE checking/inserting the new ones (so the exclusion/
// conflict check never mistakes an employee's own about-to-be-replaced
// segment for someone else's conflict), then carry over original_price
// AND price for services that are unchanged between old and new — only a
// newly-added service gets a fresh price snapshot (already baked into
// ProposedSegment.Price from Propose).
func (s *Service) Reschedule(bookingID string, in RescheduleInput) (Booking, error) {
	if err := validatePairwise(in.Segments); err != nil {
		return Booking{}, err
	}

	tx, err := s.repo.BeginTx()
	if err != nil {
		return Booking{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	b, err := s.repo.LockBookingForUpdate(tx, bookingID)
	if err != nil {
		return Booking{}, err
	}
	if b.Status == "cancelled" {
		return Booking{}, ErrAlreadyCancelled
	}
	if b.Status == "completed" {
		return Booking{}, ErrAlreadyCompleted
	}

	oldSegments, err := s.repo.SegmentsForBooking(tx, bookingID, true)
	if err != nil {
		return Booking{}, err
	}

	// Price carryover queue, keyed by service_id — FIFO in case a booking
	// somehow has more than one segment of the same service.
	oldByService := map[string][]Segment{}
	for _, seg := range oldSegments {
		oldByService[seg.ServiceID] = append(oldByService[seg.ServiceID], seg)
	}

	// Lock every employee touched by EITHER the old or the new segments,
	// sorted, BEFORE cancelling or inserting anything.
	//
	// This ordering fix came out of an actual deadlock (Error 1213) caught
	// by the concurrency tests run repeatedly, not just reasoned about in
	// advance: the first version locked the booking row, then cancelled old
	// segments (implicitly locking those rows), and only THEN locked
	// employees — while Confirm locks employees FIRST and only touches
	// segment rows after. Two transactions taking the same two resources in
	// opposite orders is a textbook deadlock, and it reproduced under real
	// concurrent load roughly one run in three. Locking employees first
	// here, before any segment row is touched, makes the acquisition order
	// identical to Confirm's in every transaction, which is what actually
	// eliminates the deadlock rather than just making it less likely.
	employeeIDs := make([]string, 0, len(oldSegments)+len(in.Segments))
	for _, seg := range oldSegments {
		employeeIDs = append(employeeIDs, seg.EmployeeID)
	}
	for _, seg := range in.Segments {
		employeeIDs = append(employeeIDs, seg.EmployeeID)
	}
	if err := s.repo.LockEmployees(tx, employeeIDs); err != nil {
		return Booking{}, err
	}

	// Cancel old BEFORE checking/inserting new — ordering matters (design
	// doc): if we checked conflicts first, an unchanged employee's own old
	// segment (still 'booked') would look like a conflict with itself.
	for _, seg := range oldSegments {
		if err := s.repo.UpdateSegmentStatus(tx, seg.ID, "cancelled"); err != nil {
			return Booking{}, err
		}
	}

	for i, seg := range in.Segments {
		conflicts, err := s.repo.FindConflicts(tx, seg.EmployeeID, seg.Start, seg.BlockedUntil)
		if err != nil {
			return Booking{}, err
		}
		if len(conflicts) > 0 {
			return Booking{}, ErrSlotNoLongerAvailable // ROLLBACK (deferred) restores the original booking exactly
		}

		originalPrice, price := seg.Price, seg.Price
		if queue := oldByService[seg.ServiceID]; len(queue) > 0 {
			old := queue[0]
			oldByService[seg.ServiceID] = queue[1:]
			originalPrice, price = old.OriginalPrice, old.Price // preserve any admin discount
		}

		if err := s.repo.InsertSegment(tx, bookingID, seg.EmployeeID, seg.ServiceID, i, seg.Start, seg.End, seg.BlockedUntil, originalPrice, price); err != nil {
			return Booking{}, err
		}
	}

	if err := s.repo.RecomputeBookingWindowAndTotal(tx, bookingID); err != nil {
		return Booking{}, err
	}

	if err := tx.Commit(); err != nil {
		return Booking{}, err
	}
	return s.repo.GetByID(bookingID)
}

func (s *Service) GetByID(bookingID string) (Booking, error) {
	return s.repo.GetByID(bookingID)
}

func (s *Service) ListByLocation(locationID string, employeeID *string, from, to *time.Time) ([]Booking, error) {
	return s.repo.ListByLocation(locationID, employeeID, from, to)
}
