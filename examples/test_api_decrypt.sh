#!/bin/bash

# MDFriday License API Decryption Test Script
# This script tests the new /api/license/decrypt endpoint

set -e  # Exit on any error

echo "🚀 Starting MDFriday License API Decryption Test"
echo "=============================================="

# Build the binary
echo "📦 Building hugov binary..."
go build -o hugov_test ./main.go

# Clean up any existing test files
echo "🧹 Cleaning up existing test files..."
rm -rf ~/.mdfriday/licenses
rm -rf ~/.mdfriday/keys
rm -rf ./test_content
rm -rf ./encrypted_content
rm -f *.mdf.license

# Step 1: Generate keys
echo ""
echo "🔑 Step 1: Generating cryptographic keys..."
./hugov_test license keygen

# Step 2: Generate licenses
echo ""
echo "📋 Step 2: Generating test licenses..."
./hugov_test license generate -plan yearly -count 1

# Step 3: Create and encrypt test content
echo ""
echo "📄 Step 3: Creating and encrypting test content..."
mkdir -p ./test_content
echo '{"message": "Hello from basic content!", "level": "basic"}' > ./test_content/basic_test.json
echo '{"message": "Hello from premium content!", "level": "premium"}' > ./test_content/premium_test.json

# Encrypt content
./hugov_test license encrypt-level -input ./test_content/basic_test.json -level basic -output-dir ./encrypted_content
./hugov_test license encrypt-level -input ./test_content/premium_test.json -level premium -output-dir ./encrypted_content

# Step 4: Activate license
echo ""
echo "🔓 Step 4: Activating license..."
YEARLY_LICENSE=$(ls ~/.mdfriday/licenses/MDF-*.json | grep -v "licenses_" | head -1 | xargs basename -s .json)
echo "Activating license: $YEARLY_LICENSE"

./hugov_test license activate -key "$YEARLY_LICENSE" -device-id "api-test-device"

ACTIVATED_LICENSE="${YEARLY_LICENSE}_yearly.mdf.license"

if [ ! -f "$ACTIVATED_LICENSE" ]; then
    echo "❌ Failed to activate license"
    exit 1
fi

echo "✅ License activated successfully: $ACTIVATED_LICENSE"

# Step 5: Start the server in background
echo ""
echo "🌐 Step 5: Starting API server..."
./hugov_test serve &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test if server is responding
echo "🔍 Testing server connectivity..."
SERVER_TEST=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:1314/ || echo "connection_failed")
echo "📡 Server response code: $SERVER_TEST"

if [ "$SERVER_TEST" = "connection_failed" ]; then
    echo "❌ Cannot connect to server at localhost:1314"
    echo "🔍 Checking if server process is running..."
    ps aux | grep hugov_test | grep -v grep || echo "No hugov_test process found"
    exit 1
fi

