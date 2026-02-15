---
name: execute
description: Implement, validate, and close existing Linear tickets using staff-engineer and qa-engineer agents. Runs the full execution loop with parallel dispatch, QA validation, retry tracking, and human escalation. Use when tickets already exist and need to be implemented.
---

# Project Execution

Implement, validate, and close existing Linear tickets. Tickets may come from a prior `/scope`
invocation or may already exist in Linear.

## Subagents Used

| Agent | Purpose |
|-------|---------|
| `staff-engineer` | Executes a single Linear ticket: implement, test, validate |
| `qa-engineer` | Validates completed work against ticket acceptance criteria |
| `project-manager` | Only if discovered work needs to be planned into new tickets |

## Phase 1: Identify Ready Tickets

1. Determine which tickets to execute:
   - If the user specified ticket IDs, use those.
   - Otherwise, query Linear: `list_issues` filtered by project and status = "Todo".
   - Filter to tickets with no unresolved `blockedBy` dependencies.

2. Group independent tickets for parallel dispatch.

3. Present the execution plan to the user: which tickets will be dispatched, in what order,
   and what parallelism is possible. Proceed when the user confirms, or adjust based on
   feedback.

## Phase 2: Implement

1. Spawn `staff-engineer` subagents in parallel — one per ready ticket:
   ```
   # In a single message, spawn multiple Task calls:
   Task(subagent_type="staff-engineer", prompt="Execute Linear ticket <TICKET-ID>. Context: <brief description of parent goal>")
   Task(subagent_type="staff-engineer", prompt="Execute Linear ticket <TICKET-ID>. Context: <brief description of parent goal>")
   ```

2. Each staff-engineer will:
   - Read the ticket via `get_issue`
   - Move the ticket to "In Progress"
   - Explore the codebase and implement the changes
   - Self-validate: build, test, lint
   - Comment on the ticket with a completion summary
   - Move the ticket to "Done"
   - Return results: success + summary, or failure + reason

3. Collect all results. For any staff-engineer that reports failure, record it for triage.

## Phase 3: Validate

1. For each ticket moved to "Done" by a staff-engineer, spawn a `qa-engineer` subagent:
   ```
   # In a single message, spawn QA for all completed tickets:
   Task(subagent_type="qa-engineer", prompt="Validate Linear ticket <TICKET-ID>. The ticket contains validation criteria in its description. Build the binary and run behavioral tests against the criteria. Report pass/fail with evidence.")
   Task(subagent_type="qa-engineer", prompt="Validate Linear ticket <TICKET-ID>. ...")
   ```

2. Each QA agent will:
   - Read the ticket's validation criteria from Linear
   - Build the binary from current source
   - Execute behavioral tests (functional via tmux, visual via VHS)
   - Comment on the Linear ticket with QA results
   - Return: PASS (all criteria met) or FAIL (which criteria failed and why)

3. **Exception:** Pure exploration/research tickets that produce no code changes do not require
   QA validation. The orchestrator may skip QA for these and close them directly based on the
   staff-engineer's completion comment.

## Phase 4: Triage

Process each result:

### Staff-Engineer Failure (couldn't complete the work)

The ticket remains in "In Progress" or was not moved to "Done":
- Comment on the ticket: `EXECUTION FAIL (attempt N/2): <reason from staff-engineer report>`
- Increment the retry counter for this ticket
- Move the ticket back to "Todo" via `update_issue(id, state="Todo")`
- On the next loop iteration, a fresh staff-engineer will attempt it with the failure context
  available in the ticket comments

### QA Pass

- Ticket stays in "Done" — no further action needed
- Check if this completion unblocks dependent tickets (their `blockedBy` resolved)
- Add newly unblocked tickets to the ready queue for the next execution phase

### QA Fail

- Comment on the ticket: `QA FAIL (attempt N/2): <summary of failures from QA report>`
- Move the ticket back to "Todo" via `update_issue(id, state="Todo")`
- Increment the retry counter for this ticket

### Retry Limit Exceeded (2 total failures of any type)

When a ticket accumulates 2 failures (execution failures and QA failures both count):

1. **Flag the ticket:**
   - Comment: `ESCALATED: This ticket has failed 2 attempts and requires human intervention.`
   - Include a summary of both failure attempts with root cause analysis from the staff-engineer
     and QA reports
   - Do NOT re-dispatch this ticket

2. **Hold dependent tickets:**
   - Identify all tickets with `blockedBy` references to this ticket (direct and transitive)
   - Comment on each dependent: `HELD: Blocked by <TICKET-ID> which has been escalated for human review.`
   - Do NOT dispatch any ticket in this dependency chain

3. **Report to the user:**
   - Which ticket was escalated and why
   - Summary of both failure attempts (what was tried, what failed, what the QA agent found)
   - Which dependent tickets are held
   - Ask the user how to proceed: fix the issue manually, provide guidance for a retry,
     re-scope the ticket, or abandon the work

## Phase 5: Loop

After triage:

1. Check for newly unblocked tickets (their `blockedBy` dependencies are all "Done" and
   QA-validated).
2. Check for tickets moved back to "Todo" that are within retry limits.
3. If there are ready tickets → return to Phase 2.
4. If all tickets are "Done" and QA-validated → report completion to the user with a full
   summary of all work completed.
5. If all remaining tickets are escalated or held → report the situation and wait for the user.

## Retry Tracking

The orchestrator maintains retry counts during the session:

- Track a map of `ticket-ID → attempt_count` (starts at 0, incremented on any failure)
- Both execution failures and QA failures increment the same counter
- Maximum attempts: **2** (after 2 failures, escalate)
- Each failure is also recorded as a comment on the Linear ticket for audit trail, so context
  persists across session boundaries if needed

## Rules

- **Linear is the source of truth.** All ticket state lives in Linear. Query it before acting
  on stale assumptions.
- **Dispatch via Task tool.** Use `subagent_type` matching the agent name:
  `staff-engineer`, `qa-engineer`, `project-manager`.
- **Maximize parallelism.** Independent tickets get parallel staff-engineers. Independent QA
  validations run in parallel. Use multiple Task calls in a single message.
- **Never implement directly.** The orchestrator dispatches and coordinates. It does not write
  code or run tests itself.
- **Report progress to the user** between phases. Summarize what was dispatched, what completed,
  what failed, and what's next. Keep it concise.
- **Discovered work flows back to the PM.** If a staff-engineer reports discovered work during
  execution, spawn the project-manager to create new tickets for it. Do not create ad-hoc
  issues directly.
- **QA is mandatory for code changes.** Every ticket that produces code changes and is marked
  "Done" by a staff-engineer must go through QA validation before it is considered complete.
- **Respect the dependency graph.** Never dispatch a ticket whose `blockedBy` dependencies
  include tickets that are not yet "Done" and QA-validated.
- **Two-failure escalation is strict.** Do not retry a third time. Do not work around a
  poisoned ticket. The human must be in the loop at that point.
