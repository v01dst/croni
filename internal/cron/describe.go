package cron

import (
	"fmt"
	"sort"
	"strings"
)

// FieldDesc is one human-readable line of the description.
type FieldDesc struct {
	Name  string `json:"name"`
	Raw   string `json:"raw"`
	Human string `json:"human"`
}

var dowLong = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
var monthLong = [...]string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

func ints(set []bool, min, max int) []int {
	var out []int
	for v := min; v <= max; v++ {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}

func runs(vs []int) [][2]int {
	if len(vs) == 0 {
		return nil
	}
	sort.Ints(vs)
	var out [][2]int
	start, prev := vs[0], vs[0]
	for _, v := range vs[1:] {
		if v == prev+1 {
			prev = v
			continue
		}
		out = append(out, [2]int{start, prev})
		start, prev = v, v
	}
	out = append(out, [2]int{start, prev})
	return out
}

func listOrRange(set []bool, min, max int, fmtVal func(int) string) string {
	vals := ints(set, min, max)
	rs := runs(vals)
	if len(rs) > 1 || (len(rs) == 1 && rs[0][0] != rs[0][1]) {
		parts := make([]string, 0, len(rs))
		consecutive := true
		expect := rs[0][0]
		for _, r := range rs {
			if r[0] != expect {
				consecutive = false
			}
			expect = r[1] + 1
		}
		if consecutive && len(rs) >= 2 {
			return fmt.Sprintf("%s through %s", fmtVal(rs[0][0]), fmtVal(rs[len(rs)-1][1]))
		}
		for _, r := range rs {
			if r[0] == r[1] {
				parts = append(parts, fmtVal(r[0]))
			} else {
				parts = append(parts, fmt.Sprintf("%s-%s", fmtVal(r[0]), fmtVal(r[1])))
			}
		}
		return strings.Join(parts, ", ")
	}
	return fmtVal(vals[0])
}

func plural(n int, unit string) string {
	if n == 1 {
		return "every " + unit
	}
	return fmt.Sprintf("every %d %ss", n, unit)
}

func clockStr(h, m int) string {
	return fmt.Sprintf("%02d:%02d", h, m)
}

// Describe renders each field in plain English.
func Describe(e *Expr) []FieldDesc {
	if e.Reboot {
		return []FieldDesc{{Name: "schedule", Raw: "@reboot", Human: "at every reboot — no recurring schedule"}}
	}
	full := func(set []bool, min, max int) bool {
		for v := min; v <= max; v++ {
			if !set[v] {
				return false
			}
		}
		return true
	}

	minute := "every minute of the hour"
	if !full(e.Minute[:], minMinute, maxMinute) {
		vals := ints(e.Minute[:], minMinute, maxMinute)
		if len(vals) > 1 {
			step := vals[1] - vals[0]
			uniform := true
			for i := 1; i < len(vals); i++ {
				if vals[i]-vals[i-1] != step {
					uniform = false
					break
				}
			}
			if uniform && vals[len(vals)-1]+step > maxMinute && step > 1 {
				minute = plural(step, "minute")
			} else {
				minute = "at minute " + listOrRange(e.Minute[:], minMinute, maxMinute, func(v int) string { return fmt.Sprint(v) })
			}
		} else {
			minute = "at minute " + fmt.Sprint(vals[0])
		}
	}

	hour := "every hour"
	if !full(e.Hour[:], minHour, maxHour) {
		hour = "at hour " + listOrRange(e.Hour[:], minHour, maxHour, func(v int) string { return clockStr(v, 0) })
	}

	dom := "every day of the month"
	if !full(e.Dom[:], minDom, maxDom) {
		dom = "on day " + listOrRange(e.Dom[:], minDom, maxDom, func(v int) string { return fmt.Sprint(v) }) + " of the month"
	}

	month := "every month"
	if !full(e.Month[:], minMonth, maxMonth) {
		month = "in " + listOrRange(e.Month[:], minMonth, maxMonth, func(v int) string { return monthLong[v] })
	}

	dow := "every day of the week"
	if !full(e.Dow[:], minDow, maxDow) {
		dow = "on " + listOrRange(e.Dow[:], minDow, maxDow, func(v int) string { return dowLong[v] })
	}

	return []FieldDesc{
		{"minute", e.Fields[0], minute},
		{"hour", e.Fields[1], hour},
		{"day-of-month", e.Fields[2], dom},
		{"month", e.Fields[3], month},
		{"day-of-week", e.Fields[4], dow},
	}
}

// Summary is a one-line English summary of the expression.
func Summary(e *Expr) string {
	if e.Reboot {
		return "at every reboot"
	}
	d := Describe(e)
	return fmt.Sprintf("%s, %s, %s, %s, %s", d[0].Human, d[1].Human, d[2].Human, d[3].Human, d[4].Human)
}
