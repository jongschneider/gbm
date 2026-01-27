---
name: prd-task
description: Convert markdown Product Requirements Documents into executable JSON format for autonomous task completion.
---

# PRD Task Skill

Convert markdown Product Requirements Documents into executable JSON format for autonomous task completion.

## Workflow

When this skill is invoked:

1. User provides path to a markdown PRD file (typically `specs/prd-<feature-name>.md`)
2. Read the markdown PRD file
3. Extract tasks with their verification criteria
4. Create `specs/state/<prd-name>/` directory structure
5. Move original markdown PRD to state folder
6. Generate JSON representation for task execution
7. Create empty progress tracking file

## State Directory Structure

```
specs/state/<prd-name>/
├── prd.md       # Original markdown (moved from source location)
├── prd.json     # Executable JSON format
└── progress.txt # Progress tracking file (initially empty)
```

## Input Requirements

Markdown PRDs must contain:

- **End State section**: Checkboxes defining desired outcomes
- **Tasks section**: Subsections with category tags like `[functional]`, `[api]`, `[db]`
- **Verification subsection**: Concrete testing steps for each task
- **Context section**: Patterns, key files, and non-goals

### Example PRD Structure

```markdown
# Feature Name PRD

## End State
- [ ] Users can do X
- [ ] System handles Y

## Tasks

### [functional] User Registration
**Verification:**
- POST /api/auth/register with valid email/password
- Verify 201 response with user object
- Attempt duplicate email, verify 409 response

### [api] Authentication Endpoint
**Verification:**
- POST /api/auth/login with valid credentials
- Verify 200 response with token

## Context

### Patterns
- API routes: src/routes/
- Validation: src/lib/validate.ts

### Key Files
- src/db/schema.ts
- src/routes/index.ts

### Non-Goals
- OAuth/social login
- Password reset flow
```

## Output JSON Schema

See `references/prd-schema.json` for the complete schema.

Each task object includes:
- **category**: Organizational grouping (functional, ui, api, security, testing, db, etc.)
- **description**: Concise statement of what completion means
- **steps**: Array of verification steps describing how to test functionality
- **passes**: Boolean flag (initially false, set true when all steps verified)

## Critical Principle

**Steps are verification, not implementation.**

Steps describe how to TEST and VALIDATE that a feature works correctly, not how to implement it. This guides the agent's validation approach rather than prescribing implementation details.

## Conversion Guidelines

Tasks should be:
- Small and logically focused
- Completable in single commits
- Broken into multiple tasks if sections feel oversized
- Prioritized for quality over speed

## Post-Conversion

After successful conversion, report:
1. Location of the new state directory
2. Total task count
3. Category breakdown
4. Excluded non-goals (if any)
5. Instruction to begin autonomous task completion
