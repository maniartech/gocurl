#!/bin/bash

# Unified Test Launcher for GoCurl Book Examples
# This script can run tests using either bash or Go, with various options

set -e  # Exit on first error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_DIR="$ROOT_DIR/book2"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Default settings
TEST_METHOD="auto"  # auto, bash, go
RUN_EXAMPLES=false
VERBOSE=false
CHAPTER_FILTER=""
PART_FILTER=""

# Usage information
usage() {
    cat << EOF
${BOLD}GoCurl Book Examples Test Launcher${NC}

${BOLD}USAGE:${NC}
    $0 [OPTIONS]

${BOLD}OPTIONS:${NC}
    -m, --method <auto|bash|go>  Test method (default: auto)
                                  auto: Use Go if available, else bash
                                  bash: Use bash-only implementation
                                  go:   Use Go test program

    -r, --run                     Actually run examples (not just compile)
                                  Warning: Makes network calls

    -v, --verbose                 Verbose output (show compilation details)

    -p, --part <1|2>              Test only specific part (1 or 2)

    -c, --chapter <N>             Test only specific chapter number

    -h, --help                    Show this help message

${BOLD}EXAMPLES:${NC}
    $0                            # Test all examples (auto method)
    $0 -m go                      # Use Go test program
    $0 -m bash                    # Use bash-only testing
    $0 -p 1                       # Test only Part 1
    $0 -p 2 -c 6                  # Test only Chapter 6 of Part 2
    $0 -r                         # Compile AND run examples
    $0 -v                         # Verbose output

${BOLD}DESCRIPTION:${NC}
    This script tests all book examples in Part 1 and Part 2 to ensure
    they compile correctly. It can use either a Go-based test program
    (faster, better output) or pure bash (no dependencies).

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--method)
            TEST_METHOD="$2"
            shift 2
            ;;
        -r|--run)
            RUN_EXAMPLES=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -p|--part)
            PART_FILTER="$2"
            shift 2
            ;;
        -c|--chapter)
            CHAPTER_FILTER="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

# Validate method
if [[ "$TEST_METHOD" != "auto" && "$TEST_METHOD" != "bash" && "$TEST_METHOD" != "go" ]]; then
    echo -e "${RED}Invalid method: $TEST_METHOD${NC}"
    echo "Must be one of: auto, bash, go"
    exit 1
fi

# Determine which method to use
determine_method() {
    if [[ "$TEST_METHOD" == "auto" ]]; then
        if command -v go &> /dev/null; then
            echo "go"
        else
            echo "bash"
        fi
    else
        echo "$TEST_METHOD"
    fi
}

ACTUAL_METHOD=$(determine_method)

# Validate method availability
if [[ "$ACTUAL_METHOD" == "go" ]] && ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not available but 'go' method was requested${NC}"
    exit 1
fi

echo -e "${BOLD}=================================================="
echo "GoCurl Book Examples Test Launcher"
echo -e "==================================================${NC}"
echo ""
echo -e "${CYAN}Test Method:${NC} $ACTUAL_METHOD"
echo -e "${CYAN}Run Examples:${NC} $RUN_EXAMPLES"
echo -e "${CYAN}Verbose:${NC} $VERBOSE"
[[ -n "$PART_FILTER" ]] && echo -e "${CYAN}Part Filter:${NC} $PART_FILTER"
[[ -n "$CHAPTER_FILTER" ]] && echo -e "${CYAN}Chapter Filter:${NC} $CHAPTER_FILTER"
echo ""

# ============================================================================
# GO-BASED TESTING
# ============================================================================

