#!/bin/bash

# MDFriday Multi-Level Encryption Test Script
# This script demonstrates the new multi-KEK architecture where:
# - Basic content: Can be decrypted by lifetime + yearly licenses
# - Premium content: Can only be decrypted by yearly licenses

set -e  # Exit on any error

echo "🚀 Starting MDFriday Multi-Level Encryption Test"
echo "==============================================="

# Build the binary
echo "📦 Building hugov binary..."
go build -o hugov_test ./main.go

# Clean up any existing test files
echo "🧹 Cleaning up existing test files..."
rm -rf ~/.mdfriday/licenses
rm -rf ~/.mdfriday/keys

# Step 1: Generate keys (including multiple KEKs)
echo ""
echo "🔑 Step 1: Generating cryptographic keys (including KEK_BASIC and KEK_PREMIUM)..."
./hugov_test license keygen

# Step 2: Generate different types of licenses
echo ""
echo "📋 Step 2: Generating test licenses..."
./hugov_test license generate -plan lifetime -count 2
./hugov_test license generate -plan yearly -count 2

# Step 3: Create test content with different access levels
echo ""
echo "📄 Step 3: Creating test content with different access levels..."
mkdir -p ./test_content

# Basic content (accessible by both lifetime and yearly)
echo '{"theme": "basic", "name": "Basic Theme", "features": ["simple-layout", "basic-colors"], "level": "basic"}' > ./test_content/basic_theme.json
echo '<html><head><title>{{.Title}}</title></head><body><h1>Basic Template</h1><p>{{.Content}}</p></body></html>' > ./test_content/basic_template.html

# Premium content (only accessible by yearly)
echo '{"theme": "premium", "name": "Premium Theme", "features": ["advanced-layout", "animations", "premium-colors"], "level": "premium"}' > ./test_content/premium_theme.json
echo '<html><head><title>{{.Title}}</title></head><body><h1>Premium Template</h1><div class="premium-feature">Advanced Layout</div><p>{{.Content}}</p></body></html>' > ./test_content/premium_template.html

# Step 4: Encrypt content with different access levels
echo ""
echo "🔒 Step 4: Encrypting content with different access levels..."

# Encrypt basic content (accessible by both license types)
./hugov_test license encrypt-level -input ./test_content/basic_theme.json -level basic -output-dir ./encrypted_content
./hugov_test license encrypt-level -input ./test_content/basic_template.html -level basic -output-dir ./encrypted_content

# Encrypt premium content (only accessible by yearly licenses)
./hugov_test license encrypt-level -input ./test_content/premium_theme.json -level premium -output-dir ./encrypted_content
./hugov_test license encrypt-level -input ./test_content/premium_template.html -level premium -output-dir ./encrypted_content

echo "📁 Encrypted files:"
ls -la ./encrypted_content/

# Step 5: Test license activation and access control
echo ""
echo "🔓 Step 5: Testing license activation and access control..."

# Get license keys by checking the actual license files
LIFETIME_LICENSES=()
YEARLY_LICENSES=()

for license_file in ~/.mdfriday/licenses/MDF-*.json; do
    if [ -f "$license_file" ]; then
        license_key=$(basename "$license_file" .json)
        plan=$(cat "$license_file" | jq -r '.plan')
        if [ "$plan" = "lifetime" ]; then
            LIFETIME_LICENSES+=("$license_key")
        elif [ "$plan" = "yearly" ]; then
            YEARLY_LICENSES+=("$license_key")
        fi
    fi
done

echo "Found licenses:"
echo "  Lifetime: ${LIFETIME_LICENSES[@]}"
echo "  Yearly: ${YEARLY_LICENSES[@]}"

# Test with lifetime license
echo ""
echo "🎯 Testing with Lifetime License: ${LIFETIME_LICENSES[0]}"
./hugov_test license activate -key "${LIFETIME_LICENSES[0]}" -device-id "device-lifetime-1"

ACTIVATED_LIFETIME="${LIFETIME_LICENSES[0]}_lifetime.mdf.license"

if [ -f "$ACTIVATED_LIFETIME" ]; then
    echo "✅ Lifetime license activated successfully"
    
    # Check what resource keys are available
    echo "📋 Checking license content..."
    cat ~/.mdfriday/licenses/${LIFETIME_LICENSES[0]}.json | jq '.resourceKeys'
    
    echo ""
    echo "🔓 Testing basic content decryption (should work)..."
    ./hugov_test license decrypt -encrypted ./encrypted_content/basic_theme.json.basic.enc -license "$ACTIVATED_LIFETIME" -output-dir ./decrypted_lifetime
    
    if [ -f "./decrypted_lifetime/basic_theme.json" ]; then
        echo "✅ Lifetime license can decrypt basic content!"
        echo "📄 Decrypted basic content:"
        cat ./decrypted_lifetime/basic_theme.json
    else
        echo "❌ Failed to decrypt basic content with lifetime license"
    fi
    
    echo ""
    echo "🚫 Testing premium content decryption (should fail)..."
    if ./hugov_test license decrypt -encrypted ./encrypted_content/premium_theme.json.premium.enc -license "$ACTIVATED_LIFETIME" -output-dir ./decrypted_lifetime 2>/dev/null; then
        echo "❌ ERROR: Lifetime license should NOT be able to decrypt premium content!"
        exit 1
    else
        echo "✅ Correctly blocked: Lifetime license cannot decrypt premium content"
    fi
