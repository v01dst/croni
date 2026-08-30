// Package cron implements parsing, plain-English description and scheduling
// of standard 5-field cron expressions.
package cron

import (
	"fmt"
	"strings"
	"time"
)

// Domain bounds for the five fields.
const (
	minMinute, maxMinute = 0, 59
	minHour, maxHour     = 0, 23
	minDom, maxDom       = 1, 31
	minMonth, maxMonth   = 1, 12
	minDow, maxDow       = 0, 6 // 0 = Sunday; 7 is normalized to 0
)

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

var dowNames = map[string]int{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

// Expr is a parsed cron expression. Each field is a bitset over its domain.
type Expr struct {
	Minute [maxMinute + 1]bool
	Hour   [maxHour + 1]bool
	Dom    [maxDom + 1]bool   // index 0 unused
	Month  [maxMonth + 1]bool // index 0 unused
	Dow    [maxDow + 1]bool
	Raw    string
	Fields [5]string
	Reboot bool // @reboot — no schedule
	// bothRestricted records POSIX dow/dom OR semantics.
	bothRestricted bool
}

// IsRestricted reports whether the field was explicitly constrained
// (i.e. not a plain "*").
func (e *Expr) domRestricted() bool {
	for d := minDom; d <= maxDom; d++ {
		if !e.Dom[d] {
			return true
		}
	}
	return false
}

func (e *Expr) dowRestricted() bool {
	for d := minDow; d <= maxDow; d++ {
		if !e.Dow[d] {
			return true
		}
	}
	return false
}

// Reboot parses @reboot-style specials, returns (expr, special-handled).
func Parse(input string) (*Expr, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	e := &Expr{Raw: strings.TrimSpace(input)}

	switch s {
	case "@reboot":
		e.Reboot = true
		return e, nil
	case "@hourly":
		s = "0 * * * *"
	case "@daily", "@midnight":
		s = "0 0 * * *"
	case "@noon":
		s = "0 12 * * *"
	case "@weekly":
		s = "0 0 * * 0"
	case "@monthly":
		s = "0 0 1 * *"
	case "@yearly", "@annually":
		s = "0 0 1 1 *"
	}

	fields := strings.Fields(s)
	if len(fields) != 5 {
		if len(fields) == 6 {
			return nil, fmt.Errorf("6 fields found (seconds prefix) — croni supports the standard 5 fields: minute hour dom month dow")
		}
		return nil, fmt.Errorf("expected 5 fields (minute hour dom month dow), got %d", len(fields))
	}

	parseField := func(spec string, min, max int, names map[string]int, allow7 bool, out []bool, label string) error {
		for _, term := range strings.Split(spec, ",") {
			step := 1
			base := term
			if i := strings.Index(term, "/"); i >= 0 {
				base = term[:i]
				var err error
				step, err = atoiPositive(term[i+1:])
				if err != nil || step < 1 || step > max-min+1 {
					return fmt.Errorf("field %s: invalid step %q", label, term[i+1:])
				}
			}
			lo, hi, err := parseRange(base, min, max, names, allow7, label)
			if err != nil {
				return err
			}
			for v := lo; ; v += step {
				norm := v
				if allow7 && norm == 7 {
					norm = 0
				}
				out[norm] = true
				if v+step > hi {
					break
				}
			}
		}
		return nil
	}

	if err := parseField(fields[0], minMinute, maxMinute, nil, false, e.Minute[:], "minute"); err != nil {
		return nil, err
	}
	if err := parseField(fields[1], minHour, maxHour, nil, false, e.Hour[:], "hour"); err != nil {
		return nil, err
	}
	if err := parseField(fields[2], minDom, maxDom, nil, false, e.Dom[:], "day-of-month"); err != nil {
		return nil, err
	}
	if err := parseField(fields[3], minMonth, maxMonth, monthNames, false, e.Month[:], "month"); err != nil {
		return nil, err
	}
	if err := parseField(fields[4], minDow, maxDow, dowNames, true, e.Dow[:], "day-of-week"); err != nil {
		return nil, err
	}

	copy(e.Fields[:], fields)
	e.bothRestricted = e.domRestricted() && e.dowRestricted()
	return e, nil
}

// parseRange turns one term (without step) into an inclusive [lo, hi].
func parseRange(spec string, min, max int, names map[string]int, allow7 bool, label string) (int, int, error) {
	lookup := func(tok string) (int, error) {
		if names != nil {
			if v, ok := names[tok]; ok {
				return v, nil
			}
		}
		v, err := atoiPositive(tok)
		if err != nil {
			return 0, fmt.Errorf("field %s: bad value %q", label, tok)
		}
		return v, nil
	}

	if spec == "*" {
		return min, max, nil
	}
	if i := strings.Index(spec, "-"); i > 0 {
		lo, err := lookup(spec[:i])
		if err != nil {
			return 0, 0, err
		}
		hi, err := lookup(spec[i+1:])
		if err != nil {
			return 0, 0, err
		}
		if allow7 {
			// Normalize 7 to 0 on either end: SUN-SAT, 0-7, FRI-SUN
			if lo == 7 {
				lo = 0
			}
			if hi == 7 {
				hi = 0
			}
			if hi < lo { // wrap-around, e.g. FRI-MON
				return lo, hi + 7, nil
			}
		}
		if lo < min || hi > max || lo > hi {
			return 0, 0, fmt.Errorf("field %s: range %q out of bounds [%d-%d]", label, spec, min, max)
		}
		return lo, hi, nil
	}
	v, err := lookup(spec)
	if err != nil {
		return 0, 0, err
	}
	if allow7 && v == 7 {
		v = 0
	}
	if v < min || v > max {
		return 0, 0, fmt.Errorf("field %s: value %q out of bounds [%d-%d]", label, spec, min, max)
	}
	return v, v, nil
}

func atoiPositive(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// Matches reports whether time t satisfies the expression. POSIX OR rule
// applies when both day-of-month and day-of-week are restricted.
func (e *Expr) Matches(t time.Time) bool {
	if !e.Minute[t.Minute()] || !e.Hour[t.Hour()] || !e.Month[int(t.Month())] {
		return false
	}
	domOK := e.Dom[t.Day()]
	dowOK := e.Dow[int(t.Weekday())]
	if e.bothRestricted {
		return domOK || dowOK
	}
	return domOK && dowOK
}
