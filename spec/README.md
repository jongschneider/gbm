# Specification Directory

Reference documentation, best practices, and implementation guidance.

## 🚀 Start Here

### [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) ⭐
**Condensed Phase 2 roadmap** — 10 stories, bullet points, citations to specs + code.  
Read this if you want: Quick overview of what to build (15 min read)

---

## Detailed Specifications

### [BUBBLETEA.md](./BUBBLETEA.md)
Comprehensive best practices for building Terminal User Interfaces (TUIs) with Bubble Tea. Covers event loop performance, debugging techniques, model architecture, layout patterns, testing strategies, and references to proven implementations.

**Keywords:** `bubbletea`, `tui`, `terminal-ui`, `event-loop`, `message-routing`, `model-tree`, `teatest`, `layout`, `debugging`, `vhs`, `demo-recording`

### [REVIEW_SUMMARY.md](./REVIEW_SUMMARY.md)
Executive summary of the TUI package review against Bubble Tea best practices. Highlights critical issues (event loop blocking, silent errors, missing tests), code quality metrics, and recommended fixes with risk assessment.

**Key Findings:**
- ⚠️ **Critical**: Synchronous service calls block event loop (JIRA/Git fetches in Update())
- ⚠️ **High**: No async command pattern - uses blocking Eval instead
- ⚠️ **High**: No root model/Navigator for multi-screen flows
- 🔴 **Missing**: Integration tests with teatest, error handling, custom field support

**Read this first** to understand the review scope and key issues.

**Keywords:** `code-review`, `tui-architecture`, `best-practices-audit`, `event-loop`, `async`, `testing`

### [TUI_IMPROVEMENTS.md](./TUI_IMPROVEMENTS.md)
Detailed specification for TUI package improvements with concrete code examples and implementation guidance. Organized by issue area with before/after code, impact analysis, and success criteria.

**Contents:**
1. Event Loop Performance - Convert async operations to Bubble Tea commands
2. Message Ordering - Use tea.Sequence() for dependent operations
3. Hierarchical Model Tree - Create Navigator for model composition
4. Layout Management - Add responsive layout helpers
5. Testing & Demos - Teatest suite and VHS recording scripts
6. Error Handling - User-visible error messages and recovery
7. Debug Message Dumping - Logging infrastructure for troubleshooting
8. Custom Field Storage - Generic field state management
9. Documentation - Architecture diagrams and best practices

**Implementation Priority:**
- Phase 1 (Critical): Event loop, message ordering, error handling
- Phase 2 (Important): Testing, Navigator, custom fields
- Phase 3 (Enhancement): Layouts, debugging, documentation

**Keywords:** `implementation-spec`, `event-loop`, `async-patterns`, `testing`, `architecture`, `code-examples`
