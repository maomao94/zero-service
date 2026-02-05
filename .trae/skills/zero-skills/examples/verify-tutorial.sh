#!/bin/bash

# Demo script to verify AI ecosystem tutorial configurations

set -e  # Exit on error

DEMO_DIR="/tmp/zero-skills-demo-$$"
RESULTS_FILE="$DEMO_DIR/verification-results.txt"

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "Zero-Skills AI Ecosystem Tutorial Verification"
echo "================================================"
echo ""

# Create temporary directory
mkdir -p "$DEMO_DIR"
echo "✓ Created test directory: $DEMO_DIR"
echo ""

# Log result function
log_result() {
    local test_name=$1
    local status=$2
    local message=$3

    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
    fi
    echo "  $message"
    echo "$test_name: $status - $message" >> "$RESULTS_FILE"
    echo ""
}

# Test 1: GitHub Copilot Configuration
test_github_copilot() {
    echo "=== Test 1: GitHub Copilot Configuration ==="
    local test_dir="$DEMO_DIR/copilot-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize git repository
    git init -q

    # Add ai-context as submodule
    if git submodule add -q https://github.com/zeromicro/ai-context.git .github/ai-context 2>/dev/null; then
        # Create symlink
        mkdir -p .github
        ln -s ai-context/00-instructions.md .github/copilot-instructions.md

        # Verify file exists
        if [ -L ".github/copilot-instructions.md" ] && [ -e ".github/copilot-instructions.md" ]; then
            # Verify content
            if grep -q "go-zero" .github/copilot-instructions.md; then
                log_result "GitHub Copilot Configuration" "PASS" "Submodule added, symlink created, content verified"
                return 0
            else
                log_result "GitHub Copilot Configuration" "FAIL" "File content does not contain go-zero related content"
                return 1
            fi
        else
            log_result "GitHub Copilot Configuration" "FAIL" "Symlink creation failed or file does not exist"
            return 1
        fi
    else
        log_result "GitHub Copilot Configuration" "FAIL" "Submodule add failed"
        return 1
    fi
}

# Test 2: Cursor Configuration
test_cursor() {
    echo "=== Test 2: Cursor Configuration ==="
    local test_dir="$DEMO_DIR/cursor-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize git repository
    git init -q

    # Add ai-context as submodule
    if git submodule add -q https://github.com/zeromicro/ai-context.git .cursorrules 2>/dev/null; then
        # Verify directory and file exist
        if [ -d ".cursorrules" ] && [ -f ".cursorrules/00-instructions.md" ]; then
            # Verify content
            if grep -q "go-zero" .cursorrules/00-instructions.md; then
                # Count .md files
                md_count=$(find .cursorrules -name "*.md" -type f | wc -l)
                log_result "Cursor Configuration" "PASS" "Submodule added, found $md_count .md files"
                return 0
            else
                log_result "Cursor Configuration" "FAIL" "File content does not contain go-zero related content"
                return 1
            fi
        else
            log_result "Cursor Configuration" "FAIL" ".cursorrules directory or file does not exist"
            return 1
        fi
    else
        log_result "Cursor Configuration" "FAIL" "Submodule add failed"
        return 1
    fi
}

# Test 3: Windsurf Configuration
test_windsurf() {
    echo "=== Test 3: Windsurf Configuration ==="
    local test_dir="$DEMO_DIR/windsurf-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize git repository
    git init -q

    # Add ai-context as submodule
    if git submodule add -q https://github.com/zeromicro/ai-context.git .windsurfrules 2>/dev/null; then
        # Verify directory and file exist
        if [ -d ".windsurfrules" ] && [ -f ".windsurfrules/00-instructions.md" ]; then
            # Verify content
            if grep -q "go-zero" .windsurfrules/00-instructions.md; then
                # Count .md files
                md_count=$(find .windsurfrules -name "*.md" -type f | wc -l)
                log_result "Windsurf Configuration" "PASS" "Submodule added, found $md_count .md files"
                return 0
            else
                log_result "Windsurf Configuration" "FAIL" "File content does not contain go-zero related content"
                return 1
            fi
        else
            log_result "Windsurf Configuration" "FAIL" ".windsurfrules directory or file does not exist"
            return 1
        fi
    else
        log_result "Windsurf Configuration" "FAIL" "Submodule add failed"
        return 1
    fi
}

