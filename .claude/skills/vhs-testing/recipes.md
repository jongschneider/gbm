# TUI Testing Recipes

Reusable patterns for common TUI testing scenarios. Adapt the commands and binary paths to your project.

## Single-Select List

Test a component that presents options and allows selecting one.

```tape
Output test-output/choose-test.gif
Set Shell "bash"
Set Width 800
Set Height 400
Set TypingSpeed 50ms

Require ./bin/my-tui

# Start list component
Type "./bin/my-tui choose 'Apple' 'Banana' 'Cherry' 'Date'"
Enter
Sleep 300ms
Screenshot screenshots/chooseinitial01.png

# Navigate down twice
Down
Sleep 100ms
Screenshot screenshots/choosefirstdown02.png

Down
Sleep 100ms
Screenshot screenshots/chooseseconddown03.png

# Select (should output "Cherry")
Enter
Sleep 200ms
Screenshot screenshots/chooseselected04.png
```

**Verify:**
- Initial state shows first item highlighted
- After each `Down`, highlight moves to next item
- Final screenshot shows selected value was output

## Filterable List

Test a component with a text input that filters visible options.

```tape
Output test-output/filter-test.gif
Set Shell "bash"
Set Width 800
Set Height 400

Require ./bin/my-tui

# Start filter with several options
Type "./bin/my-tui filter 'apple' 'apricot' 'banana' 'blackberry' 'cherry'"
Enter
Sleep 300ms
Screenshot screenshots/filterinitial01.png

# Type filter text
Type "ap"
Sleep 200ms
Screenshot screenshots/filterfiltered02.png

# Clear and try different filter
Ctrl+U
Sleep 100ms
Type "berry"
Sleep 200ms
Screenshot screenshots/filterberry03.png

# Select filtered item
Enter
Sleep 200ms
Screenshot screenshots/filterselected04.png
```

**Verify:**
- Initial state shows all options
- After typing "ap", only matching items visible
- After typing "berry", only matching items visible
- Selection outputs the correct filtered value

## Confirmation Prompt

Test a yes/no confirmation dialog.

```tape
Output test-output/confirm-test.gif
Set Shell "bash"
Set Width 800
Set Height 300

Require ./bin/my-tui

# Test accepting default (No)
Type "./bin/my-tui confirm 'Proceed with operation?'"
Enter
Sleep 300ms
Screenshot screenshots/confirmprompt01.png

Enter
Sleep 200ms
Screenshot screenshots/confirmdefaultno02.png

# Test selecting Yes
Type "./bin/my-tui confirm 'Delete all files?'"
Enter
Sleep 300ms
Screenshot screenshots/confirmnewprompt03.png

Left
Sleep 100ms
Screenshot screenshots/confirmyesselected04.png

Enter
Sleep 200ms
Screenshot screenshots/confirmconfirmed05.png
```

**Verify:**
- Prompt text displayed correctly
- Default selection is "No"
- Arrow keys switch between Yes/No
- Correct value output after confirmation

## Multi-Select (Checkbox) List

Test a component where multiple items can be toggled.

```tape
Output test-output/multiselect-test.gif
Set Shell "bash"
Set Width 800
Set Height 400

Require ./bin/my-tui

# Start multi-select
Type "./bin/my-tui choose --multi 'Red' 'Green' 'Blue' 'Yellow'"
Enter
Sleep 300ms
Screenshot screenshots/multiinitial01.png

# Toggle first item
Space
Sleep 100ms
Screenshot screenshots/multifirst02.png

# Navigate and toggle another
Down
Down
Space
Sleep 100ms
Screenshot screenshots/multithird03.png

# Confirm selection
Enter
Sleep 200ms
Screenshot screenshots/multiresult04.png
```

**Verify:**
- All items shown unchecked initially
- Space toggles checkbox on current item
- Multiple items can be checked simultaneously
- Enter outputs all checked values

## Text Input

Test a text input field with placeholder and validation.

```tape
Output test-output/input-test.gif
Set Shell "bash"
Set Width 800
Set Height 300

Require ./bin/my-tui

# Start text input
Type "./bin/my-tui input --placeholder 'Enter your name'"
Enter
Sleep 300ms
Screenshot screenshots/inputplaceholder01.png

# Type a value
Type "Jane Doe"
Sleep 200ms
Screenshot screenshots/inputfilled02.png

# Submit
Enter
Sleep 200ms
Screenshot screenshots/inputsubmitted03.png
```

**Verify:**
- Placeholder text visible before typing
- Typed text appears in the input field
- Submit outputs the entered value

## Interactive Command Selection

Test a command browser or launcher that lists commands and executes the selection.

```tape
Output test-output/interactive-test.gif
Set Shell "bash"
Set Width 1000
Set Height 600

Require ./bin/my-tui

# Launch interactive mode
Type "./bin/my-tui -i"
Enter
Sleep 500ms
Screenshot screenshots/interactivecmdlist01.png

# Filter commands
Type "hello"
Sleep 300ms
Screenshot screenshots/interactivefiltered02.png

# Select and execute
Enter
Sleep 500ms
Screenshot screenshots/interactiveexec03.png
```

## Tab Navigation

Test a component with multiple tabs or panels.

```tape
Output test-output/tabs-test.gif
Set Shell "bash"
Set Width 1000
Set Height 600

Require ./bin/my-tui

Type "./bin/my-tui tabs"
Enter
Sleep 300ms
Screenshot screenshots/tabsinitial01.png

# Switch tabs
Tab
Sleep 200ms
Screenshot screenshots/tabssecond02.png

Tab
Sleep 200ms
Screenshot screenshots/tabsthird03.png

# Navigate within a tab
Down
Down
Sleep 100ms
Screenshot screenshots/tabsnavigation04.png
```

**Verify:**
- Tab indicator shows which tab is active
- Content changes when switching tabs
- Navigation within a tab works independently

## Error State

Test how the TUI handles and displays errors.

```tape
Output test-output/error-test.gif
Set Shell "bash"
Set Width 800
Set Height 400

Require ./bin/my-tui

# Trigger an error condition
Type "./bin/my-tui choose"
Enter
Sleep 300ms
Screenshot screenshots/errornoargs01.png
```

**Verify:**
- Error message is displayed clearly
- Error text uses appropriate colors (typically red)
- Exit code is non-zero (check text output if captured)

## Long List with Scrolling

Test that scrolling and viewport work correctly with many items.

```tape
Output test-output/scroll-test.gif
Set Shell "bash"
Set Width 800
Set Height 400

Require ./bin/my-tui

# Start with many items (more than fit on screen)
Type "./bin/my-tui choose 'Item 1' 'Item 2' 'Item 3' 'Item 4' 'Item 5' 'Item 6' 'Item 7' 'Item 8' 'Item 9' 'Item 10' 'Item 11' 'Item 12' 'Item 13' 'Item 14' 'Item 15'"
Enter
Sleep 300ms
Screenshot screenshots/scrolltop01.png

# Navigate past the visible area
Down
Down
Down
Down
Down
Down
Down
Down
Down
Down
Sleep 100ms
Screenshot screenshots/scrollmiddle02.png

# Continue to the end
Down
Down
Down
Down
Sleep 100ms
Screenshot screenshots/scrollbottom03.png
```

**Verify:**
- Initial viewport shows first N items
- Scrolling reveals items beyond the viewport
- Cursor/highlight tracks correctly through scroll
- Scroll indicator (if any) updates position
