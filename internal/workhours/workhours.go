package workhours

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// WeeklySchedule represents a work schedule for a full week, with each day containing multiple working shifts.
type WeeklySchedule [7][]WorkingShiftSchedule

// ParseWeeklySchedule parses a raw schedule string into a WeeklySchedule.
func ParseWeeklySchedule(raw string) (WeeklySchedule, error) {
	week := strings.Split(raw, ",")
	if len(week) != 7 {
		return WeeklySchedule{}, fmt.Errorf("expected a full week schedule: %s", raw)
	}

	var schedule WeeklySchedule

	for wd, day := range week {
		if day == "" {
			schedule[wd] = []WorkingShiftSchedule{}
			continue
		}

		var previous *WorkingShiftSchedule

		for shift := range strings.SplitSeq(day, "+") {
			s := strings.Split(shift, "-")
			if len(s) != 2 {
				return schedule, fmt.Errorf("%s's shift is invalid: %s", time.Weekday(wd).String(), day)
			}

			start, errStart := time.ParseDuration(s[0])
			end, errEnd := time.ParseDuration(s[1])

			if err := errors.Join(errStart, errEnd); err != nil {
				return schedule, fmt.Errorf("unable to parse %s's shift %s: %w", time.Weekday(wd).String(), day, err)
			}

			if start > end {
				return schedule, fmt.Errorf("%s's shift starts after it ends: %s", time.Weekday(wd).String(), shift)
			}

			if previous != nil && previous[1] > start {
				return schedule, fmt.Errorf("%s's shifts are not correctly sorted: %s", time.Weekday(wd).String(), shift)
			}

			if end >= (24 * time.Hour) {
				return schedule, fmt.Errorf("end must be less to 24h: %s", end)
			}

			previous = &WorkingShiftSchedule{start, end}
			schedule[wd] = append(schedule[wd], *previous)
		}
	}

	for _, shifts := range schedule {
		if len(shifts) > 0 {
			return schedule, nil
		}
	}

	return WeeklySchedule{}, errors.New("schedule is empty")
}

// Inverted returns a WeeklySchedule with all working hours inverted to represent non-working hours.
func (ws WeeklySchedule) Inverted() WeeklySchedule {
	var inverted WeeklySchedule

	eod := (24 * time.Hour) - 1

	for day, shifts := range ws {
		if len(shifts) == 0 {
			inverted[day] = []WorkingShiftSchedule{{0, eod}}
			continue
		}

		for i := range shifts {
			if shifts[i][0] <= 0 {
				continue
			}

			if i == 0 {
				inverted[day] = append(inverted[day], WorkingShiftSchedule{0, shifts[i][0]})
				continue
			}

			if shifts[i-1][1] >= eod {
				continue
			}

			inverted[day] = append(inverted[day], WorkingShiftSchedule{shifts[i-1][1], shifts[i][0]})
		}

		if shifts[len(shifts)-1][1] < eod {
			inverted[day] = append(inverted[day], WorkingShiftSchedule{shifts[len(shifts)-1][1], eod})
		}

		if inverted[day] == nil {
			inverted[day] = []WorkingShiftSchedule{}
		}
	}

	return inverted
}

// CurrentShift returns the current working shift for a given time, or nil if not within working hours.
func (ws WeeklySchedule) CurrentShift(t time.Time) *WorkingShift {
	year, month, day := t.Date()

	for _, schedule := range ws[t.Weekday()] {
		shift := schedule.At(year, month, day, t.Location())
		if t.After(shift[0]) && t.Before(shift[1]) {
			return &shift
		}
	}

	return nil
}

// PreviousShift returns the most recent working shift that occurred before the given time.
func (ws WeeklySchedule) PreviousShift(t time.Time) *WorkingShift {
	return ws.findClosestShift(t,
		func(a, b WorkingShiftSchedule) int { return int(a[0] - b[0]) },
		func(shift WorkingShift, t time.Time) bool { return shift[0].Before(t) && shift[1].Before(t) },
		func(year int, month time.Month, day int) time.Time {
			return time.Date(year, month, day, 0, 0, 0, 0, t.Location()).Add(-1)
		},
		7,
	)
}

// NextShift returns the next working shift that will occur after the given time.
func (ws WeeklySchedule) NextShift(t time.Time) *WorkingShift {
	return ws.findClosestShift(t,
		func(a, b WorkingShiftSchedule) int { return int(b[0] - a[0]) },
		func(shift WorkingShift, t time.Time) bool { return shift[1].After(t) && shift[0].After(t) },
		func(year int, month time.Month, day int) time.Time {
			return time.Date(year, month, day, 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
		},
		7,
	)
}

func (ws WeeklySchedule) findClosestShift(t time.Time, sortShifts func(WorkingShiftSchedule, WorkingShiftSchedule) int, isShiftValid func(shift WorkingShift, t time.Time) bool, nextTime func(year int, month time.Month, day int) time.Time, iterations int8) *WorkingShift {
	var closest *WorkingShift

	year, month, day := t.Date()

	shifts := make([]WorkingShiftSchedule, len(ws[t.Weekday()]))
	copy(shifts, ws[t.Weekday()])
	slices.SortFunc(shifts, sortShifts)

	for _, schedule := range shifts {
		shift := schedule.At(year, month, day, t.Location())
		if !isShiftValid(shift, t) {
			break
		}

		closest = &shift
	}

	if closest == nil && iterations > 0 {
		closest = ws.findClosestShift(nextTime(year, month, day), sortShifts, isShiftValid, nextTime, iterations-1)
	}

	return closest
}

// WorkingShiftSchedule represents a single working shift defined by start and end durations from midnight.
type WorkingShiftSchedule [2]time.Duration

// At converts a WorkingShiftSchedule to a concrete WorkingShift for a specific date and timezone.
func (ws WorkingShiftSchedule) At(year int, month time.Month, day int, tz *time.Location) WorkingShift {
	return WorkingShift{
		time.Date(year, month, day, 0, 0, 0, 0, tz).Add(ws[0]),
		time.Date(year, month, day, 0, 0, 0, 0, tz).Add(ws[1]),
	}
}

// WorkingShift represents a concrete working shift with specific start and end times.
type WorkingShift [2]time.Time

// String returns a human-readable representation of the working shift.
func (ws *WorkingShift) String() string {
	if ws == nil {
		return "undefined shift"
	}

	return ws[0].Format(time.DateTime) + " -> " + ws[1].Format(time.DateTime)
}
