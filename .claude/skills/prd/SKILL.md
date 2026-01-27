---
name: prd
description: Create structured Product Requirements Documents suitable for RFC review by Principal Engineers, Designers, and Product Owners.
---

# PRD Creation Skill

Create structured Product Requirements Documents suitable for RFC review by Principal Engineers, Designers, and Product Owners.

## Core Principle

**The PRD describes WHAT to build and WHY, not HOW or in WHAT ORDER.**

## Workflow

When this skill is invoked:

1. User provides feature context or problem statement
2. Ask clarifying questions across key categories (see below)
3. Explore codebase for existing patterns and constraints
4. Generate markdown PRD to `specs/prd-<feature-name>.md`

## Question Categories

Before writing the PRD, gather information across these areas:

### Problem & Motivation
- What problem are we solving?
- What is the cost of inaction?
- Why now?

### Users & Stakeholders
- Who are the primary users?
- Who are secondary stakeholders?
- How does this affect existing users?

### End State & Success
- What does "done" look like?
- How will we measure success?
- What are the acceptance criteria?

### Scope & Boundaries
- What is explicitly out of scope?
- What work is deferred to future iterations?
- What are the MVP requirements vs nice-to-haves?

### Constraints & Requirements
- Performance requirements?
- Security considerations?
- Compatibility requirements?
- Accessibility needs?

### Risks & Dependencies
- What are the technical risks?
- External service dependencies?
- What could block this work?

## Output Location

PRDs are written to: `specs/prd-<feature-name>.md`

Use kebab-case for the feature name (e.g., `prd-user-authentication.md`).

## Output Structure

Generate markdown with these sections:

```markdown
# <Feature Name> PRD

## Problem Statement
Clear description of the problem being solved and why it matters.

## Proposed Solution

### Overview
High-level description of the solution approach.

### UX Flows
User journey and interaction patterns.

### Design Considerations
Visual and interaction design notes.

## End State
- [ ] Checkbox completion criteria
- [ ] Each item is a verifiable outcome
- [ ] Written as "User can..." or "System does..."

## Success Metrics

### Quantitative
- Measurable metrics (latency, throughput, error rates)

### Qualitative
- User satisfaction, developer experience

## Acceptance Criteria
- [ ] Feature-specific verification checkboxes
- [ ] Testable statements

## Technical Context

### Existing Patterns
Reference existing code patterns to follow.

### Key Files
List files that will be modified or referenced.

### Dependencies
External libraries or services required.

### Data Model Changes
Database schema or data structure modifications.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Description | Low/Med/High | Low/Med/High | How to address |

## Alternatives Considered

### <Alternative 1>
Description and why it was rejected.

### <Alternative 2>
Description and why it was rejected.

## Non-Goals
Explicit list of what this PRD does NOT cover:
- Item 1
- Item 2

## Interface Specifications

### CLI (if applicable)
Command structure and flags.

### API (if applicable)
Endpoints, request/response formats.

### UI (if applicable)
Screen layouts, component specifications.

## Documentation Requirements
What documentation needs to be created or updated.

## Open Questions

| Question | Owner | Status | Resolution |
|----------|-------|--------|------------|
| Unresolved question | Who decides | Open/Resolved | Answer |

## Appendix

### Glossary
Domain-specific terms.

### References
Links to related documents, issues, or discussions.
```

## Key Principles

1. **Problem clarity before solution details** - Ensure the problem is well-understood before proposing solutions
2. **Define end states, not processes** - Focus on outcomes, not implementation steps
3. **Provide technical context for autonomy** - Give enough detail for engineers to work independently
4. **Establish scope boundaries** - Prevent scope creep with explicit non-goals
5. **Demonstrate rigor** - Show alternatives considered and risks analyzed

## Integration with prd-task

After a PRD is reviewed and approved, use the `prd-task` skill to convert it into executable JSON format for autonomous task completion.

The Tasks section with `[category]` tags and Verification subsections are specifically designed to be parsed by `prd-task`.
