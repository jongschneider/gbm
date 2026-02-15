# VHS Tape Syntax Reference

Complete reference for VHS tape file commands. See the [main skill](SKILL.md) for workflow guidance.

## Configuration Commands

```tape
# Output settings
Output output.gif              # Primary output (GIF, MP4, or WebM)
Output output.txt              # Text capture (optional)

# Terminal settings
Set Shell "bash"               # Shell to use
Set FontSize 14                # Font size in pixels
Set Width 1280                 # Terminal width in pixels
Set Height 720                 # Terminal height in pixels
Set TypingSpeed 50ms           # Delay between keystrokes

# Window settings
Set Theme "Dracula"            # Color theme
Set Padding 20                 # Padding around terminal
```

## Input Commands

### Typing

```tape
Type "some text"               # Type text character by character
Type@100ms "fast typing"       # Override typing speed for this line
```

### Keys

```tape
Enter                          # Press Enter
Tab                            # Press Tab
Space                          # Press Space
Backspace                      # Press Backspace
Delete                         # Press Delete
Escape                         # Press Escape
```

### Arrow keys

```tape
Up                             # Arrow up
Down                           # Arrow down
Left                           # Arrow left
Right                          # Arrow right
```

### Control sequences

```tape
Ctrl+C                         # Interrupt
Ctrl+D                         # EOF
Ctrl+L                         # Clear screen
Ctrl+U                         # Clear line
Ctrl+A                         # Move to beginning of line
Ctrl+E                         # Move to end of line
Ctrl+W                         # Delete word backward
```

## Flow Control

### Timing

```tape
Sleep 500ms                    # Wait for specified duration
Sleep 1s                       # Supports ms, s time units
Sleep 2.5s                     # Fractional seconds
```

### Screenshots

```tape
Screenshot path/to/file.png    # Capture current terminal state to PNG
```

**Important:** VHS tokenizes screenshot paths at numbers followed by non-alphanumeric characters. Use number suffixes (`initial01.png`) not prefixes (`01-initial.png`).

### Recording control

```tape
Hide                           # Stop recording output to GIF/video
Show                           # Resume recording output
```

## Environment Setup

### Environment variables

```tape
Env MY_VAR "value"             # Set environment variable
Env HOME "/tmp/test-home"      # Override HOME for isolated testing
Env TERM "xterm-256color"      # Set terminal type
```

### Requirements

```tape
Require my-binary              # Fail immediately if command not available
Require git                    # Guard against missing dependencies
```

## Common Configuration Presets

### Minimal test (fast, small)

```tape
Set Shell "bash"
Set Width 800
Set Height 400
Set TypingSpeed 50ms
```

### Full-screen demo

```tape
Set Shell "bash"
Set Width 1920
Set Height 1080
Set FontSize 16
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 75ms
```

### CI-friendly (no visual output needed)

```tape
Set Shell "bash"
Set Width 800
Set Height 400
Set TypingSpeed 0
```

Setting `TypingSpeed 0` makes typing instant, which speeds up execution when only screenshots matter.
