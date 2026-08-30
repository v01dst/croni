package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/v01dst/croni/internal/cron"
)

const version = "0.1.0"

type colorMode int

const (
	colorAuto colorMode = iota
	colorAlways
	colorNever
)

var (
	cReset, cBold, cDim, cField, cGood, cBad, cAccent string
	useColor                                          = false
)

func setupColor(m colorMode) {
	force := os.Getenv("FORCE_COLOR") == "1"
	noColor := os.Getenv("NO_COLOR") != ""
	switch {
	case m == colorNever || noColor:
		useColor = false
	case m == colorAlways || force:
		useColor = true
	default:
		fi, _ := os.Stdout.Stat()
		useColor = (fi.Mode() & os.ModeCharDevice) != 0
	}
	if useColor {
		cReset = "\x1b[0m"
		cBold = "\x1b[1m"
		cDim = "\x1b[2m"
		cField = "\x1b[38;5;75m"
		cGood = "\x1b[38;5;114m"
		cBad = "\x1b[38;5;203m"
		cAccent = "\x1b[38;5;220m"
	}
}

func paint(s, c string) string {
	if !useColor {
		return s
	}
	return c + s + cReset
}

func usage() {
	fmt.Printf(`%scroni %s%s — cron expressions, explained.

%sUSAGE%s
  croni explain <expr>         explain a cron expression in plain English
  croni next <expr> [-n N]     show the next N executions (default 5)
  croni validate <crontab>     validate every line of a crontab file

%sOPTIONS%s
  --json            machine-readable output
  --from <RFC3339>  compute "next" from this time (default: now)
  -n <N>            number of upcoming runs for "next" (default 5)
  --color <mode>    always | auto | never        (env: FORCE_COLOR, NO_COLOR)
  -h, --help        show this help
  -V, --version     print version

%sEXAMPLES%s
  croni explain "*/5 9-17 * * 1-5"
  croni next "0 4 * * *" -n 3
  croni next "@weekly" --json
  croni validate /etc/crontab

Exit codes: 0 ok · 1 usage error · 2 invalid expression
`, paint("", ""), version, cReset, paint("USAGE", cBold), cReset, paint("OPTIONS", cBold), cReset, paint("EXAMPLES", cBold), cReset)
}

func failUsage(msg string) int {
	fmt.Fprintln(os.Stderr, paint("error:", cBad), msg)
	fmt.Fprintln(os.Stderr, "try: croni --help")
	return 1
}

type explainOut struct {
	Expression string           `json:"expression"`
	Summary    string           `json:"summary"`
	Fields     []cron.FieldDesc `json:"fields"`
}

type nextOut struct {
	Expression string   `json:"expression"`
	Summary    string   `json:"summary"`
	From       string   `json:"from"`
	Runs       []string `json:"runs"`
	Humanized  []string `json:"humanized"`
}

type validateOut struct {
	File  string         `json:"file"`
	Lines []validateLine `json:"lines"`
	Valid int            `json:"valid"`
	Total int            `json:"total"`
}

type validateLine struct {
	Line    int    `json:"line"`
	Raw     string `json:"raw"`
	OK      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

func cmdExplain(args []string, asJSON bool) int {
	if len(args) == 0 {
		return failUsage("explain needs an expression")
	}
	e, err := cron.Parse(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, paint("invalid expression:", cBad), err)
		return 2
	}
	if asJSON {
		b, _ := json.MarshalIndent(explainOut{
			Expression: e.Raw,
			Summary:    cron.Summary(e),
			Fields:     cron.Describe(e),
		}, "", "  ")
		fmt.Println(string(b))
		return 0
	}
	fmt.Println(paint("expression", cBold), e.Raw)
	fmt.Println()
	for _, f := range cron.Describe(e) {
		fmt.Printf("  %s  %s\n", paint(fmt.Sprintf("%-13s", f.Name), cField), f.Human)
	}
	fmt.Println()
	fmt.Println(paint("➜", cAccent), cron.Summary(e))
	return 0
}

