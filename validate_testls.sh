#!/bin/bash
# Comprehensive validation script for gbm2 wt testls
# This script runs automated and manual tests using tmux

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test tracking
TESTS_PASSED=0
TESTS_FAILED=0
MANUAL_TESTS=0

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ INFO${NC}: $1"
}

# Automated Tests
print_header "AUTOMATED VALIDATION TESTS"

# TC-001: Build and Installation
print_info "TC-001: Verifying build and installation"
if command -v gbm2 &> /dev/null; then
    print_pass "gbm2 installed in PATH"
else
    print_fail "gbm2 not found in PATH"
    exit 1
fi

# TC-002: Help text
print_info "TC-002: Verifying help text"
help_output=$(gbm2 wt testls --help 2>&1)
if echo "$help_output" | grep -q "Displays a table of mock worktrees"; then
    print_pass "Help text displays correctly"
else
    print_fail "Help text missing"
fi

# TC-003: Flag parsing
print_info "TC-003: Verifying --delay flag"
if echo "$help_output" | grep -q "delay"; then
    print_pass "--delay flag recognized"
else
    print_fail "--delay flag not found"
fi

# TC-004: Default delay value
print_info "TC-004: Verifying default delay (1000ms)"
if echo "$help_output" | grep -q "1000"; then
    print_pass "Default delay is 1000ms"
else
    print_fail "Default delay not set to 1000ms"
fi

# Manual Tests via Tmux
print_header "SETTING UP TMUX SESSION FOR MANUAL TESTS"

# Create temp directory for tmux
TMPDIR=$(mktemp -d)
SESSION="testls_$(date +%s)"
cd "$TMPDIR"

# Create tmux session
tmux new-session -d -s "$SESSION" -c "$TMPDIR" 2>/dev/null || {
    print_fail "Could not create tmux session"
    exit 1
}

print_info "Created tmux session: $SESSION"
print_info "You can attach with: tmux attach -t $SESSION"
print_info "Test directory: $TMPDIR"

# Function to send command and capture output
run_test() {
    local test_name="$1"
    local command="$2"
    local delay="${3:-2}"
    
    echo -e "\n${YELLOW}Test: $test_name${NC}"
    echo "Command: $command"
    echo -e "${YELLOW}Send the command, then test as described.${NC}"
    echo -e "${YELLOW}When done, press Ctrl+C in the tmux session and then hit Enter here.${NC}"
    
    # Send command to tmux
    tmux send-keys -t "$SESSION" "$command" Enter
    
    # Wait for user to complete manual test
    read -p "Press Enter when test is complete: "
}

# Interactive test flow
echo -e "\n${BLUE}=== MANUAL TEST INSTRUCTIONS ===${NC}\n"
echo "A tmux session has been created. You can now run tests manually."
echo ""
echo "To view the session: tmux attach -t $SESSION"
echo "In the session:"
echo "  - Run the commands listed below"
echo "  - Observe the table behavior"
echo "  - After each test, report results (pass/fail)"
echo "  - Exit with Ctrl+C to quit testls, then continue to next test"
echo ""
read -p "Ready to begin manual tests? (y/n): " response
if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Skipping manual tests"
    exit 0
fi

# TC-M001: Launch and display
echo -e "\n${BLUE}=== TC-M001: Launch and Display ===${NC}"
echo "Expected: Table displays with 4 columns (Name, Branch, Kind, Git Status)"
echo "          All 8 worktrees visible, spinner animates in Git Status column"
echo ""
run_test "TC-M001: Launch and Display" "gbm2 wt testls --delay 2000"
read -p "Did table display correctly with spinner animation? (y/n): " tc001
[[ "$tc001" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M002: Navigation
echo -e "\n${BLUE}=== TC-M002: Navigation Up/Down ===${NC}"
echo "Expected: Cursor moves with arrow keys, wraps at boundaries"
echo ""
run_test "TC-M002: Navigation" "gbm2 wt testls --delay 2000"
read -p "Did navigation work smoothly (arrow keys)? (y/n): " tc002
[[ "$tc002" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M003: Help text
echo -e "\n${BLUE}=== TC-M003: Help Text (Dynamic) ===${NC}"
echo "Expected: Move between tracked (main, release/v1.0) and ad-hoc worktrees"
echo "          'p: push' option should disappear for tracked worktrees"
echo ""
run_test "TC-M003: Help Text" "gbm2 wt testls --delay 2000"
read -p "Did help text show/hide 'p: push' correctly? (y/n): " tc003
[[ "$tc003" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M004: Quit
echo -e "\n${BLUE}=== TC-M004: Exit (q/esc) ===${NC}"
echo "Expected: Press 'q' or 'esc' to quit cleanly"
echo ""
run_test "TC-M004: Quit" "gbm2 wt testls --delay 2000"
read -p "Did quit work (q/esc) without errors? (y/n): " tc004
[[ "$tc004" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M005: Selection
echo -e "\n${BLUE}=== TC-M005: Selection Output (space/enter) ===${NC}"
echo "Expected: Navigate to row 2 (feature/auth), press space/enter"
echo "          Should output: /tmp/feature-auth"
echo ""
run_test "TC-M005: Selection" "gbm2 wt testls --delay 2000"
read -p "Did selection output correct path? (y/n): " tc005
[[ "$tc005" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M006: Operations
echo -e "\n${BLUE}=== TC-M006: Operations (l, p, d) ===${NC}"
echo "Expected: 'l' shows 'Would pull', 'p' on ad-hoc shows 'Would push'"
echo "          'p' on tracked shows 'Cannot push', 'd' shows 'Would delete'"
echo ""
run_test "TC-M006: Operations" "gbm2 wt testls --delay 2000"
read -p "Did operations display correct messages? (y/n): " tc006
[[ "$tc006" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M007: Async loading
echo -e "\n${BLUE}=== TC-M007: Async Loading Spinner ===${NC}"
echo "Expected: Spinner animates for ~2 seconds, then shows status value"
echo "          Each worktree eventually shows: ✓, ↑ N, ↓ N, ↕, or ?"
echo ""
run_test "TC-M007: Async Loading" "gbm2 wt testls --delay 2000"
read -p "Did spinner animate then show values correctly? (y/n): " tc007
[[ "$tc007" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# TC-M008: Stress test
echo -e "\n${BLUE}=== TC-M008: Stress Test (Rapid Navigation) ===${NC}"
echo "Expected: Rapidly press up/down arrows, application stays responsive"
echo ""
run_test "TC-M008: Stress Test" "gbm2 wt testls --delay 1000"
read -p "Did application handle rapid navigation smoothly? (y/n): " tc008
[[ "$tc008" =~ ^[Yy]$ ]] && ((MANUAL_TESTS++))

# Summary
print_header "TEST SUMMARY"

echo "Automated Tests:"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"
echo ""
echo "Manual Tests Passed: $MANUAL_TESTS / 8"
echo ""

if [ $TESTS_FAILED -eq 0 ] && [ $MANUAL_TESTS -eq 8 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ SOME TESTS FAILED - REVIEW RESULTS${NC}"
    exit 1
fi
