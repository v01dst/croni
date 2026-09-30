# Cron Grammar

croni accepts five-field cron expressions plus supported aliases.

## Fields

    minute hour day-of-month month day-of-week

Each field accepts wildcards, lists, ranges, and steps where the field's domain allows them.

## Names

Month names and weekday names are normalized case-insensitively. Day-of-week 7 is normalized to Sunday (0).

## Special expressions

Aliases such as @hourly, @daily, @weekly, @monthly, and @yearly expand to their standard five-field equivalents. @reboot is a semantic special and does not represent a recurring calendar schedule.

## DOM/DOW semantics

When both day-of-month and day-of-week are restricted, POSIX cron semantics use OR behavior: a timestamp matches when either restricted field matches.

Calendar calculations must account for month lengths and leap years rather than approximating months as fixed durations.
