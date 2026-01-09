🚀 Starting MDFriday SubDomain API Test
========================================

🔧 STEP 0: Build and Setup
==========================
📁 Project root: /Users/weisun/github/mdfriday/hugoverse
📦 Building hugov binary...
✅ hugov binary built successfully

🌐 STEP 1: Starting Caddy Server
================================
⚠️  Caddy is already running, stopping it first...
🛑 Stopping Caddy server...
⚠️  Caddy is not running
🚀 Starting Caddy server...
🚀 Starting Caddy server in background...
Admin API: http://127.0.0.1:2019
Backend: 127.0.0.1:1314
CouchDB: 127.0.0.1:5984
Domain: localhost
Config: /tmp/caddy-config.json
PID File: /tmp/caddy.pid
Log File: /tmp/caddy.log
Mode: Development (HTTP only, no sudo required)
✅ Caddy server started successfully in background
PID: 78760

📡 Access URLs:
Core Service:  http://localhost:8080
CouchDB:       http://cdb.localhost:8080

Note: Development mode uses port 8080 (no sudo required)

💡 Tips:
- Use 'hugov caddy status' to check server status
- Use 'hugov caddy add' to add static sites
- Use 'hugov caddy stop' to stop the server
- Use 'hugov caddy export' to export current config
- Logs: tail -f /tmp/caddy.log
  ✅ PASS: Caddy server started successfully

🌐 STEP 2: Starting API Server
==============================
🚀 Starting API server...
{"level":"info","ts":"2026-01-09T14:52:58.167+0800","caller":"images/vips.go:46","msg":"starting vips worker queue with 3 workers"}
The preview site cleanup task has been initiated and will run once every hour...
🔍 Testing server connectivity...
✅ PASS: API server started (HTTP status: 404)

🔑 STEP 3: Login and Create Test License
=========================================

📋 3.1: Admin Login
-------------------
Email: me@sunwei.xyz
📡 Login response: {"data":["eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTQ6NTM6MDEuMjA2ODcxKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.vu1arJnrEGw-XBe8HKjUa53aaE0ElYK6r38B8xC82j4"]}
✅ PASS: Admin login successful
Token: eyJ0eXAiOiJKV1QiLCJh...

📋 3.2: Create Test License via CLI
------------------------------------
🚀 Running: hugov license generate -email me@sunwei.xyz -password *** -plan starter -count 1
DEBUG Started user database: d66e65ad75
🚀 Batch License Generation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Email: me@sunwei.xyz
API: http://127.0.0.1:1314
Plan: starter
Count: 1
Expiry: 365 days
Max Devices: 3
Max IPs: 3

📝 Step 1: Logging in...
✅ Login successful

📝 Step 2: Generating 1 licenses...

[1/1] Creating: MDF-VEAV-VYTV-M6L6
→ Creating user: veav-vytv-m6l6@mdfriday.com
✅ User created
→ Creating license
✅ License created

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Generation Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 1
Success: 1
Failed: 0

✅ Generated Licenses:

1. License Key: MDF-VEAV-VYTV-M6L6
   Email:       veav-vytv-m6l6@mdfriday.com
   Password:    dmVhdi12eXR2LW02bDY=

💡 Tips:
- Save these credentials in a secure location
- Users can login with Email + Password
- License Key is used for activation
  🎉 All licenses and users created successfully!
  ✅ PASS: License created successfully
  License Key: MDF-VEAV-VYTV-M6L6

📋 3.3: Verify License via API
-------------------------------
Response: {
"data": [
{
"activated": false,
"activated_at": 0,
"current_devices": 0,
"current_ips": 0,
"expires_at": 0,
"features": {
"max_devices": 3,
"max_ips": 3,
"sync_enabled": true,
"sync_quota": 500,
"publish_enabled": true,
"max_sites": 3,
"max_storage": 1024,
"custom_domain": false,
"validity_days": 365
},
"is_expired": true,
"is_valid": false,
"issue_date": 0,
"license_key": "MDF-VEAV-VYTV-M6L6",
"max_devices": 3,
"max_ips": 3,
"plan": "starter"
}
]
}
ℹ️  Could not verify license via API (may need activation first)

