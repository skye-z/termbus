#!/bin/bash

# E2E Test Report Generator
# Usage: ./generate_report.sh
# Run from project root: cd /path/to/termbus && ./test/e2e/generate_report.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

REPORT_DIR="$SCRIPT_DIR/report"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$REPORT_DIR/report_$TIMESTAMP.html"

echo "Generating E2E Test Report..."
echo "Project Root: $PROJECT_ROOT"
echo "Report will be saved to: $REPORT_FILE"

mkdir -p "$REPORT_DIR"

echo "Running tests..."
go test -v -json ./test/e2e/ 2>&1 | tee "$REPORT_DIR/test_output.json"

# Generate coverage separately
go test -coverprofile="$REPORT_DIR/coverage.out" ./test/e2e/ 2>/dev/null || true

echo "Calculating coverage..."
go tool cover -func="$REPORT_DIR/coverage.out" > "$REPORT_DIR/coverage.txt" 2>/dev/null || echo "No coverage data"

# Parse test results from JSON output - get unique test results
PASSED=$(grep -c '"Action":"pass"' "$REPORT_DIR/test_output.json" 2>/dev/null | head -1 || echo "0")
FAILED=$(grep -c '"Action":"fail"' "$REPORT_DIR/test_output.json" 2>/dev/null | head -1 || echo "0")
SKIPPED=$(grep -c '"Action":"skip"' "$REPORT_DIR/test_output.json" 2>/dev/null | head -1 || echo "0")

# Build test results table - get the pass/fail action for each TestE2E test
echo "Building test results table..."

# Get unique test names and their status
TESTS=$(grep -E '"Action":"(pass|fail|skip)".*"Test":"TestE2E' "$REPORT_DIR/test_output.json" 2>/dev/null | \
    grep -oE '"Test":"TestE2E[^"]+' | sed 's/"Test":"//' | sort -u)

cat > "$REPORT_FILE" << 'EOFHEAD'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Termbus E2E Test Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 1400px; margin: 0 auto; padding: 20px; background: #f5f5f5; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 30px; }
        .header h1 { margin: 0; font-size: 2em; }
        .header .timestamp { opacity: 0.8; margin-top: 10px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .summary-card { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .summary-card h3 { margin: 0 0 10px 0; color: #666; font-size: 0.9em; text-transform: uppercase; }
        .summary-card .value { font-size: 2em; font-weight: bold; }
        .summary-card.pass .value { color: #10b981; }
        .summary-card.fail .value { color: #ef4444; }
        .summary-card.skip .value { color: #f59e0b; }
        .test-section { background: white; padding: 20px; border-radius: 10px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .test-section h2 { margin-top: 0; color: #333; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f5f5f5; font-weight: 600; }
        tr:hover { background: #f9f9f9; }
        .status-pass { color: #10b981; font-weight: bold; }
        .status-fail { color: #ef4444; font-weight: bold; }
        .status-skip { color: #f59e0b; font-weight: bold; }
        .footer { text-align: center; color: #999; margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; }
        .test-details { background: #f8f9fa; padding: 15px; margin-top: 10px; border-radius: 5px; font-family: monospace; font-size: 13px; }
        .test-details details { margin-top: 10px; }
        .test-details summary { cursor: pointer; color: #667eea; font-weight: bold; }
        .input-log { color: #2563eb; }
        .output-log { color: #059669; }
        .log-line { padding: 2px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Termbus E2E Test Report</h1>
        <div class="timestamp">Generated: TIMESTAMP_PLACEHOLDER</div>
    </div>
    <div class="summary">
        <div class="summary-card pass"><h3>Total Passed</h3><div class="value" id="passed">PASSED_PLACEHOLDER</div></div>
        <div class="summary-card fail"><h3>Total Failed</h3><div class="value" id="failed">FAILED_PLACEHOLDER</div></div>
        <div class="summary-card skip"><h3>Total Skipped</h3><div class="value" id="skipped">SKIPPED_PLACEHOLDER</div></div>
    </div>
    <div class="test-section">
        <h2>Test Results</h2>
        <table>
            <thead><tr><th>Test Name</th><th>Status</th><th>Time</th><th>Input/Output</th></tr></thead>
            <tbody>
EOFHEAD

# Replace placeholders
sed -i '' "s/TIMESTAMP_PLACEHOLDER/$(date)/g" "$REPORT_FILE"
sed -i '' "s/PASSED_PLACEHOLDER/$PASSED/g" "$REPORT_FILE"
sed -i '' "s/FAILED_PLACEHOLDER/$FAILED/g" "$REPORT_FILE"
sed -i '' "s/SKIPPED_PLACEHOLDER/$SKIPPED/g" "$REPORT_FILE"

# Function to get test details
get_test_details() {
    local test_name="$1"
    local details=""
    local in_details=0
    
    # Get INPUT/OUTPUT logs for this test
    while IFS= read -r line; do
        # Extract Output field from JSON line
        output=$(echo "$line" | grep -oE '"Output":"[^"]*INPUT:[^"]*"' | sed 's/"Output":"//;s/"$//')
        if [ -n "$output" ]; then
            details="$details<div class=\"log-line input-log\">$output</div>"
        fi
        
        output=$(echo "$line" | grep -oE '"Output":"[^"]*OUTPUT:[^"]*"' | sed 's/"Output":"//;s/"$//')
        if [ -n "$output" ]; then
            details="$details<div class=\"log-line output-log\">$output</div>"
        fi
    done < <(grep -E "\"Test\":\"$test_name\"" "$REPORT_DIR/test_output.json" 2>/dev/null)
    
    echo "$details"
}

# Add test rows
while IFS= read -r test_name; do
    # Get test result
    result_line=$(grep -E "\"Action\":\"(pass|fail|skip)\".*\"Test\":\"$test_name\"" "$REPORT_DIR/test_output.json" 2>/dev/null | head -1)
    
    if [ -z "$result_line" ]; then
        continue
    fi
    
    action=$(echo "$result_line" | grep -oE '"Action":"(pass|fail|skip)"' | sed 's/"Action":"//;s/"//')
    elapsed=$(echo "$result_line" | grep -oE '"Elapsed":[0-9.]+' | sed 's/"Elapsed"://')
    action_upper=$(echo "$action" | tr '[:lower:]' '[:upper:]')
    
    # Get test details (INPUT/OUTPUT logs)
    details=$(get_test_details "$test_name")
    
    if [ -n "$details" ]; then
        echo "                <tr><td>$test_name</td><td class=\"status-$action\">$action_upper</td><td>${elapsed}s</td><td><details><summary>View Details</summary><div class=\"test-details\">$details</div></details></td></tr>" >> "$REPORT_FILE"
    else
        echo "                <tr><td>$test_name</td><td class=\"status-$action\">$action_upper</td><td>${elapsed}s</td><td>-</td></tr>" >> "$REPORT_FILE"
    fi
done <<< "$TESTS"

cat >> "$REPORT_FILE" << 'EOFBODY'
            </tbody>
        </table>
    </div>
    <div class="footer"><p>Termbus E2E Test Suite</p></div>
</body>
</html>
EOFBODY

echo ""
echo "=========================================="
echo "Test Report Generated Successfully!"
echo "=========================================="
echo "Report Location: $REPORT_FILE"
echo "Coverage Data: $REPORT_DIR/coverage.txt"
echo ""

if [ "$1" = "--open" ] || [ "$1" = "-o" ]; then
    open "$REPORT_FILE"
fi
