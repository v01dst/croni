package cron

import (
	"fmt"
	"strings"
	"time"
)

// Next returns the next n run times strictly after 'from'.
// It scans minute-by-minute with a 5-year horizon.
func Next(e *Expr, from time.Time, n int) ([]time.Time, error) {
	if e.Reboot {
		return nil, fmt.Errorf("@reboot has no recurring schedule")
	}
	if n <= 0 {
		return nil, nil
	}
	t := from.Truncate(time.Minute).Add(time.Minute)
	horizon := from.AddDate(5, 0, 0)
	var out []time.Time
	for t.Before(horizon) {
		if e.Matches(t) {
			out = append(out, t)
			if len(out) == n {
				return out, nil
			}
		}
		t = t.Add(time.Minute)
	}
	return out, nil
}

// HumanizeDelta renders d as a compact relative time like "in 2d 3h 5m".
func HumanizeDelta(d time.Duration) string {
	if d < time.Minute {
		return "in less than a minute"
	}
	mins := int(d.Minutes())
	days := mins / (24 * 60)
	hours := (mins % (24 * 60)) / 60
	minutes := mins % 60
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 && days == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if len(parts) == 0 {
		return "in less than a minute"
	}
	return "in " + strings.Join(parts, " ")
}
