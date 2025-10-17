#!/bin/bash
# Example 6: Output Formatting
# Demonstrates custom output formatting with -w flag

echo "📊 Example 6: Output Formatting"
echo "================================"
echo ""

# Example 1: Basic output variables
echo "1️⃣  HTTP Status Code"
echo "Command: gocurl -w '%{http_code}\\n' -o /dev/null -s https://httpbin.org/get"
gocurl -w "%{http_code}\n" -o /dev/null -s https://httpbin.org/get
echo ""

# Example 2: Multiple output variables
echo "2️⃣  Multiple Output Variables"
echo "Command: gocurl -w 'Status: %{http_code}\\nTime: %{time_total}s\\nSize: %{size_download} bytes\\n' -o /dev/null -s https://httpbin.org/get"
gocurl -w "Status: %{http_code}\nTime: %{time_total}s\nSize: %{size_download} bytes\n" -o /dev/null -s https://httpbin.org/get
echo ""

# Example 3: Formatted report
echo "3️⃣  Formatted Performance Report"
echo "Command: gocurl -w '╔════════════════════════════╗\\n║   Performance Report      ║\\n╠════════════════════════════╣\\n║ Status:     %{http_code}           ║\\n║ Time:       %{time_total}s       ║\\n║ Download:   %{size_download} bytes     ║\\n╚════════════════════════════╝\\n' -o /dev/null -s https://httpbin.org/get"
gocurl -w "╔════════════════════════════╗\n║   Performance Report      ║\n╠════════════════════════════╣\n║ Status:     %{http_code}           ║\n║ Time:       %{time_total}s       ║\n║ Download:   %{size_download} bytes     ║\n╚════════════════════════════╝\n" -o /dev/null -s https://httpbin.org/get
echo ""

# Example 4: JSON format output
echo "4️⃣  JSON Format Output"
echo "Command: gocurl -w '{\"status\":%{http_code},\"time\":%{time_total},\"size\":%{size_download}}\\n' -o /dev/null -s https://httpbin.org/get"
gocurl -w '{"status":%{http_code},"time":%{time_total},"size":%{size_download}}\n' -o /dev/null -s https://httpbin.org/get
echo ""

# Example 5: CSV format output
echo "5️⃣  CSV Format Output"
cat > format-csv.sh << 'EOF'
#!/bin/bash
echo "url,status,time,size"
urls=(
    "https://httpbin.org/get"
    "https://httpbin.org/json"
    "https://httpbin.org/uuid"
)

for url in "${urls[@]}"; do
    gocurl -w "$url,%{http_code},%{time_total},%{size_download}\n" -o /dev/null -s "$url"
done
EOF

chmod +x format-csv.sh
./format-csv.sh
rm format-csv.sh
echo ""

# Example 6: Monitoring script with formatted output
echo "6️⃣  API Monitoring Script"
cat > monitor-api.sh << 'EOF'
#!/bin/bash
URL="${1:-https://httpbin.org/get}"
INTERVAL="${2:-5}"
COUNT="${3:-3}"

echo "Monitoring $URL every ${INTERVAL}s (${COUNT} checks)..."
echo ""
echo "Time               Status  Duration  Size"
echo "═════════════════════════════════════════════"

for i in $(seq 1 $COUNT); do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    output=$(gocurl -w "%{http_code} %{time_total} %{size_download}" -o /dev/null -s "$URL")
    echo "$timestamp  $output"
    [ $i -lt $COUNT ] && sleep "$INTERVAL"
done

echo "═════════════════════════════════════════════"
echo "Monitoring complete"
EOF

chmod +x monitor-api.sh
echo "Running: ./monitor-api.sh (3 checks, 2s interval)"
./monitor-api.sh https://httpbin.org/get 2 3
rm monitor-api.sh
echo ""

# Example 7: Comprehensive stats
echo "7️⃣  Comprehensive Stats"
echo "Command: Multiple output variables"
gocurl -w "\n📊 Request Statistics:\n   URL:            %{url_effective}\n   Status:         %{http_code}\n   Total Time:     %{time_total}s\n   DNS Lookup:     %{time_namelookup}s\n   TCP Connect:    %{time_connect}s\n   TLS Handshake:  %{time_appconnect}s\n   Size Download:  %{size_download} bytes\n   Speed Download: %{speed_download} bytes/s\n\n" -o /dev/null -s https://api.github.com/zen
echo ""

# Example 8: Save formatted output to file
echo "8️⃣  Save Formatted Output to File"
cat > benchmark.sh << 'EOF'
#!/bin/bash
URL="$1"
OUTPUT_FILE="${2:-benchmark.txt}"

echo "Benchmarking $URL..." > "$OUTPUT_FILE"
echo "═════════════════════════════════════" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

gocurl -w "Status:         %{http_code}\nTotal Time:     %{time_total}s\nSize:           %{size_download} bytes\nSpeed:          %{speed_download} bytes/s\n" \
    -o /dev/null -s "$URL" >> "$OUTPUT_FILE"

echo "" >> "$OUTPUT_FILE"
echo "Completed: $(date)" >> "$OUTPUT_FILE"

echo "✅ Results saved to $OUTPUT_FILE"
cat "$OUTPUT_FILE"
EOF

chmod +x benchmark.sh
./benchmark.sh https://httpbin.org/get results.txt
rm benchmark.sh results.txt
echo ""

echo "✅ Output formatting examples complete!"
echo ""
echo "💡 Available Format Variables:"
echo "   %{http_code}        - HTTP status code"
echo "   %{time_total}       - Total time in seconds"
echo "   %{size_download}    - Downloaded size in bytes"
echo "   %{speed_download}   - Download speed (bytes/s)"
echo "   %{time_namelookup}  - DNS lookup time"
echo "   %{time_connect}     - TCP connect time"
echo "   %{time_appconnect}  - TLS handshake time"
echo "   %{url_effective}    - Final URL (after redirects)"
echo ""
echo "💡 Use Cases:"
echo "   • API monitoring dashboards"
echo "   • Performance benchmarking"
echo "   • CI/CD test reports"
echo "   • CSV export for analysis"
echo "   • JSON for programmatic parsing"
