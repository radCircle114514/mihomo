package constant

import (
	"fmt"
	"time"
)

// TimeWindow is a daily local-time interval, with an inclusive start and exclusive end.
type TimeWindow struct {
	Start int
	End   int
}

// ParseTimeWindow parses HH:MM times. Equal endpoints are invalid.
func ParseTimeWindow(start, end string) (TimeWindow, error) {
	parse := func(value string) (int, error) {
		if len(value) != 5 || value[2] != ':' {
			return 0, fmt.Errorf("invalid time %q: expected HH:MM", value)
		}
		t, err := time.Parse("15:04", value)
		if err != nil {
			return 0, fmt.Errorf("invalid time %q: expected HH:MM", value)
		}
		return t.Hour()*60 + t.Minute(), nil
	}
	from, err := parse(start)
	if err != nil {
		return TimeWindow{}, err
	}
	to, err := parse(end)
	if err != nil {
		return TimeWindow{}, err
	}
	if from == to {
		return TimeWindow{}, fmt.Errorf("start and end must differ")
	}
	return TimeWindow{Start: from, End: to}, nil
}

// Contains reports whether t falls in the window, including overnight windows.
func (w TimeWindow) Contains(t time.Time) bool {
	minute := t.Hour()*60 + t.Minute()
	if w.Start < w.End {
		return minute >= w.Start && minute < w.End
	}
	return minute >= w.Start || minute < w.End
}
