# Cron semantics

croni intentionally follows POSIX-style cron behavior rather than treating the expression as a simple five-field interval format.

## Day-of-month and day-of-week

When both fields are restricted, a matching time in either field is eligible to run. For example, 0 0 13 * 5 matches every Friday and every 13th day of the month.

## Day-of-week normalization

Both 0 and 7 represent Sunday. Named weekdays are normalized to the same internal representation.

## Calendar correctness

Next-run calculations must account for month lengths, leap years, field ranges and steps, named months and weekdays, and special presets such as @weekly.

These rules are part of croni's user-facing behavior, so changes to parser or calendar code should include regression tests around boundary dates.
