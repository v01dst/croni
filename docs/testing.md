# Cron Correctness Matrix

Calendar code is unusually easy to get subtly wrong, so tests should target boundaries.

## Parser cases

Cover wildcards, lists, ranges, steps, names, aliases, malformed values, out-of-range values, and whitespace variations.

## Calendar cases

Cover February in leap and non-leap years, month transitions, year transitions, midnight boundaries, DOM/DOW OR semantics, and Sunday represented by both 0 and 7.

## CLI contract

Verify exit codes and JSON output for both valid and invalid expressions. Keep expected JSON fixtures stable so shell integrations can depend on the contract.

## Regression workflow

Every calendar bug should become a minimal reproducible test before changing the implementation. Prefer a test that isolates the incorrect date calculation rather than relying only on a broad integration case.
