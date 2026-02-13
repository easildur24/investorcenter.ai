#!/bin/bash
# Direct test of Polygon.io API to isolate the real issue
# Tests the exact same endpoint our Go backend is using

API_KEY="${POLYGON_API_KEY:-}"
if [ -z "$API_KEY" ]; then
    echo "❌ ERROR: POLYGON_API_KEY environment variable is not set"
    echo "   Export it before running: export POLYGON_API_KEY='your-api-key'"
    exit 1
fi
BASE_URL="https://api.polygon.io"
SYMBOL="X:ETHUSD"

echo "🧪 Direct Polygon.io API Test"
echo "Testing the EXACT same endpoint our Go backend uses"
echo "This will help identify if the issue is in our code or Polygon API"
echo ""
echo "📋 Endpoint: ${BASE_URL}/v2/aggs/ticker/${SYMBOL}/prev?adjusted=true&apikey=${API_KEY}"
echo "🔄 Making 5 calls with 2s delay"
echo "============================================================"

# Test function
test_call() {
    local call_num=$1
    local url="${BASE_URL}/v2/aggs/ticker/${SYMBOL}/prev?adjusted=true&apikey=${API_KEY}"
    
    echo ""
    echo "📞 Call ${call_num}/5 at $(date +'%H:%M:%S')"
    
    # Make the request and capture both response and status
    response=$(curl -s -w "\nSTATUS_CODE:%{http_code}\nRESPONSE_TIME:%{time_total}" "$url")
    
    # Extract status code and response time
    status_code=$(echo "$response" | grep "STATUS_CODE:" | cut -d':' -f2)
    response_time=$(echo "$response" | grep "RESPONSE_TIME:" | cut -d':' -f2)
    response_body=$(echo "$response" | sed '/STATUS_CODE:/,$d')
    
    echo "📊 Status Code: $status_code"
    echo "⏱️  Response Time: ${response_time}s"
    
    if [ "$status_code" = "200" ]; then
        # Parse JSON response
        api_status=$(echo "$response_body" | jq -r '.status // "UNKNOWN"')
        echo "✅ API Status: $api_status"
        
        if [ "$api_status" = "OK" ]; then
            price=$(echo "$response_body" | jq -r '.results[0].c // "N/A"')
            volume=$(echo "$response_body" | jq -r '.results[0].v // "N/A"')
            timestamp=$(echo "$response_body" | jq -r '.results[0].t // "N/A"')
            
            echo "💰 Price: \$${price}"
            echo "📈 Volume: ${volume}"
            echo "🕐 Timestamp: ${timestamp}"
            
            if [ "$timestamp" != "N/A" ]; then
                # Convert timestamp to readable date (assuming it's in milliseconds)
                if command -v date >/dev/null 2>&1; then
                    readable_date=$(date -r $((timestamp/1000)) 2>/dev/null || echo "Invalid timestamp")
                    echo "📅 Date: $readable_date"
                fi
            fi
        else
            echo "❌ API Error Status: $api_status"
            echo "🚫 Full response: $response_body"
        fi
    else
        echo "❌ HTTP Error: $status_code"
        echo "🚫 Error response: $response_body"
    fi
}

# Run the tests
for i in {1..5}; do
    test_call $i
    
    if [ $i -lt 5 ]; then
        echo "⏳ Waiting 2 seconds..."
        sleep 2
    fi
done

echo ""
echo "============================================================"
echo "📊 CONCLUSION:"
echo "============================================================"
echo "If all calls returned 200 with 'OK' status → Issue is in our Go backend"
echo "If calls failed with rate limit errors → Issue is Polygon API limits"
echo "If calls failed with auth errors → Issue is API key/permissions"
echo ""
echo "🎯 Next Steps:"
echo "- If API works: Debug our Go backend implementation"
echo "- If API fails: Check Polygon plan/key configuration"
