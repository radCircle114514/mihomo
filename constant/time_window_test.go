package constant

import (
	"testing"
	"time"
)

func TestTimeWindow(t *testing.T) {
	for _, tc := range []struct {
		name, start, end, now string
		want                  bool
	}{
		{"before", "02:00", "06:00", "01:59", false},
		{"start", "02:00", "06:00", "02:00", true},
		{"inside", "02:00", "06:00", "05:59", true},
		{"end", "02:00", "06:00", "06:00", false},
		{"overnight start", "22:00", "06:00", "22:00", true},
		{"midnight", "22:00", "06:00", "00:00", true},
		{"overnight end", "22:00", "06:00", "06:00", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			window, err := ParseTimeWindow(tc.start, tc.end)
			if err != nil {
				t.Fatal(err)
			}
			now, err := time.Parse("15:04", tc.now)
			if err != nil {
				t.Fatal(err)
			}
			if got := window.Contains(now); got != tc.want {
				t.Fatalf("Contains(%s) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

func TestInvalidTimeWindow(t *testing.T) {
	for _, tc := range [][2]string{{"2:00", "06:00"}, {"24:00", "06:00"}, {"02:60", "06:00"}, {"02:00", "02:00"}, {"", "06:00"}} {
		if _, err := ParseTimeWindow(tc[0], tc[1]); err == nil {
			t.Errorf("ParseTimeWindow(%q, %q) succeeded", tc[0], tc[1])
		}
	}
}
