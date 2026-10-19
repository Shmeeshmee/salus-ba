---
description: Implement one package of a Salus plan with the package-runner agent. Usage: /run-package docs/<feature> 1.2
---

Use the package-runner subagent for this package: $ARGUMENTS

Pass it: the plan folder and package number from the arguments, the commit prefix from the plan's
`RUN.tsv` (`# prefix:` line), the tickets file `.claude/runs/<prefix>/<package>.tickets.md` if it exists
(otherwise say Jira was not exported and it should work from the plan docs), and the report path
`.claude/runs/<prefix>/<package>.report.md`.

When it returns, check that its commit exists with `git log -1`, then show me its report.