🔑 STEP 4: Activate License
===========================
ℹ️  Activating license: MDF-VEAV-VYTV-M6L6
ℹ️  Device ID: test-device-1767941581
📡 Activation response: {
"data": [
{
"activated": true,
"expires_at": 1799477581509,
"features": {
"max_devices": 3,
"max_ips": 3,
"sync_enabled": true,
"sync_quota": 500,
"publish_enabled": true,
"max_sites": 3,
"max_storage": 1024,
"custom_domain": false,
"validity_days": 365
},
"first_time": true,
"license_key": "MDF-VEAV-VYTV-M6L6",
"plan": "starter",
"success": true,
"sync": {
"db_endpoint": "https://cdb.127.0.0.1",
"db_name": "userdb-996e2873fb0730f4",
"db_password": "dmVhdi12eXR2LW02bDY=",
"email": "veav-vytv-m6l6@mdfriday.com",
"status": "active"
},
"user": {
"email": "veav-vytv-m6l6@mdfriday.com",
"user_dir": "d66e65ad75"
}
}
]
}
✅ PASS: License activated successfully
User directory: d66e65ad75
First time activation: true
Sync enabled: yes

📡 STEP 5: Test SubDomain APIs
==============================

📋 5.1: GET /api/license/subdomain - Get SubDomain Info
-------------------------------------------------------
Response: {
"data": [
{
"error": "Publish domain not found",
"success": false
}
]
}
ℹ️  GET subdomain returned expected error (no active subdomain)

📋 5.2: POST /api/license/subdomain/check - Check Valid SubDomain
------------------------------------------------------------------
Response: {
"data": [
{
"available": true,
"message": "Subdomain is available",
"subdomain": "testsite1767941582"
}
]
}
✅ PASS: Valid subdomain 'testsite1767941582' is available

📋 5.3: POST /api/license/subdomain/check - Check Too Short SubDomain
----------------------------------------------------------------------
Response: {
"data": [
{
"available": false,
"message": "Subdomain must be at least 4 characters long",
"reason": "invalid_format",
"subdomain": "ab"
}
]
}
✅ PASS: Too short subdomain correctly rejected

📋 5.4: POST /api/license/subdomain/check - Check Reserved SubDomain
---------------------------------------------------------------------
Response: {
"data": [
{
"available": false,
"message": "Subdomain 'admin' is reserved",
"reason": "reserved",
"subdomain": "admin"
}
]
}
✅ PASS: Reserved subdomain correctly rejected

📋 5.5: POST /api/license/subdomain/check - Check Invalid Characters
---------------------------------------------------------------------
Response: {
"data": [
{
"available": false,
"message": "Subdomain can only contain lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen",
"reason": "invalid_format",
"subdomain": "test_site!"
}
]
}
✅ PASS: Invalid characters subdomain correctly rejected

📋 5.6: POST /api/license/subdomain/update - Update SubDomain
--------------------------------------------------------------
Response: {
"data": [
{
"error": "Publish domain not found",
"success": false
}
]
}
ℹ️  Update subdomain returned error (expected if license not activated)
Error: Publish domain not found

📋 5.7: POST /api/license/subdomain/update - Update with Too Short Name
------------------------------------------------------------------------
Response: {
"data": [
{
"error": "Subdomain must be at least 4 characters long",
"success": false
}
]
}
✅ PASS: Too short subdomain update correctly rejected

🌐 STEP 6: Test Custom Domain APIs
===================================

📋 6.1: GET /api/license/domains - Get All Domains
---------------------------------------------------
Response: {
"data": [
{
"error": "Publish domain not found",
"success": false
}
]
}
ℹ️  GET domains returned expected error (license may not have been activated)

📋 6.2: POST /api/license/domain/check - Check Domain Readiness
----------------------------------------------------------------
Response: {
"data": [
{
"error": "Custom domain feature not enabled for this license plan",
"success": false
}
]
}
ℹ️  Domain check returned error (expected for non-configured domain)

🌐 STEP 7: Test Caddy Integration
==================================

📋 7.1: List Caddy domains
--------------------------

🧹 Cleaning up...
Stopping API server (PID: 78766)...
Stopping Caddy...
🛑 Stopping Caddy server...
Stopping Caddy (PID: 78760)...
✅ Caddy server stopped successfully
✅ Cleanup completed