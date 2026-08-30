# Changelog

## v0.1.0 — 2026-08-30

- `explain` — plain-English per-field breakdown + summary line
- `next` — upcoming runs with humanized deltas, `--from`, `-n`
- `validate` — crontab linter with per-line diagnostics
- `--json` output for all subcommands
- Special strings: `@hourly @daily @midnight @noon @weekly @monthly @yearly @annually @reboot`
- Names, ranges, steps, lists; dow `7` → Sunday normalization
- POSIX dom/dow OR semantics; correct calendar math (leap years, month lengths)
- `--color always|auto|never`, `FORCE_COLOR` / `NO_COLOR`
- Zero dependencies
