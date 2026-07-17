package booking

import (
	"database/sql"
	"fmt"
	"time"

	"app-booking/internal/solver"
)

// freeIntervalsFor computes one employee's free absolute-time intervals for
// targetDate at the given location timezone: EMPLOYEE_SCHEDULE (converted
// to absolute instants for this specific date) minus EMPLOYEE_TIME_OFF
// minus existing BOOKING_SEGMENT rows — the design doc's freeIntervalsFor.
//
// targetDate is a plain calendar date — the admin's "which day am I
// booking for" choice, in the location's own local calendar, not an
// instant to be re-interpreted through loc. Only its Year/Month/Day are
// read (via targetDate.Date()); its own zone/clock time are ignored. This
// matters: converting an arbitrary instant via targetDate.In(loc) can land
// on the wrong calendar day entirely (e.g. UTC midnight on the 9th is still
// the evening of the 8th in New York) — callers must construct targetDate
// so its Y/M/D already equal the intended local date (e.g.
// time.Date(y,m,d,0,0,0,0,time.UTC), never a real "now" instant converted
// through a different zone).
//
// DST-safety: combineDateAndTimeOfDay builds each instant via time.Date(y,
// m, d, hh, mm, ss, 0, loc) for the CONCRETE target date, never a cached/
// precomputed UTC offset. Go resolves the correct offset for that specific
// date from loc's rules, so a schedule that straddles a DST transition
// (e.g. "09:00-17:00" on the day clocks change) still converts correctly —
// the offset is looked up per-date, not assumed constant.
func freeIntervalsFor(db *sql.DB, employeeID string, targetDate time.Time, loc *time.Location) ([]solver.Interval, error) {
	y, m, d := targetDate.Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	dayOfWeek := int(dayStart.Weekday()) // time.Sunday == 0, matches our schema's 0-6

	free, err := scheduleIntervals(db, employeeID, dayOfWeek, y, m, d, loc)
	if err != nil {
		return nil, fmt.Errorf("schedule intervals: %w", err)
	}
	if len(free) == 0 {
		return nil, nil // no schedule rows this day — employee simply isn't working
	}

	occupied, err := occupiedIntervals(db, employeeID, dayStart.UTC(), dayEnd.UTC())
	if err != nil {
		return nil, fmt.Errorf("occupied intervals: %w", err)
	}

	return solver.Subtract(free, occupied), nil
}

func scheduleIntervals(db *sql.DB, employeeID string, dayOfWeek, y int, m time.Month, d int, loc *time.Location) ([]solver.Interval, error) {
	rows, err := db.Query(`
		SELECT start_time, end_time FROM employee_schedules
		WHERE employee_id = ? AND day_of_week = ?
	`, employeeID, dayOfWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []solver.Interval
	for rows.Next() {
		var startStr, endStr string
		if err := rows.Scan(&startStr, &endStr); err != nil {
			return nil, err
		}
		start, err := combineDateAndTimeOfDay(y, m, d, startStr, loc)
		if err != nil {
			return nil, err
		}
		end, err := combineDateAndTimeOfDay(y, m, d, endStr, loc)
		if err != nil {
			return nil, err
		}
		out = append(out, solver.Interval{Start: start.UTC(), End: end.UTC()})
	}
	return out, rows.Err()
}

// combineDateAndTimeOfDay is the DST-safe conversion point this file's doc
// comment describes — see it for why time.Date (not a fixed-offset
// arithmetic shortcut) is what makes this safe.
func combineDateAndTimeOfDay(y int, m time.Month, d int, hhmmss string, loc *time.Location) (time.Time, error) {
	t, err := time.Parse("15:04:05", hhmmss)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", hhmmss, err)
	}
	return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), 0, loc), nil
}

func occupiedIntervals(db *sql.DB, employeeID string, dayStartUTC, dayEndUTC time.Time) ([]solver.Interval, error) {
	var out []solver.Interval

	toRows, err := db.Query(`
		SELECT start_datetime, end_datetime FROM employee_time_off
		WHERE employee_id = ? AND start_datetime < ? AND end_datetime > ?
	`, employeeID, dayEndUTC, dayStartUTC)
	if err != nil {
		return nil, err
	}
	for toRows.Next() {
		var s, e time.Time
		if err := toRows.Scan(&s, &e); err != nil {
			toRows.Close()
			return nil, err
		}
		out = append(out, solver.Interval{Start: s, End: e})
	}
	if err := toRows.Err(); err != nil {
		toRows.Close()
		return nil, err
	}
	toRows.Close()

	// status <> 'cancelled': a cancelled segment must never block
	// availability (design doc, Status rules — "cancelling frees the time
	// automatically").
	segRows, err := db.Query(`
		SELECT start_time, blocked_until FROM booking_segments
		WHERE employee_id = ? AND status <> 'cancelled' AND start_time < ? AND blocked_until > ?
	`, employeeID, dayEndUTC, dayStartUTC)
	if err != nil {
		return nil, err
	}
	defer segRows.Close()
	for segRows.Next() {
		var s, e time.Time
		if err := segRows.Scan(&s, &e); err != nil {
			return nil, err
		}
		out = append(out, solver.Interval{Start: s, End: e})
	}
	return out, segRows.Err()
}