else
    echo "❌ Failed to activate lifetime license"
    exit 1
fi

# Test with yearly license
echo ""
echo "🎯 Testing with Yearly License: ${YEARLY_LICENSES[0]}"
./hugov_test license activate -key "${YEARLY_LICENSES[0]}" -device-id "device-yearly-1"

ACTIVATED_YEARLY="${YEARLY_LICENSES[0]}_yearly.mdf.license"

if [ -f "$ACTIVATED_YEARLY" ]; then
    echo "✅ Yearly license activated successfully"
    
    # Check what resource keys are available
    echo "📋 Checking license content..."
    cat ~/.mdfriday/licenses/${YEARLY_LICENSES[0]}.json | jq '.resourceKeys'
    
    echo ""
    echo "🔓 Testing basic content decryption (should work)..."
    ./hugov_test license decrypt -encrypted ./encrypted_content/basic_theme.json.basic.enc -license "$ACTIVATED_YEARLY" -output-dir ./decrypted_yearly
    
    if [ -f "./decrypted_yearly/basic_theme.json" ]; then
        echo "✅ Yearly license can decrypt basic content!"
        echo "📄 Decrypted basic content:"
        cat ./decrypted_yearly/basic_theme.json
    else
        echo "❌ Failed to decrypt basic content with yearly license"
    fi
    
    echo ""
    echo "🔓 Testing premium content decryption (should work)..."
    ./hugov_test license decrypt -encrypted ./encrypted_content/premium_theme.json.premium.enc -license "$ACTIVATED_YEARLY" -output-dir ./decrypted_yearly
    
    if [ -f "./decrypted_yearly/premium_theme.json" ]; then
        echo "✅ Yearly license can decrypt premium content!"
        echo "📄 Decrypted premium content:"
        cat ./decrypted_yearly/premium_theme.json
    else
        echo "❌ Failed to decrypt premium content with yearly license"
        exit 1
    fi
else
    echo "❌ Failed to activate yearly license"
    exit 1
fi

# Step 6: Verify content integrity
echo ""
echo "🔍 Step 6: Verifying content integrity..."

echo "Comparing basic content (original vs lifetime decryption):"
if diff ./test_content/basic_theme.json ./decrypted_lifetime/basic_theme.json; then
    echo "✅ Basic content integrity verified (lifetime)"
else
    echo "❌ Basic content integrity check failed (lifetime)"
    exit 1
fi

echo "Comparing basic content (original vs yearly decryption):"
if diff ./test_content/basic_theme.json ./decrypted_yearly/basic_theme.json; then
    echo "✅ Basic content integrity verified (yearly)"
else
    echo "❌ Basic content integrity check failed (yearly)"
    exit 1
fi

echo "Comparing premium content (original vs yearly decryption):"
if diff ./test_content/premium_theme.json ./decrypted_yearly/premium_theme.json; then
    echo "✅ Premium content integrity verified (yearly)"
else
    echo "❌ Premium content integrity check failed (yearly)"
    exit 1
fi

echo ""
echo "🎉 Multi-Level Encryption Test PASSED!"
echo "✨ Key findings:"
echo "   ✅ Lifetime licenses can decrypt basic content only"
echo "   ✅ Yearly licenses can decrypt both basic and premium content"
echo "   ✅ Access control is enforced at the cryptographic level"
echo "   ✅ Content integrity is maintained across all decryption operations"
echo "   ✅ Different KEKs successfully isolate content by access level"

echo ""
echo "🏆 CONCLUSION: Multi-level encryption architecture works perfectly!"
echo "   - Basic content uses KEK_BASIC (accessible by lifetime + yearly)"
echo "   - Premium content uses KEK_PREMIUM (accessible by yearly only)"
echo "   - Cryptographic access control prevents unauthorized decryption"

# Cleanup
echo ""
echo "🧹 Cleaning up test files..."
rm -rf ~/.mdfriday/licenses
rm -rf ~/.mdfriday/keys
rm -rf ./test_content
rm -rf ./encrypted_content
rm -rf ./decrypted_lifetime
rm -rf ./decrypted_yearly
rm -f hugov_test
rm -f *.mdf.license

echo "✅ Test completed and cleaned up!"
