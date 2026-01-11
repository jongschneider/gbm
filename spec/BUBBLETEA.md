# Bubble Tea Best Practices

This specification documents best practices and patterns for building Terminal User Interfaces (TUIs) with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## 1. Keep the Event Loop Fast

The event loop processes messages sequentially through `Update()` and `View()` methods. Both must execute quickly to maintain responsiveness.

**Pattern:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Don't do expensive work here:
        // time.Sleep(time.Minute)  ❌
        
        // Instead, return a command:
        return m, func() tea.Msg {  // ✓
            time.Sleep(time.Minute)
            return someMsg{}
        }
    }
    return m, nil
}
```

**Rationale:** Blocking `Update()` or `View()` causes user input lag and unresponsive UI.

## 2. Debug with Message Dumps

When debugging, dump all messages to a log file and tail it in another terminal.

**Pattern:**
```go
type model struct {
    dump io.Writer
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.dump != nil {
        spew.Fdump(m.dump, msg)  // Pretty-print each message
    }
    // ... rest of Update logic
    return m, nil
}

func main() {
    var dump *os.File
    if _, ok := os.LookupEnv("DEBUG"); ok {
        dump, _ = os.OpenFile("messages.log", os.O_CREATE|os.O_WRONLY, 0o644)
    }
    p := tea.NewProgram(model{dump: dump})
    p.Run()
}
```

**Activation:** `DEBUG=1 go run . & tail -f messages.log`

## 3. Use Pointer Receivers Judiciously

Bubble Tea models in documentation use **value receivers** (following Elm architecture). However, pointer receivers are useful for helper methods and maintaining state modifications.

**Pattern - Main Update() with value receiver:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Changes persisted via returned model
    m.content = "updated"
    return m, nil  // ✓
}
```

**Pattern - Helper methods with pointer receiver:**
```go
func (m *model) updateDimensions(w, h int) {
    m.width = w
    m.height = h
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.updateDimensions(msg.Width, msg.Height)
    }
    return m, nil
}
```

**⚠️ Anti-pattern:** Never modify model state outside the event loop:
```go
func (m *model) Init() tea.Cmd {
    go func() {
        m.content = "init"  // ❌ Race condition!
    }()
    return nil
}
```

## 4. Messages Are Not Ordered (When Sent Concurrently)

Commands execute concurrently in goroutines. Their resulting messages may arrive in any order.

**Problem:**
```go
// Messages 0-9 sent concurrently: may arrive as [0,1,9,8,5,6,4,2,3,7]
for i := 0; i < 10; i++ {
    p.Send(myMsg(i))
}
```

**Solutions:**

1. **Update directly in `Update()` if order is critical:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        m.ordered = append(m.ordered, nextItem())  // In-order
        return m, nil
}
```

2. **Use `tea.Sequence()` to run commands sequentially:**
```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        return m, tea.Sequence(doThis, thenThat)  // Sequential
}
```

3. **Redesign to not require order:** Embrace concurrency.

## 5. Build a Hierarchical Model Tree

For non-trivial TUIs, organize models hierarchically with a root model that routes messages and composes views.

**Architecture:**
```
Root Model (message router, compositor)
├── Header Model
├── Content Model (current visible model)
└── Footer Model
```

**Pattern:**
```go
type rootModel struct {
    header  headerModel
    content tea.Model  // Current/active model
    footer  footerModel
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmds := []tea.Cmd{}
    
    // 1. Global keys (quit, help)
    if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
        return m, tea.Quit
    }
    
    // 2. Route to current model
    var cmd tea.Cmd
    m.content, cmd = m.content.Update(msg)
    cmds = append(cmds, cmd)
    
    // 3. Broadcast to all (window resize, etc.)
    // ...
    
    return m, tea.Batch(cmds...)
}

