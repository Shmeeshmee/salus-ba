# 1. Priorities: fast, then simple, then elegant

Status: accepted, 2026-10-19

## Context

A telemedicine platform has real latency budgets (booking under load, video join, live transcription), and a
small team has to read and change all of it. Rules that pull against each other need an order, and
"performance" is the value most often claimed without evidence.

## Decision

The order is performance, then simplicity, then elegance. Performance counts only when measured: a benchmark
(`go test -bench`), a profile or a load test, with the numbers recorded in the package's `PROGRESS.md`. Without
a measurement, the simplest code that passes the tests wins. GDPR is never traded for any of the three.

## Consequences

- A change that adds complexity for speed carries its measurement in the same commit, before and after.
- Reviewers reject "faster" without numbers.
- Elegance breaks ties only; it never justifies an abstraction on its own.
- CLAUDE.md "Top rules" rule 1 states this order.