# Function to cleanup server
cleanup_server() {
    if [ ! -z "$SERVER_PID" ]; then
        echo "🛑 Stopping server..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup_server EXIT

# Step 6: Test API decryption
echo ""
echo "🔍 Step 6: Testing API decryption..."

# Read license file
LICENSE_CONTENT=$(cat "$ACTIVATED_LICENSE")
LICENSE_PAYLOAD=$(echo "$LICENSE_CONTENT" | jq -r '.payload')
LICENSE_SIGNATURE=$(echo "$LICENSE_CONTENT" | jq -r '.signature')

# Read encrypted content
ENCRYPTED_BASIC=$(base64 < ./encrypted_content/basic_test.json.basic.enc)
ENCRYPTED_PREMIUM=$(base64 < ./encrypted_content/premium_test.json.premium.enc)

echo "📋 License payload length: ${#LICENSE_PAYLOAD}"
echo "📋 Signature length: ${#LICENSE_SIGNATURE}"
echo "📋 Encrypted basic content length: ${#ENCRYPTED_BASIC}"
echo "📋 Encrypted premium content length: ${#ENCRYPTED_PREMIUM}"

# Test basic content decryption
echo ""
echo "🔓 Testing basic content decryption..."
echo "📡 Making request to: http://localhost:1314/api/license/decrypt"
BASIC_RESPONSE=$(curl -s -w "HTTP_CODE:%{http_code}" -X POST http://localhost:1314/api/license/decrypt \
  -H "Content-Type: application/json" \
  -d "{
    \"encryptedContent\": \"$ENCRYPTED_BASIC\",
    \"license\": \"$LICENSE_PAYLOAD\",
    \"signature\": \"$LICENSE_SIGNATURE\"
  }")

# Extract HTTP code and response body
HTTP_CODE=$(echo "$BASIC_RESPONSE" | grep -o "HTTP_CODE:[0-9]*" | cut -d: -f2)
BASIC_RESPONSE_BODY=$(echo "$BASIC_RESPONSE" | sed 's/HTTP_CODE:[0-9]*$//')
echo "📡 HTTP Status Code: $HTTP_CODE"
echo "📄 Response Body: $BASIC_RESPONSE_BODY"
BASIC_RESPONSE="$BASIC_RESPONSE_BODY"

echo "📄 Basic decryption response:"
echo "$BASIC_RESPONSE" | jq '.'

# Check if decryption was successful
BASIC_SUCCESS=$(echo "$BASIC_RESPONSE" | jq -r '.success')
if [ "$BASIC_SUCCESS" = "true" ]; then
    echo "✅ Basic content decryption successful!"
    
    # Decode and display content
    BASIC_CONTENT=$(echo "$BASIC_RESPONSE" | jq -r '.content' | base64 -d)
    echo "📄 Decrypted basic content: $BASIC_CONTENT"
    
    # Verify content matches original
    ORIGINAL_BASIC=$(cat ./test_content/basic_test.json)
    if [ "$BASIC_CONTENT" = "$ORIGINAL_BASIC" ]; then
        echo "✅ Basic content matches original!"
    else
        echo "❌ Basic content doesn't match original"
        echo "Expected: $ORIGINAL_BASIC"
        echo "Got: $BASIC_CONTENT"
    fi
else
    echo "❌ Basic content decryption failed!"
    echo "Error: $(echo "$BASIC_RESPONSE" | jq -r '.errorMsg')"
fi

# Test premium content decryption
echo ""
echo "🔓 Testing premium content decryption..."
PREMIUM_RESPONSE=$(curl -s -X POST http://localhost:1314/api/license/decrypt \
  -H "Content-Type: application/json" \
  -d "{
    \"encryptedContent\": \"$ENCRYPTED_PREMIUM\",
    \"license\": \"$LICENSE_PAYLOAD\",
    \"signature\": \"$LICENSE_SIGNATURE\"
  }")

echo "📄 Premium decryption response:"
echo "$PREMIUM_RESPONSE" | jq '.'

# Check if decryption was successful
PREMIUM_SUCCESS=$(echo "$PREMIUM_RESPONSE" | jq -r '.success')
if [ "$PREMIUM_SUCCESS" = "true" ]; then
    echo "✅ Premium content decryption successful!"
    
    # Decode and display content
    PREMIUM_CONTENT=$(echo "$PREMIUM_RESPONSE" | jq -r '.content' | base64 -d)
    echo "📄 Decrypted premium content: $PREMIUM_CONTENT"
    
    # Verify content matches original
    ORIGINAL_PREMIUM=$(cat ./test_content/premium_test.json)
    if [ "$PREMIUM_CONTENT" = "$ORIGINAL_PREMIUM" ]; then
        echo "✅ Premium content matches original!"
    else
        echo "❌ Premium content doesn't match original"
        echo "Expected: $ORIGINAL_PREMIUM"
        echo "Got: $PREMIUM_CONTENT"
    fi
else
    echo "❌ Premium content decryption failed!"
    echo "Error: $(echo "$PREMIUM_RESPONSE" | jq -r '.errorMsg')"
fi

# Test with invalid signature
echo ""
echo "🚫 Testing with invalid signature..."
INVALID_RESPONSE=$(curl -s -X POST http://localhost:1314/api/license/decrypt \
  -H "Content-Type: application/json" \
  -d "{
    \"encryptedContent\": \"$ENCRYPTED_BASIC\",
    \"license\": \"$LICENSE_PAYLOAD\",
    \"signature\": \"invalid-signature\"
  }")

INVALID_SUCCESS=$(echo "$INVALID_RESPONSE" | jq -r '.success')
if [ "$INVALID_SUCCESS" = "false" ]; then
    echo "✅ Invalid signature correctly rejected!"
    echo "Error: $(echo "$INVALID_RESPONSE" | jq -r '.errorMsg')"
else
    echo "❌ Invalid signature should have been rejected!"
fi

echo ""
echo "🎉 API Decryption Test Completed!"
echo "✨ Summary:"
echo "   - Basic content decryption: $([ "$BASIC_SUCCESS" = "true" ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "   - Premium content decryption: $([ "$PREMIUM_SUCCESS" = "true" ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "   - Invalid signature rejection: $([ "$INVALID_SUCCESS" = "false" ] && echo "✅ PASS" || echo "❌ FAIL")"

# Cleanup will be handled by trap
echo ""
echo "🧹 Cleaning up..."
rm -rf ~/.mdfriday/licenses
rm -rf ~/.mdfriday/keys
rm -rf ./test_content
rm -rf ./encrypted_content
rm -f hugov_test
rm -f *.mdf.license

echo "✅ Test completed and cleaned up!"
