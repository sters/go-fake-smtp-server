#!/bin/bash

API_HOST="${API_HOST:-localhost}"
API_PORT="${API_PORT:-11080}"
API_BASE="http://$API_HOST:$API_PORT"

echo "=== Testing Search APIs ==="
echo "API Base URL: $API_BASE"
echo ""

test_api() {
    local endpoint="$1"
    local description="$2"
    
    echo "Testing: $description"
    echo "Endpoint: $endpoint"
    
    response=$(curl -s "$API_BASE$endpoint")
    
    if [ $? -eq 0 ]; then
        count=$(echo "$response" | jq '. | length' 2>/dev/null || echo "error")
        
        if [ "$count" != "error" ]; then
            echo "✓ Success: Found $count email(s)"
            
            if [ "$count" -gt 0 ]; then
                echo "  First email:"
                # Try to parse email details, fallback to SMTP envelope data if headers are null
                echo "$response" | jq '.[0] | if .headers then {from: (.from[0].Address // .smtpFrom), to: (.to[0].Address // .smtpTo[0]), subject: (.headers[] | select(.key == "Subject") | .value)} else {from: .smtpFrom, to: .smtpTo[0], subject: "(no subject parsed)"} end' 2>/dev/null || echo "  (Could not parse email details)"
            fi
        else
            echo "✗ Error: Invalid JSON response"
            echo "  Response: $response"
        fi
    else
        echo "✗ Error: Failed to connect to API"
    fi
    
    echo ""
}

echo "1. List all emails"
test_api "/" "Get all emails"

echo "2. Search by recipient (To field)"
test_api "/search/to?email=recipient@example.com" "Search emails sent to recipient@example.com"
test_api "/search/to?email=bob@example.com" "Search emails sent to bob@example.com"

echo "3. Search by CC field"
test_api "/search/cc?email=charlie@example.com" "Search emails with charlie@example.com in CC"

echo "4. Search by BCC field"
test_api "/search/bcc?email=hr@company.com" "Search emails with hr@company.com in BCC"
test_api "/search/bcc?email=admin@example.com" "Search emails with admin@example.com in BCC"

echo "5. Search by sender (From field)"
test_api "/search/from?email=sender@example.com" "Search emails from sender@example.com"
test_api "/search/from?email=alice@example.com" "Search emails from alice@example.com"
test_api "/search/from?email=newsletter@example.com" "Search emails from newsletter@example.com"

echo "6. Search for non-existent email"
test_api "/search/to?email=nonexistent@example.com" "Search for emails to non-existent address"

echo ""
echo "=== API tests completed! ==="

if ! command -v jq &> /dev/null; then
    echo ""
    echo "Note: Install 'jq' for better JSON formatting:"
    echo "  macOS: brew install jq"
    echo "  Ubuntu/Debian: sudo apt-get install jq"
    echo "  RHEL/CentOS: sudo yum install jq"
fi