# Test 4: Submodule Update
test_submodule_update() {
    echo "=== Test 4: Submodule Update ==="
    local test_dir="$DEMO_DIR/update-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize git repository and add submodule
    git init -q
    git submodule add -q https://github.com/zeromicro/ai-context.git .github/ai-context 2>/dev/null

    # Record initial commit hash
    cd .github/ai-context
    initial_commit=$(git rev-parse HEAD)
    cd ../..

    # Try to update
    if git submodule update --remote .github/ai-context 2>/dev/null; then
        cd .github/ai-context
        updated_commit=$(git rev-parse HEAD)
        cd ../..

        log_result "Submodule Update" "PASS" "Update successful (commit: ${updated_commit:0:8})"
        return 0
    else
        log_result "Submodule Update" "FAIL" "Submodule update failed"
        return 1
    fi
}

# Test 5: Verify ai-context Content Structure
test_content_structure() {
    echo "=== Test 5: Verify ai-context Content Structure ==="
    local test_dir="$DEMO_DIR/content-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize and clone
    git init -q
    git submodule add -q https://github.com/zeromicro/ai-context.git .ai-context 2>/dev/null

    # Verify key content
    local required_sections=(
        "Decision Tree"
        "File Priority"
        "Patterns"
        "zero-skills"
    )

    local missing_sections=()
    for section in "${required_sections[@]}"; do
        if ! grep -q "$section" .ai-context/00-instructions.md; then
            missing_sections+=("$section")
        fi
    done

    if [ ${#missing_sections[@]} -eq 0 ]; then
        log_result "Content Structure" "PASS" "All required sections exist"
        return 0
    else
        log_result "Content Structure" "FAIL" "Missing sections: ${missing_sections[*]}"
        return 1
    fi
}

# Test 6: Verify zero-skills References
test_zero_skills_references() {
    echo "=== Test 6: Verify zero-skills References ==="
    local test_dir="$DEMO_DIR/reference-test"
    mkdir -p "$test_dir"
    cd "$test_dir"

    # Initialize and clone
    git init -q
    git submodule add -q https://github.com/zeromicro/ai-context.git .ai-context 2>/dev/null

    # Verify zero-skills links
    local required_links=(
        "rest-api-patterns.md"
        "rpc-patterns.md"
        "database-patterns.md"
        "resilience-patterns.md"
    )

    local missing_links=()
    for link in "${required_links[@]}"; do
        if ! grep -q "$link" .ai-context/00-instructions.md; then
            missing_links+=("$link")
        fi
    done

    if [ ${#missing_links[@]} -eq 0 ]; then
        log_result "zero-skills References" "PASS" "All pattern document references exist"
        return 0
    else
        log_result "zero-skills References" "FAIL" "Missing references: ${missing_links[*]}"
        return 1
    fi
}

# Run all tests
main() {
    echo "Running tests..."
    echo ""

    local total=0
    local passed=0

    # Run tests
    test_github_copilot && ((passed++)) || true
    ((total++))

    test_cursor && ((passed++)) || true
    ((total++))

    test_windsurf && ((passed++)) || true
    ((total++))

    test_submodule_update && ((passed++)) || true
    ((total++))

    test_content_structure && ((passed++)) || true
    ((total++))

    test_zero_skills_references && ((passed++)) || true
    ((total++))

    # Output summary
    echo "================================================"
    echo "Test Summary"
    echo "================================================"
    echo "Total Tests: $total"
    echo -e "Passed: ${GREEN}$passed${NC}"
    echo -e "Failed: ${RED}$((total - passed))${NC}"
    echo ""

    if [ $passed -eq $total ]; then
        echo -e "${GREEN}✓ All tests passed! Tutorial verified successfully!${NC}"
    else
        echo -e "${YELLOW}⚠ Some tests failed, please check the configuration${NC}"
    fi

    echo ""
    echo "Detailed results saved to: $RESULTS_FILE"
    echo "Test directory: $DEMO_DIR"
    echo ""

    # Auto-cleanup temporary directory
    echo "Cleaning up temporary test directory..."
    if rm -rf "$DEMO_DIR" 2>/dev/null; then
        echo -e "${GREEN}✓ Temporary directory cleaned up${NC}"
    else
        echo -e "${YELLOW}⚠ Auto-cleanup failed, please delete manually:${NC} rm -rf $DEMO_DIR"
    fi
    echo ""
}

# Execute main function
main
