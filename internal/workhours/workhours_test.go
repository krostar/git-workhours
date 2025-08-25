package workhours

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func getRegularWorkhoursSchedule() WeeklySchedule {
	return WeeklySchedule{
		{},
		{{8 * time.Hour, 18 * time.Hour}},
		{{8 * time.Hour, 18 * time.Hour}},
		{{8 * time.Hour, 18 * time.Hour}},
		{{8 * time.Hour, 18 * time.Hour}},
		{{8 * time.Hour, 18 * time.Hour}},
		{},
	}
}

func getCustomWorkhoursSchedule() WeeklySchedule {
	return WeeklySchedule{
		{},
		{{8 * time.Hour, 12 * time.Hour}, {14 * time.Hour, 19 * time.Hour}},
		{{8 * time.Hour, 12 * time.Hour}, {14 * time.Hour, 19 * time.Hour}},
		{{8 * time.Hour, 13 * time.Hour}},
		{{8 * time.Hour, 12 * time.Hour}, {14 * time.Hour, 19 * time.Hour}},
		{{8 * time.Hour, 12 * time.Hour}, {14 * time.Hour, 19 * time.Hour}},
		{},
	}
}

func compareWorkingShifts(t test.TestingT, want, got WorkingShift) (test.TestingT, bool, string) {
	same := got[0].Equal(want[0]) && got[1].Equal(want[1])

	msg := "shift are not the same"
	if same {
		msg = "same shift"
	}

	return t, same, fmt.Sprintf("%s:\n%s -> %s\n%s -> %s", msg, want[0].String(), want[1].String(), got[0].String(), got[1].String())
}

func Test_ParseWeeklySchedule(t *testing.T) {
	for name, tc := range map[string]struct {
		repr               string
		expected           WeeklySchedule
		expectErrorMessage string
	}{
		"regular": {
			repr:     "9h-18h,9h-18h,9h-18h,9h-18h,9h-18h,9h-18h,9h-18h",
			expected: WeeklySchedule{{{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}, {{time.Hour * 9, time.Hour * 18}}},
		},
		"empty days": {
			repr:     ",,,9h-18h,,,",
			expected: WeeklySchedule{{}, {}, {}, {{time.Hour * 9, time.Hour * 18}}, {}, {}, {}},
		},
		"multiple shift same day": {
			repr:     ",,,9h-12h+13h-18h,,,",
			expected: WeeklySchedule{{}, {}, {}, {{time.Hour * 9, time.Hour * 12}, {time.Hour * 13, time.Hour * 18}}, {}, {}, {}},
		},
		"invalid shift": {
			repr:               ",,,9h,,,",
			expectErrorMessage: "Wednesday's shift is invalid",
		},
		"unable to parse shift": {
			repr:               ",,,9j-18h,,,",
			expectErrorMessage: "unable to parse Wednesday's shift",
		},
		"impossible shift": {
			repr:               ",,,12h-9h,,,",
			expectErrorMessage: "Wednesday's shift starts after it ends",
		},
		"shift overlap": {
			repr:               ",,,9h-12h+10h-13h,,,",
			expectErrorMessage: "Wednesday's shifts are not correctly sorted",
		},
		"shift not sorted": {
			repr:               ",,,14h-18h+9h-12h,,,",
			expectErrorMessage: "Wednesday's shifts are not correctly sorted",
		},
		"end over 24h": {
			repr:               ",,,9h-24h,,,",
			expectErrorMessage: "end must be less to 24h",
		},
		"invalid number of days": {
			repr:               ",,,,,,,,",
			expectErrorMessage: "expected a full week schedul",
		},
		"full empty": {
			repr:               ",,,,,,",
			expectErrorMessage: "schedule is empty",
		},
	} {
		t.Run(name, func(t *testing.T) {
			ws, err := ParseWeeklySchedule(tc.repr)
			if tc.expectErrorMessage == "" {
				test.Assert(t, err == nil, err)
				test.Assert(check.Compare(t, ws, tc.expected))
			} else {
				test.Require(t, err != nil && strings.Contains(err.Error(), tc.expectErrorMessage), err)
			}
		})
	}
}

func Test_WeeklySchedule_Inverted(t *testing.T) {
	for name, tc := range map[string]struct {
		ws       WeeklySchedule
		expected WeeklySchedule
	}{
		"empty": {
			ws: WeeklySchedule{},
			expected: WeeklySchedule{
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
			},
		},
		"regular": {
			ws: getRegularWorkhoursSchedule(),
			expected: WeeklySchedule{
				{{0, (24 * time.Hour) - 1}},
				{{0, 8 * time.Hour}, {18 * time.Hour, (24 * time.Hour) - 1}},
				{{0, 8 * time.Hour}, {18 * time.Hour, (24 * time.Hour) - 1}},
				{{0, 8 * time.Hour}, {18 * time.Hour, (24 * time.Hour) - 1}},
				{{0, 8 * time.Hour}, {18 * time.Hour, (24 * time.Hour) - 1}},
				{{0, 8 * time.Hour}, {18 * time.Hour, (24 * time.Hour) - 1}},
				{{0, (24 * time.Hour) - 1}},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			test.Assert(check.Compare(t, tc.ws.Inverted(), tc.expected))
			test.Assert(check.Compare(t, tc.ws.Inverted().Inverted().Inverted(), tc.expected))
		})
	}
}

func Test_WeeklySchedule_CurrentShift(t *testing.T) {
	for name, tc := range map[string]struct {
		ws          WeeklySchedule
		time        time.Time
		expectShift *WorkingShift
	}{
		"empty schedule": {
			time:        time.Date(2020, time.March, 29, 9, 24, 38, 420, time.UTC),
			expectShift: nil,
		},
		"sunday on regular schedule": {
			ws:          getRegularWorkhoursSchedule(),
			time:        time.Date(2020, time.March, 29, 9, 24, 38, 420, time.UTC),
			expectShift: nil,
		},
		"friday evening on regular schedule": {
			ws:          getRegularWorkhoursSchedule(),
			time:        time.Date(2020, time.March, 27, 22, 0, 23, 123, time.UTC),
			expectShift: nil,
		},
		"friday afternoon on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 27, 14, 3, 10, 391, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 27, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 27, 18, 0, 0, 0, time.UTC),
			},
		},
		"monday afternoon on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 16, 39, 13, 830, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 8, 14, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 8, 19, 0, 0, 0, time.UTC),
			},
		},
		"monday mid-morning on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 9, 54, 19, 282, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 8, 12, 0, 0, 0, time.UTC),
			},
		},
		"monday early-morning on custom schedule": {
			ws:          getCustomWorkhoursSchedule(),
			time:        time.Date(2024, time.July, 8, 7, 9, 52, 29, time.UTC),
			expectShift: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			shift := tc.ws.CurrentShift(tc.time)
			test.Assert(t, (shift != nil && tc.expectShift != nil) || (shift == nil && tc.expectShift == nil))

			if tc.expectShift != nil && shift != nil {
				test.Assert(compareWorkingShifts(t, *tc.expectShift, *shift))
			}
		})
	}
}

