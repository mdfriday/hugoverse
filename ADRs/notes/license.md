
curl -X POST http://127.0.0.1:1314/api/user \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=mdf_public@mdfriday.com&password=987123"


eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDEtMjVUMDc6MzU6MjcuMDM5MTE3KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZGZfcHVibGljQG1kZnJpZGF5LmNvbSJ9.amp5-gT5MpAMQLVxZjH4Jr7Lq-h5MzF_0J_8l8OiZ2A


curl -s -X POST "http://127.0.0.1:1314/api/license/create" \
-H "Content-Type: application/json" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDEtMjVUMDc6MzU6MjcuMDM5MTE3KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZGZfcHVibGljQG1kZnJpZGF5LmNvbSJ9.amp5-gT5MpAMQLVxZjH4Jr7Lq-h5MzF_0J_8l8OiZ2A" \
-d "{
\"license_key\": \"mdf-xxx-xxx-xxx\",
\"plan\": \"starter\",
\"expiry_days\": 365
}"


curl -v -X POST "http://127.0.0.1:1314/api/content?type=License" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDEtMjVUMDc6MzU6MjcuMDM5MTE3KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZGZfcHVibGljQG1kZnJpZGF5LmNvbSJ9.amp5-gT5MpAMQLVxZjH4Jr7Lq-h5MzF_0J_8l8OiZ2A" \
-F "id=-1" \
-F "license_key=mdf-xxx-xxx-xxx" \
-F "plan=starter" \
-F "expiry_days=365"