func cmdNext(args []string, asJSON bool, n int, from time.Time) int {
	if len(args) == 0 {
		return failUsage("next needs an expression")
	}
	e, err := cron.Parse(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, paint("invalid expression:", cBad), err)
		return 2
	}
	runs, err := cron.Next(e, from, n)
	if err != nil {
		fmt.Fprintln(os.Stderr, paint("error:", cBad), err)
		return 2
	}
	if asJSON {
		o := nextOut{Expression: e.Raw, Summary: cron.Summary(e), From: from.Format(time.RFC3339)}
		for _, r := range runs {
			o.Runs = append(o.Runs, r.Format(time.RFC3339))
			o.Humanized = append(o.Humanized, cron.HumanizeDelta(r.Sub(from)))
		}
		b, _ := json.MarshalIndent(o, "", "  ")
		fmt.Println(string(b))
		return 0
	}
	fmt.Println(paint("schedule", cBold), e.Raw, paint("·", cDim), cron.Summary(e))
	fmt.Println(paint("from", cBold), from.Format("Mon 2006-01-02 15:04"))
	fmt.Println()
	for _, r := range runs {
		fmt.Printf("  %s  %s\n", paint(r.Format("2006-01-02 15:04"), cAccent), paint(cron.HumanizeDelta(r.Sub(from)), cDim))
	}
	return 0
}

func cmdValidate(path string, asJSON bool) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, paint("error:", cBad), err)
		return 1
	}
	out := validateOut{File: path}
	status := 0
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		vl := validateLine{Line: i + 1, Raw: line}
		parts := strings.SplitN(line, " ", 6)
		if len(parts) < 6 {
			vl.OK, vl.Error = false, fmt.Sprintf("expected 6+ fields (5 schedule + command), got %d", len(parts))
			status = 2
		} else {
			if _, err := cron.Parse(strings.Join(parts[:5], " ")); err != nil {
				vl.OK, vl.Error = false, err.Error()
				status = 2
			} else {
				vl.OK = true
				e, _ := cron.Parse(strings.Join(parts[:5], " "))
				vl.Summary = cron.Summary(e)
			}
		}
		out.Lines = append(out.Lines, vl)
		if vl.OK {
			out.Valid++
		}
		out.Total++
	}
	if asJSON {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return status
	}
	for _, l := range out.Lines {
		mark := paint("✅", cGood)
		detail := l.Summary
		if !l.OK {
			mark = paint("❌", cBad)
			detail = l.Error
		}
		fmt.Printf("  %s %s%s%s  %s\n", mark, cDim, fmt.Sprintf("L%03d", l.Line), cReset, detail)
	}
	fmt.Printf("\n%s\n", paint(fmt.Sprintf("%d/%d lines valid", out.Valid, out.Total), cBold))
	return status
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	var asJSON bool
	var color = colorAuto
	var n = 5
	var from = time.Now()

	var args []string
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		switch a {
		case "--json":
			asJSON = true
		case "--color":
			i++
			if i >= len(argv) {
				return failUsage("--color needs a mode")
			}
			switch argv[i] {
			case "always":
				color = colorAlways
			case "never":
				color = colorNever
			case "auto":
			default:
				return failUsage("invalid --color mode " + argv[i])
			}
		case "-n":
			i++
			if i >= len(argv) {
				return failUsage("-n needs a number")
			}
			v, err := strconv.Atoi(argv[i])
			if err != nil || v < 1 || v > 100 {
				return failUsage("-n must be 1..100")
			}
			n = v
		case "--from":
			i++
			if i >= len(argv) {
				return failUsage("--from needs an RFC3339 time")
			}
			t, err := time.Parse(time.RFC3339, argv[i])
			if err != nil {
				return failUsage("bad --from time: " + err.Error())
			}
			from = t
		case "-h", "--help":
			usage()
			return 0
		case "-V", "--version":
			fmt.Println("croni version", version)
			return 0
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag " + a)
			}
			args = append(args, a)
		}
	}

	setupColor(color)

	if len(args) == 0 {
		usage()
		return 1
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "explain":
		return cmdExplain(rest, asJSON)
	case "next":
		return cmdNext(rest, asJSON, n, from)
	case "validate":
		if len(rest) == 0 {
			return failUsage("validate needs a crontab file")
		}
		return cmdValidate(rest[0], asJSON)
	default:
		// bare expression → explain (syntactic sugar)
		return cmdExplain(args, asJSON)
	}
}