run_go_tests() {
    echo -e "${BLUE}Using Go test program...${NC}"
    echo ""

    cd "$SCRIPT_DIR"

    # Build arguments for Go program
    GO_ARGS=""
    [[ -n "$PART_FILTER" ]] && GO_ARGS="$GO_ARGS -part=$PART_FILTER"
    [[ -n "$CHAPTER_FILTER" ]] && GO_ARGS="$GO_ARGS -chapter=$CHAPTER_FILTER"
    [[ "$RUN_EXAMPLES" == true ]] && GO_ARGS="$GO_ARGS -run"
    [[ "$VERBOSE" == true ]] && GO_ARGS="$GO_ARGS -verbose"

    # Run the Go test program
    if go run test-examples.go $GO_ARGS; then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# BASH-BASED TESTING
# ============================================================================

run_bash_tests() {
    echo -e "${BLUE}Using bash test implementation...${NC}"
    echo ""

    local total_examples=0
    local passed_examples=0
    local failed_examples=0
    local skipped_examples=0

    declare -a failed_list=()
    declare -a skipped_list=()

    # Function to test a single example
    test_example() {
        local example_dir=$1
        local example_name=$(basename "$example_dir")
        local chapter_name=$(basename "$(dirname "$(dirname "$example_dir")")")
        local part_name=$(basename "$(dirname "$(dirname "$(dirname "$example_dir")")")")

        # Apply filters
        if [[ -n "$PART_FILTER" ]]; then
            if [[ "$part_name" == "part1-foundations" && "$PART_FILTER" != "1" ]]; then
                return
            fi
            if [[ "$part_name" == "part2-api-approaches" && "$PART_FILTER" != "2" ]]; then
                return
            fi
        fi

        if [[ -n "$CHAPTER_FILTER" ]]; then
            chapter_num=$(echo "$chapter_name" | grep -o '[0-9]\+' | head -1)
            if [[ "$chapter_num" != "$CHAPTER_FILTER" ]]; then
                return
            fi
        fi

        total_examples=$((total_examples + 1))

        local display_path="$part_name/$chapter_name/$example_name"
        printf "${BLUE}[%d]${NC} Testing: %s ... " "$total_examples" "$display_path"

        # Check if main.go exists
        if [ ! -f "$example_dir/main.go" ]; then
            printf "${YELLOW}SKIP${NC} (no main.go)\n"
            skipped_examples=$((skipped_examples + 1))
            skipped_list+=("$display_path")
            return
        fi

        # Try to build the example
        cd "$example_dir"

        if [[ "$RUN_EXAMPLES" == true ]]; then
            # Actually run the example
            if output=$(timeout 10s go run main.go 2>&1); then
                printf "${GREEN}PASS${NC}\n"
                [[ "$VERBOSE" == true ]] && echo "$output" | head -5
                passed_examples=$((passed_examples + 1))
            else
                printf "${RED}FAIL${NC}\n"
                failed_examples=$((failed_examples + 1))
                failed_list+=("$display_path")
                echo "$output" | head -10
                echo ""
            fi
        else
            # Just compile
            if output=$(go build -o /tmp/test-example 2>&1); then
                printf "${GREEN}PASS${NC}\n"
                passed_examples=$((passed_examples + 1))
                rm -f /tmp/test-example
            else
                printf "${RED}FAIL${NC}\n"
                failed_examples=$((failed_examples + 1))
                failed_list+=("$display_path")
                [[ "$VERBOSE" == true ]] && echo "$output" | head -10
                echo ""
            fi
        fi

        cd "$ROOT_DIR"
    }

    # Find and test Part 1 examples
    if [[ -z "$PART_FILTER" || "$PART_FILTER" == "1" ]]; then
        if [ -d "$BOOK_DIR/part1-foundations" ]; then
            echo -e "${BOLD}Testing Part 1: Foundations${NC}"
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
    fi

    # Find and test Part 2 examples
    if [[ -z "$PART_FILTER" || "$PART_FILTER" == "2" ]]; then
        if [ -d "$BOOK_DIR/part2-api-approaches" ]; then
            echo -e "${BOLD}Testing Part 2: API Approaches${NC}"
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
        echo -e "${RED}Failed Examples:${NC}"
        for item in "${failed_list[@]}"; do
            echo "  ❌ $item"
        done
        echo ""
    fi

    # Show skipped examples if any
    if [ $skipped_examples -gt 0 ]; then
        echo -e "${YELLOW}Skipped Examples:${NC}"
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
        echo -e "${RED}❌ Some examples failed!${NC}"
        return 1
    else
        echo -e "${GREEN}✅ All examples passed!${NC}"
        return 0
    fi
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

if [[ "$ACTUAL_METHOD" == "go" ]]; then
    run_go_tests
    exit_code=$?
else
    run_bash_tests
    exit_code=$?
fi

exit $exit_code
