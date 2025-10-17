#!/bin/bash

# Script to test all book examples compile correctly
# This ensures all example code is working without fail

set -e  # Exit on first error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_DIR="$ROOT_DIR/book2"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

total_examples=0
passed_examples=0
failed_examples=0
skipped_examples=0

declare -a failed_list=()
declare -a skipped_list=()

echo "=================================================="
echo "Testing All Book Examples"
echo "=================================================="
echo ""

# Function to test a single example
test_example() {
    local example_dir=$1
    local example_name=$(basename "$example_dir")
    local chapter_name=$(basename "$(dirname "$(dirname "$example_dir")")")
    local part_name=$(basename "$(dirname "$(dirname "$(dirname "$example_dir")")")")

    total_examples=$((total_examples + 1))

    printf "${BLUE}[%d]${NC} Testing: %s/%s/%s ... " "$total_examples" "$part_name" "$chapter_name" "$example_name"

    # Check if main.go exists
    if [ ! -f "$example_dir/main.go" ]; then
        printf "${YELLOW}SKIP${NC} (no main.go)\n"
        skipped_examples=$((skipped_examples + 1))
        skipped_list+=("$part_name/$chapter_name/$example_name")
        return
    fi

    # Try to build the example
    cd "$example_dir"

    # Build without running (some examples require network or specific setup)
    if output=$(go build -o /tmp/test-example 2>&1); then
        printf "${GREEN}PASS${NC}\n"
        passed_examples=$((passed_examples + 1))
        rm -f /tmp/test-example
    else
        printf "${RED}FAIL${NC}\n"
        failed_examples=$((failed_examples + 1))
        failed_list+=("$part_name/$chapter_name/$example_name")
        echo "$output" | head -10
        echo ""
    fi

    cd "$ROOT_DIR"
}

echo "Scanning for examples in Part 1 and Part 2..."
echo ""

# Find all example directories in part1-foundations
if [ -d "$BOOK_DIR/part1-foundations" ]; then
    echo "${BLUE}Testing Part 1: Foundations${NC}"
    echo "-----------------------------------"

    for chapter_dir in "$BOOK_DIR/part1-foundations"/chapter*; do
        if [ -d "$chapter_dir/examples" ]; then
            for example_dir in "$chapter_dir/examples"/*; do
                if [ -d "$example_dir" ]; then
                    test_example "$example_dir"
                fi
            done
        fi
    done
    echo ""
fi

# Find all example directories in part2-api-approaches
if [ -d "$BOOK_DIR/part2-api-approaches" ]; then
    echo ""
    echo "${BLUE}Testing Part 2: API Approaches${NC}"
    echo "-----------------------------------"

    for chapter_dir in "$BOOK_DIR/part2-api-approaches"/chapter*; do
        if [ -d "$chapter_dir/examples" ]; then
            for example_dir in "$chapter_dir/examples"/*; do
                if [ -d "$example_dir" ]; then
                    test_example "$example_dir"
                fi
            done
        fi
    done
    echo ""
fi

# Print summary
echo ""
echo "=================================================="
echo "Test Summary"
echo "=================================================="
echo ""
echo "Total Examples:   $total_examples"
echo -e "${GREEN}Passed:${NC}          $passed_examples"
echo -e "${RED}Failed:${NC}          $failed_examples"
echo -e "${YELLOW}Skipped:${NC}         $skipped_examples"
echo ""

# Show failed examples if any
if [ $failed_examples -gt 0 ]; then
    echo "${RED}Failed Examples:${NC}"
    for item in "${failed_list[@]}"; do
        echo "  ❌ $item"
    done
    echo ""
fi

# Show skipped examples if any
if [ $skipped_examples -gt 0 ]; then
    echo "${YELLOW}Skipped Examples:${NC}"
    for item in "${skipped_list[@]}"; do
        echo "  ⊘ $item"
    done
    echo ""
fi

# Calculate success rate
if [ $total_examples -gt 0 ]; then
    success_rate=$((passed_examples * 100 / total_examples))
    echo "Success Rate: ${success_rate}%"
    echo ""
fi

# Exit with error if any tests failed
if [ $failed_examples -gt 0 ]; then
    echo "${RED}❌ Some examples failed to compile!${NC}"
    exit 1
else
    echo "${GREEN}✅ All examples compiled successfully!${NC}"
    exit 0
fi
