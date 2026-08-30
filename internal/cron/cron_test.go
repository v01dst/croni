package cron

import (
	"reflect"
	"testing"
	"time"
)

func TestParseSpecials(t *testing.T) {
	for in, want := range map[string]string{
		"@hourly": "0 * * * *", "@daily": "0 0 * * *", "@midnight": "0 0 * * *",
		"@noon": "0 12 * * *", "@weekly": "0 0 * * 0", "@monthly": "0 0 1 * *",
		"@yearly": "0 0 1 1 *", "@annually": "0 0 1 1 *",
	} {
		e, err := Parse(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if e.Raw != in {
			t.Fatalf("%s: raw = %q", in, e.Raw)
		}
		if got := Summary(e); got == "" {
			t.Fatalf("%s: empty summary", in)
		}
		_ = want
	}
}

func TestParseReboot(t *testing.T) {
	e, err := Parse("@reboot")
	if err != nil || !e.Reboot {
		t.Fatalf("expected @reboot, got %v %v", e, err)
	}
	if _, err := Next(e, time.Now(), 1); err == nil {
		t.Fatal("Next(@reboot) should error")
	}
}

func TestParseErrors(t *testing.T) {
	for _, bad := range []string{
		"", "* * * *", "* * * * * *", "60 * * * *", "* 24 * * *",
		"*/0 * * * *", "* * 32 * *", "* * * 13 *", "* * * * 8",
		"a * * * *", "*/5 9-17 * MON-FRIextra",
	} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestParseStepsAndNames(t *testing.T) {
	e, err := Parse("*/15 0 * JAN-MAR SUN")
	if err != nil {
		t.Fatal(err)
	}
	if !e.Minute[0] || !e.Minute[15] || e.Minute[7] {
		t.Fatal("minutes wrong")
	}
	if !e.Month[1] || !e.Month[2] || !e.Month[3] || e.Month[4] {
		t.Fatal("months wrong")
	}
	if !e.Dow[0] || e.Dow[1] {
		t.Fatal("dow wrong")
	}
	// 7 normalizes to Sunday
	e2, _ := Parse("* * * * 7")
	if !e2.Dow[0] {
		t.Fatal("dow 7 should normalize to 0")
	}
}

func TestMatchesPosixOrRule(t *testing.T) {
	// dom=13, dow=Friday (5) — both restricted → OR semantics
	e, err := Parse("0 0 13 * 5")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-03-13 is a Friday the 13th
	if !e.Matches(time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Friday the 13th should match (OR rule)")
	}
	// 2026-04-13 is a Monday the 13th (dom matches, dow doesn't) — still matches
	if !e.Matches(time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Monday the 13th should match via dom (OR rule)")
	}
	// 2026-03-10 is a Tuesday, not the 13th — no match
	if e.Matches(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Tuesday the 10th should not match")
	}
	// dom unrestricted + dow restricted → AND semantics
	e2, _ := Parse("0 0 * * 1")
	if e2.Matches(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Tuesday should not match MON-only")
	}
	if !e2.Matches(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("Monday should match MON-only")
	}
}

func TestNextBasic(t *testing.T) {
	from := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	e, _ := Parse("*/15 * * * *")
	got, err := Next(e, from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		from.Add(15 * time.Minute),
		from.Add(30 * time.Minute),
		from.Add(45 * time.Minute),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNextRollovers(t *testing.T) {
	// year rollover
	e, _ := Parse("0 0 1 1 *")
	got, _ := Next(e, time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC), 1)
	if got[0].Year() != 2027 || got[0].Month() != time.January || got[0].Day() != 1 {
		t.Fatalf("year rollover wrong: %v", got[0])
	}
	// leap day
	e2, _ := Parse("0 0 29 2 *")
	g2, _ := Next(e2, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 1)
	if g2[0].Year() != 2028 || g2[0].Day() != 29 {
		t.Fatalf("leap day wrong: %v", g2[0])
	}
	// hour rollover
	e3, _ := Parse("0 9 * * *")
	g3, _ := Next(e3, time.Date(2026, 8, 30, 10, 30, 0, 0, time.UTC), 1)
	if g3[0].Day() != 31 || g3[0].Hour() != 9 {
		t.Fatalf("next-9am wrong: %v", g3[0])
	}
}

func TestDescribe(t *testing.T) {
	e, _ := Parse("*/5 9-17 * * 1-5")
	d := Describe(e)
	if d[0].Human != "every 5 minutes" {
		t.Fatalf("minute: %q", d[0].Human)
	}
	if d[1].Human != "at hour 09:00-17:00" {
		t.Fatalf("hour: %q", d[1].Human)
	}
	if d[4].Human != "on Monday-Friday" {
		t.Fatalf("dow: %q", d[4].Human)
	}
	e2, _ := Parse("30 14 1 * *")
	d2 := Describe(e2)
	if d2[0].Human != "at minute 30" || d2[1].Human != "at hour 14:00" {
		t.Fatalf("describe simple wrong: %+v", d2)
	}
}