func (m rootModel) View() string {
    return lipgloss.JoinVertical(
        lipgloss.Top,
        m.header.View(),
        m.content.View(),
        m.footer.View(),
    )
}
```

**Patterns:**
- Maintain current model (with stack for navigation history)
- Cache dynamically-created models
- Route global keys → current model → broadcast messages

## 6. Use `lipgloss` for Layout Arithmetic

Never hardcode layout dimensions. Use `lipgloss.Height()` and `lipgloss.Width()` to calculate available space.

**❌ Brittle (hardcoded):**
```go
func (m model) View() string {
    header := lipgloss.NewStyle().Render("header")
    footer := lipgloss.NewStyle().Render("footer")
    content := lipgloss.NewStyle().Height(m.height - 2).Render("content")
    // Adding border to header breaks: content height now wrong!
}
```

**✓ Adaptive:**
```go
func (m model) View() string {
    header := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder(), false, false, true, false).
        Render("header")
    footer := lipgloss.NewStyle().Render("footer")
    content := lipgloss.NewStyle().
        Height(m.height - lipgloss.Height(header) - lipgloss.Height(footer)).
        Render("content")
    
    return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}
```

**Pattern:** Always calculate dimensions from rendered widgets, not magic numbers.

## 7. Recover Your Terminal After Panics

Panics in `Update()` or `View()` are recovered by Bubble Tea, but panics in commands leave the terminal in a broken state (raw mode not disabled, no cursor).

**Recovery:**
```bash
reset
```

**Mitigation:** Wrap commands with error handling:
```go
func someCommand() tea.Msg {
    defer func() {
        if r := recover(); r != nil {
            // Log error, return error message
        }
    }()
    // ... potentially panicking code
}
```

**Note:** [Open issue](https://github.com/charmbracelet/bubbletea/issues/234) for Bubble Tea to auto-recover.

## 8. Test with `teatest`

Use Charm's `teatest` framework for end-to-end testing of TUI behavior.

**Pattern:**
```go
import "github.com/charmbracelet/x/exp/teatest"

func TestQuit(t *testing.T) {
    m := model{}
    tm := teatest.NewTestModel(t, m)
    
    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
        return strings.Contains(string(b), "Running")
    }, teatest.WithDuration(time.Second*5))
    
    // Send input
    tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
    tm.Type("y")
    
    // Verify finish
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

**Features:**
- Emulate user keypresses: `tm.Send()`, `tm.Type()`
- Check output: `tm.Output()`
- Golden files for regression testing (requires regeneration on content changes)

**Status:** Part of Charm's experimental repo; no backwards compatibility guarantee.

## 9. Record Demos with VHS

Use [VHS](https://github.com/charmbracelet/vhs) to record declarative terminal scripts into animated GIFs and screenshots.

**Pattern:**
```
Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Framerate 24

Hide
Type `go run main.go` Enter
Sleep 1s
Show

Type "some-input"
Sleep 0.5s
Screenshot demo-screenshot.png
```

**Usage:**
```bash
vhs demo.tape
```

**Benefits:**
- Reproducible demos
- Documentation + testing in one script
- Can be part of CI/CD pipeline

## 10. Reference Real Projects

Study implementations:
- [PUG](../../deps/pug) - Complex TUI for Terraform ([GitHub](https://github.com/leg100/pug))
  - Table widget with selections, sorting, filtering
  - Split model with adjustable panes
  - Navigator with model caching
  - Integration tests

## Key Takeaways

| Principle | Benefit |
|-----------|---------|
| Fast event loop | Responsive UI, no lag |
| Message dumps | Easier debugging |
| Live reload | Faster development feedback |
| Model tree | Scalable architecture |
| Layout arithmetic | Resilient layouts |
| `teatest` + VHS | Testable + reproducible demos |
| Study projects | Learn patterns and anti-patterns |

## Resources

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - GitHub
- [Bubble Tea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest)
- [VHS](https://github.com/charmbracelet/vhs)
- [Original Blog Post](https://leg100.github.io/en/posts/building-bubbletea-programs/)