func Test_WeeklySchedule_PreviousShift(t *testing.T) {
	for name, tc := range map[string]struct {
		ws          WeeklySchedule
		time        time.Time
		expectShift *WorkingShift
	}{
		"empty schedule": {
			time:        time.Date(2020, time.March, 29, 9, 24, 38, 420, time.UTC),
			expectShift: nil,
		},
		"sunday on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 29, 9, 24, 38, 420, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 27, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 27, 18, 0, 0, 0, time.UTC),
			},
		},
		"friday evening on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 27, 22, 0, 23, 123, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 27, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 27, 18, 0, 0, 0, time.UTC),
			},
		},
		"friday afternoon on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 27, 14, 3, 10, 391, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 26, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 26, 18, 0, 0, 0, time.UTC),
			},
		},
		"monday afternoon on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 16, 39, 13, 830, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 8, 12, 0, 0, 0, time.UTC),
			},
		},
		"monday mid-morning on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 9, 54, 19, 282, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 5, 14, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 5, 19, 0, 0, 0, time.UTC),
			},
		},
		"monday early-morning on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 7, 9, 52, 29, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 5, 14, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 5, 19, 0, 0, 0, time.UTC),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			shift := tc.ws.PreviousShift(tc.time)
			test.Assert(t, (shift != nil && tc.expectShift != nil) || (shift == nil && tc.expectShift == nil))

			if tc.expectShift != nil && shift != nil {
				test.Assert(compareWorkingShifts(t, *tc.expectShift, *shift))
			}
		})
	}
}

func Test_WeeklySchedule_NextShift(t *testing.T) {
	for name, tc := range map[string]struct {
		ws          WeeklySchedule
		time        time.Time
		expectShift *WorkingShift
	}{
		"empty schedule": {
			time:        time.Date(2020, time.March, 29, 9, 24, 38, 420, time.UTC),
			expectShift: nil,
		},
		"friday evening on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 27, 22, 0, 23, 123, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 30, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 30, 18, 0, 0, 0, time.UTC),
			},
		},
		"friday afternoon on regular schedule": {
			ws:   getRegularWorkhoursSchedule(),
			time: time.Date(2020, time.March, 27, 14, 3, 10, 391, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2020, time.March, 30, 8, 0, 0, 0, time.UTC),
				time.Date(2020, time.March, 30, 18, 0, 0, 0, time.UTC),
			},
		},
		"monday afternoon on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 16, 39, 13, 830, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 9, 8, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 9, 12, 0, 0, 0, time.UTC),
			},
		},
		"monday mid-morning on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 9, 54, 19, 282, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 8, 14, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 8, 19, 0, 0, 0, time.UTC),
			},
		},
		"monday early-morning on custom schedule": {
			ws:   getCustomWorkhoursSchedule(),
			time: time.Date(2024, time.July, 8, 7, 9, 52, 29, time.UTC),
			expectShift: &WorkingShift{
				time.Date(2024, time.July, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, time.July, 8, 12, 0, 0, 0, time.UTC),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			shift := tc.ws.NextShift(tc.time)
			test.Assert(t, (shift != nil && tc.expectShift != nil) || (shift == nil && tc.expectShift == nil))

			if tc.expectShift != nil && shift != nil {
				test.Assert(compareWorkingShifts(t, *tc.expectShift, *shift))
			}
		})
	}
}

func Test_WorkingShift_String(t *testing.T) {
	var ws *WorkingShift
	test.Assert(t, ws.String() == "undefined shift")

	now := time.Now()
	ws = &WorkingShift{
		now.Add(-time.Hour),
		now,
	}

	test.Assert(t, ws.String() == ws[0].Format(time.DateTime)+" -> "+ws[1].Format(time.DateTime))
}
