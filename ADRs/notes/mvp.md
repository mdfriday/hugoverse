
- 初始化用户本地环境
	- 帮助用户生成一个新的系统用户
	- 用户创建站点
		- 帮助用户生成Author
			- 一个是主题Author: mdfriday.com。 生成前查询是否已经创建。
			- 一个是用户自己：随机first name, last name。创建后记录在配置信息中。
		- 注册用户选择的主题
			- 查询主题是否已经存在
		- 创建站点

### 创建 License

```shell
./hugov serve

export PATH="$HOME/go/bin:$PATH"
./hugov caddy start

./hugov license generate -email me@sunwei.xyz -password 123456 -plan starter -count 1

#   1. License Key: MDF-7XV5-DPW2-4VTT
#      Email:       7xv5-dpw2-4vtt@mdfriday.com
#      Password:    N3h2NS1kcHcyLTR2dHQ=

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=7xv5-dpw2-4vtt@mdfriday.com&password=N3h2NS1kcHcyLTR2dHQ="

# eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMTFUMTY6NDI6MDEuNDQ3MDY0KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiI3eHY1LWRwdzItNHZ0dEBtZGZyaWRheS5jb20ifQ.kOKXRLFbv8VV_g-OsnQBVuranUabF8W47Fuwf_CqiwM

curl -s -X GET "http://127.0.0.1:1314/api/license/info?key=MDF-7XV5-DPW2-4VTT" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMTFUMTY6NDI6MDEuNDQ3MDY0KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiI3eHY1LWRwdzItNHZ0dEBtZGZyaWRheS5jb20ifQ.kOKXRLFbv8VV_g-OsnQBVuranUabF8W47Fuwf_CqiwM"

#{"data":[{"activated":true,"activated_at":1768207508890,"current_devices":0,"current_ips":0,"expires_at":1799743508890,"features":{"max_devices":3,"max_ips":3,"sync_enabled":true,"sync_quota":500,"publish_enabled":true,"max_sites":3,"max_storage":1024,"custom_domain":false,"validity_days":365},"is_expired":false,"is_valid":true,"issue_date":1768207508890,"license_key":"MDF-7XV5-DPW2-4VTT","max_devices":3,"max_ips":3,"plan":"starter"}]}

curl -s -X POST "http://127.0.0.1:1314/api/license/activate" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMTFUMTY6NDI6MDEuNDQ3MDY0KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiI3eHY1LWRwdzItNHZ0dEBtZGZyaWRheS5jb20ifQ.kOKXRLFbv8VV_g-OsnQBVuranUabF8W47Fuwf_CqiwM" \
    -F "license_key=MDF-7XV5-DPW2-4VTT" \
    -F "device_id=DID_11223322" \
    -F "device_name=Test Device 22" \
    -F "device_type=desktop"

# {"data":[{"activated":true,"expires_at":1799480892903,"features":{"max_devices":3,"max_ips":3,"sync_enabled":true,"sync_quota":500,"publish_enabled":true,"max_sites":3,"max_storage":1024,"custom_domain":false,"validity_days":365},"first_time":false,"license_key":"MDF-CPJB-CJSG-2B7C","plan":"starter","success":true,"sync":{"db_endpoint":"https://cdb.127.0.0.1","db_name":"userdb-6873577151b0b6e2","db_password":"Y3BqYi1janNnLTJiN2M=","email":"cpjb-cjsg-2b7c@mdfriday.com","status":"active"},"user":{"email":"cpjb-cjsg-2b7c@mdfriday.com","user_dir":"6873577151"}}]}

curl -s -X GET "http://127.0.0.1:1314/api/license/subdomain?key=MDF-CPJB-CJSG-2B7C" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18"

# {"data":[{"created_at":1767944892903,"folder":"6873577151","full_domain":"6873577151.127.0.0.1","subdomain":"6873577151"}]}


curl -s -X POST "http://127.0.0.1:1314/api/license/subdomain/check" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18" \
    -F "license_key=MDF-CPJB-CJSG-2B7C" \
    -F "subdomain=td66e65ad75"
    
    
curl -s -X POST "http://127.0.0.1:1314/api/license/subdomain/update" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18" \
    -F "license_key=MDF-CPJB-CJSG-2B7C" \
    -F "new_subdomain=td66e65ad75"
    
curl -s -X GET "http://127.0.0.1:1314/api/license/domains?key=MDF-CPJB-CJSG-2B7C" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18"


#1. 检查自定义域名就绪状态 (DNS + HTTP 可达性)
curl -s -X POST "http://127.0.0.1:1314/api/license/domain/check" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18" \
    -F "license_key=MDF-CPJB-CJSG-2B7C" \
    -F "domain=hello.com"

# 预期响应 (域名就绪):
# {"data":[{"domain":"hello.com","dns_valid":true,"resolved_ips":["1.2.3.4"],"http_reachable":true,"ready":true,"message":"Domain is ready for HTTPS certificate issuance"}]}

# 预期响应 (DNS 未配置):
# {"data":[{"domain":"hello.com","dns_valid":false,"resolved_ips":["5.6.7.8"],"http_reachable":false,"ready":false,"error":"DNS does not point to server IP"}]}


# 2. 添加自定义域名
curl -s -X POST "http://127.0.0.1:1314/api/license/domain/add" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18" \
    -F "license_key=MDF-CPJB-CJSG-2B7C" \
    -F "domain=hello.com"

# 预期响应 (成功):
# {"data":[{"domain":"hello.com","status":"pending_certificate","message":"Custom domain added. SSL certificate is being issued."}]}

# 预期响应 (功能未启用):
# {"data":[{"error":"Custom domain feature not enabled for this license plan","success":false}]}


# 3. 移除自定义域名
curl -s -X POST "http://127.0.0.1:1314/api/license/domain/remove" \
    -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjYtMDItMDhUMTU6NDQ6MjIuMDMyMjI4KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJjcGpiLWNqc2ctMmI3Y0BtZGZyaWRheS5jb20ifQ.XMib-pxIwb6fPD60c7A1KaqQ2gN1O0kHt4tXZ63oe18" \
    -F "license_key=MDF-CPJB-CJSG-2B7C" \
    -F "domain=hello.com"

# 预期响应 (成功):
# {"data":[{"domain":"hello.com","message":"Custom domain removed successfully"}]}

# 预期响应 (域名不属于该用户):
# {"data":[{"error":"Domain does not belong to this license","success":false}]}
     
```

### 创建用户

curl -X POST http://127.0.0.1:1314/api/user \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=mdf_public@mdfriday.com&password=987123"

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=me@sunwei.xyz&password=123456"

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=user_625216@mdfriday.com&password=123456"

### 创建 Counter

curl -v -X POST "http://127.0.0.1:1314/api/counter?type=Counter" \
-F "id=-1" \
-F "kind=preview" \
-F "request_id=test-request-id" 

### 创建 CTA

curl -v -X POST "http://127.0.0.1:1314/api/content?type=CTA" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDYtMjVUMTM6NTY6NDkuMDEwNDYyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.Y-yMv7_sacxSAsYVBUriBOzMiBl72uSp-W3rA-50uy8" \
-F "id=-1" \
-F "name=test-cta" \
-F "email=xxx@example.com"

curl -X GET "http://127.0.0.1:1314/api/contents?type=CTA&count=10&offset=0&order=desc" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDYtMjVUMTM6NTY6NDkuMDEwNDYyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.Y-yMv7_sacxSAsYVBUriBOzMiBl72uSp-W3rA-50uy8" 

### 创建站点

curl -X POST "http://127.0.0.1:1314/api/content?type=Site" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDRUMDg6MTQ6NTIuNTk2MDI5KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.foManZwcdG0h52dCxeKY6jE6iTkdSZFcEbnGFanLZU0" \
-F "type=Site" \
-F "title=Demo" \
-F "description=This is my first demo site created by hugoverse" \
-F "base_url=/" \
-F "theme=github.com/mdfriday/theme-manual-of-me" \
-F "owner=me@sunwei.xyz" \
-F "Params=Author = '老袁讲敏捷'
CoverImage = 'cover.jpeg'" \
-F "working_dir=/.local/share/temp"


#### 站点语言，可选，默认EN


#### 创建Post

curl -X POST "http://127.0.0.1:1314/api/content?type=Post" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDVUMTY6MTE6NDQuOTU3ODUxKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJ1c2VyXzk3NjkxNkBtZGZyaWRheS5jb20ifQ.iLWi8wWHg2C9JuJsEQW7WB7m0k524Wcv_Ok0TW3j2zA" \
-F "type=Post" \
-F "title=关于我" \
-F "author=laoyuan" \
-F "params=weight: 1" \
-F "assets.0=@/Users/sunwei/Downloads/building.jpg" \
-F "assets.1=@/Users/sunwei/Downloads/laoyuan-bili.jpg" \
-F "content=- **个人长期陪跑教练**
- 企业级敏捷教练
- 研发团队效能顾问
- unFIX中文社区发起人
- 中国最大的敏捷主题个人自媒体（bilibili \"老袁讲敏捷\"）
- \"老袁讲敏捷\" 公众号和视频号
- 长篇小说作家（湖北省作协会员）

\![good](building.jpg)

---

"



curl -X POST "http://127.0.0.1:1314/api/content?type=Post" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDVUMTY6MTE6NDQuOTU3ODUxKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJ1c2VyXzk3NjkxNkBtZGZyaWRheS5jb20ifQ.iLWi8wWHg2C9JuJsEQW7WB7m0k524Wcv_Ok0TW3j2zA" \
-F "type=Post" \
-F "content=- **个人长期陪跑教练**
- 企业级敏捷教练
- 研发团队效能顾问
- unFIX中文社区发起人
- 中国最大的敏捷主题个人自媒体（bilibili \"老袁讲敏捷\"）
- \"老袁讲敏捷\" 公众号和视频号
- 长篇小说作家（湖北省作协会员）

\![good](building.jpg)

---

"

创建SitePost

curl -X POST "http://127.0.0.1:1314/api/content?type=SitePost" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDRUMDg6MTQ6NTIuNTk2MDI5KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.foManZwcdG0h52dCxeKY6jE6iTkdSZFcEbnGFanLZU0" \
-F "site=/api/content?type=Site&id=2" \
-F "post=/api/content?type=Post&id=3" \
-F "path=/content/01.service.md"

#### Preview

curl -X POST "http://127.0.0.1:1314/api/preview?type=Site&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDZUMTk6MDk6MjcuNDUwNzA1KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJ1c2VyXzYyNTIxNkBtZGZyaWRheS5jb20ifQ.ZJMDUiRshJUAUXts6lZCtPNDZnFyPqx-ujqeIfi6xJw"


#### Deployment

curl -X POST "http://127.0.0.1:1314/api/deploy?type=Site&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjQtMTItMDRUMDg6MTQ6NTIuNTk2MDI5KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.foManZwcdG0h52dCxeKY6jE6iTkdSZFcEbnGFanLZU0"

### Sign

curl -X GET "http://127.0.0.1:1314/api/signature" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDYtMjVUMTM6NTY6NDkuMDEwNDYyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.Y-yMv7_sacxSAsYVBUriBOzMiBl72uSp-W3rA-50uy8"\

curl -v -X POST "http://127.0.0.1:1314/api/cta/submit?type=CTA" \
-H "X-Signature: bGRN2Mg7WPWIr87gzOyNHnHT1lZbGvaFru7WGY-fh_8" \
-H "X-Signer: me@sunwei.xyz" \
-F "id=-1" \
-F "name=test-cta" \
-F "email=xxx@example.com"

#### Search

curl -X GET "http://127.0.0.1:1314/api/search?type=Image&q=tags:Test" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDUtMDJUMDg6NTk6NTkuNzM0MzE2KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.JlpCgsiRyLqqamxhmQOaCHL3vJP45bhqztTHnQBaWAk"

curl -X GET "http://127.0.0.1:1314/api/search2?type=Language" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTA6MTY6MDAuMTY5NjQ2KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.p9puX1tjZ5WpaUKtCclzkB9W6qCWVbFmmKJtAlBRV6Y"

curl -X GET "http://127.0.0.1:1314/api/content?type=Image&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTA6NDg6MzYuMTQ0MTUyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.cDTG9kJQoXcM00YdwuEXDqfPuH2XQNXvzS86WN7Gc-w"


curl -X GET "http://127.0.0.1:1314/api/content?type=Image&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTM6MzM6MjkuMzU4OTQrMDg6MDAiLCJpYXQiOm51bGwsImlzcyI6bnVsbCwianRpIjpudWxsLCJuYmYiOm51bGwsInN1YiI6bnVsbCwidXNlciI6ImFiY0BzdW53ZWkueHl6In0.-33W-Z0Epz9Ve8d0n_oLtOW9dw5FzBHZqzAQE2y6EkQ"


#### Tags

curl -X GET "http://127.0.0.1:1314/api/content/tags?type=Image" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDUtMDJUMDg6NTk6NTkuNzM0MzE2KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.JlpCgsiRyLqqamxhmQOaCHL3vJP45bhqztTHnQBaWAk"


#### Image

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/images?type=Image&count=10&offset=0&order=desc"
{"data":[{"uuid":"efd6c43b-161d-4008-a9ab-cd7e22e4d66a","status":"public","namespace":"Image","id":1,"slug":"name","hash":"","timestamp":1743471048000,"updated":1743471048797,"name":"name","asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/screencapture-notes-sunwei-xyz-zh-2025-03-24-081335.png","tags":["Test"],"width":1905,"height":1311}]}

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/image/search?type=Image&count=10&offset=0&order=desc&q=name%3ATest%20OR%20tags%3ATest"
{"data":[{"uuid":"efd6c43b-161d-4008-a9ab-cd7e22e4d66a","status":"public","namespace":"Image","id":1,"slug":"name","hash":"","timestamp":1743471048000,"updated":1743471048797,"name":"name","asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/screencapture-notes-sunwei-xyz-zh-2025-03-24-081335.png","tags":["Test"],"width":1905,"height":1311}]}

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/image/tags?type=Image"
{"data":[["Test"]]}

curl -X GET "http://127.0.0.1:1314/image/100/100"
http://127.0.0.1:1314/image/id/1/100/100.jpg?hmac=jBxrm5Pz0xzYEY01kLEk9KsbVLvJX88tDtIpL0S8E9U

#### ShortCode

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/sc/tags?type=ShortCode"
{"data":[["xhs","小红书"]]}

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/sc/search?type=ShortCode&count=10&offset=0&order=desc&q=tags%3Axhs%20OR%20tags%3A小红书"
{"data":[{"uuid":"98b64ab7-27a4-48d6-9083-da9ae9af093c","status":"public","namespace":"ShortCode","id":1,"slug":"cardbanner","hash":"","timestamp":1743726576000,"updated":1743726636122,"name":"cardBanner","template":"\u003cstyle\u003e\r\n.cardbanner {\r\n    font-family: Arial, sans-serif;\r\n            padding: 40px;\r\n            background-color: #f5f5f5;\r\n            max-width: 1080px;\r\n            margin: 0 auto;\r\n}\r\n\r\n.cardbanner .header {\r\n   display: flex;\r\n            justify-content: space-between;\r\n            align-items: flex-start;\r\n            margin-bottom: 100px;\r\n}\r\n\r\n.cardbanner .logo {\r\n    font-size: 24px;\r\n    font-weight: bold;\r\n}\r\n\r\n.cardbanner .avatar {\r\n    width: 60px;\r\n            height: 60px;\r\n            font-size: 40px;\r\n            border-radius: 50%; /* 让图片变成圆形 */\r\n            object-fit: cover; /* 确保图片填充整个圆形 */\r\n            display: block;\r\n}\r\n\r\n.cardbanner .main-title {\r\n    font-size: 52px;\r\n            font-weight: bold;\r\n            margin-bottom: 20px;\r\n            line-height: 1.2;\r\n}\r\n\r\n.cardbanner .subtitle {\r\n    font-size: 65px;\r\n            font-weight: bold;\r\n            background: linear-gradient(transparent 60%, #FFB6C1 40%);\r\n            display: inline-block;\r\n            margin-bottom: 15px;\r\n            letter-spacing: 15px;\r\n}\r\n\r\n.cardbanner .description {\r\n    font-size: 23px;\r\n            color: #666;\r\n            margin-bottom: 45px;\r\n}\r\n\r\n.cardbanner .new-label {\r\n    position: relative;\r\n            display: inline-block;\r\n            margin-top: 60px;\r\n            transform: rotate(-10deg);\r\n            width: 100%;\r\n}\r\n\r\n.cardbanner .new-tag {\r\n    background: #4169E1;\r\n            color: white;\r\n            padding: 10px 20px;\r\n            border-radius: 15px;\r\n            position: absolute;  /* 绝对定位 */\r\n            right: 0;  /* 让它紧贴 .new-label 右侧 */\r\n            top: 50%;  /* 垂直居中 */\r\n            transform: translateY(-50%) rotate(30deg);  /* 保持旋转但居中 */\r\n            display: inline-block;\r\n            font-weight: bold;\r\n            font-size: 28px;\r\n            box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);\r\n}\r\n\r\n.cardbanner .new-tag::after {\r\n    content: \"!!\";\r\n    color: white;\r\n    margin-left: 5px;\r\n}\r\n\r\n.cardbanner .footer {\r\n     margin-top: 174px;\r\n            display: flex;\r\n            justify-content: space-between;\r\n            font-size: 20px;\r\n            color: #333;\r\n}\r\n\r\n.cardbanner .footer span {\r\n    margin: 0 10px;\r\n}\r\n\r\n.cardbanner .divider {\r\n    color: #999;\r\n}\r\n      \u003c/style\u003e\r\n      \u003cdiv class=\"cardbanner\"\u003e\r\n        \u003cdiv class=\"header\"\u003e\r\n          \u003cdiv class=\"logo\"\u003e{{ .Get \"logo\" }}\u003c/div\u003e\r\n          \u003cdiv class=\"avatar\"\u003e\r\n           \u003cimg class=\"avatar\" src='{{ .Get \"avatar\" }}' alt=\"头像\"\u003e\r\n          \u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"main-title\"\u003e\r\n          {{ .Get \"mainTitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"subtitle\"\u003e\r\n          {{ .Get \"subtitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"description\"\u003e\r\n          {{ .Get \"description\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"new-label\"\u003e\r\n          \u003cdiv class=\"new-tag\"\u003e{{ .Get \"newTagText\" }}\u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"footer\"\u003e\r\n        {{ $topics := split (.Get \"footerContent\") \",\" }}\r\n        {{ range $index, $topic := $topics }}\r\n            {{ if gt $index 0 }} \u003cspan class=\"divider\"\u003e|\u003c/span\u003e {{ end }}\r\n            {{ $topic }}\r\n        {{ end }}\r\n        \u003c/div\u003e\r\n      \u003c/div\u003e","example":"{{\u003c cardBanner\r\n    logo=\"不黑学长\"\r\n    avatar=\"/images/avatar.png\"\r\n    mainTitle=\"让完播率\u003e50% (3/3)\"\r\n    subtitle=\"6种文案公式\"\r\n    description=\"爆款拆解/爆款要素/文案结构\"\r\n    newTagText=\"全新整理\"\r\n    footerContent=\"运营技巧,爆款选题,文案写作,数据复盘\"\r\n/\u003e}}","asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/1743726636150-1.jpg","tags":["xhs","小红书"],"width":1080,"height":1440}]}

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/scs?type=ShortCode&count=10&offset=0&order=desc"
{"data":[{"uuid":"98b64ab7-27a4-48d6-9083-da9ae9af093c","status":"public","namespace":"ShortCode","id":1,"slug":"cardbanner","hash":"","timestamp":1743726576000,"updated":1743726636122,"name":"cardBanner","template":"\u003cstyle\u003e\r\n.cardbanner {\r\n    font-family: Arial, sans-serif;\r\n            padding: 40px;\r\n            background-color: #f5f5f5;\r\n            max-width: 1080px;\r\n            margin: 0 auto;\r\n}\r\n\r\n.cardbanner .header {\r\n   display: flex;\r\n            justify-content: space-between;\r\n            align-items: flex-start;\r\n            margin-bottom: 100px;\r\n}\r\n\r\n.cardbanner .logo {\r\n    font-size: 24px;\r\n    font-weight: bold;\r\n}\r\n\r\n.cardbanner .avatar {\r\n    width: 60px;\r\n            height: 60px;\r\n            font-size: 40px;\r\n            border-radius: 50%; /* 让图片变成圆形 */\r\n            object-fit: cover; /* 确保图片填充整个圆形 */\r\n            display: block;\r\n}\r\n\r\n.cardbanner .main-title {\r\n    font-size: 52px;\r\n            font-weight: bold;\r\n            margin-bottom: 20px;\r\n            line-height: 1.2;\r\n}\r\n\r\n.cardbanner .subtitle {\r\n    font-size: 65px;\r\n            font-weight: bold;\r\n            background: linear-gradient(transparent 60%, #FFB6C1 40%);\r\n            display: inline-block;\r\n            margin-bottom: 15px;\r\n            letter-spacing: 15px;\r\n}\r\n\r\n.cardbanner .description {\r\n    font-size: 23px;\r\n            color: #666;\r\n            margin-bottom: 45px;\r\n}\r\n\r\n.cardbanner .new-label {\r\n    position: relative;\r\n            display: inline-block;\r\n            margin-top: 60px;\r\n            transform: rotate(-10deg);\r\n            width: 100%;\r\n}\r\n\r\n.cardbanner .new-tag {\r\n    background: #4169E1;\r\n            color: white;\r\n            padding: 10px 20px;\r\n            border-radius: 15px;\r\n            position: absolute;  /* 绝对定位 */\r\n            right: 0;  /* 让它紧贴 .new-label 右侧 */\r\n            top: 50%;  /* 垂直居中 */\r\n            transform: translateY(-50%) rotate(30deg);  /* 保持旋转但居中 */\r\n            display: inline-block;\r\n            font-weight: bold;\r\n            font-size: 28px;\r\n            box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);\r\n}\r\n\r\n.cardbanner .new-tag::after {\r\n    content: \"!!\";\r\n    color: white;\r\n    margin-left: 5px;\r\n}\r\n\r\n.cardbanner .footer {\r\n     margin-top: 174px;\r\n            display: flex;\r\n            justify-content: space-between;\r\n            font-size: 20px;\r\n            color: #333;\r\n}\r\n\r\n.cardbanner .footer span {\r\n    margin: 0 10px;\r\n}\r\n\r\n.cardbanner .divider {\r\n    color: #999;\r\n}\r\n      \u003c/style\u003e\r\n      \u003cdiv class=\"cardbanner\"\u003e\r\n        \u003cdiv class=\"header\"\u003e\r\n          \u003cdiv class=\"logo\"\u003e{{ .Get \"logo\" }}\u003c/div\u003e\r\n          \u003cdiv class=\"avatar\"\u003e\r\n           \u003cimg class=\"avatar\" src='{{ .Get \"avatar\" }}' alt=\"头像\"\u003e\r\n          \u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"main-title\"\u003e\r\n          {{ .Get \"mainTitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"subtitle\"\u003e\r\n          {{ .Get \"subtitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"description\"\u003e\r\n          {{ .Get \"description\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"new-label\"\u003e\r\n          \u003cdiv class=\"new-tag\"\u003e{{ .Get \"newTagText\" }}\u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"footer\"\u003e\r\n        {{ $topics := split (.Get \"footerContent\") \",\" }}\r\n        {{ range $index, $topic := $topics }}\r\n            {{ if gt $index 0 }} \u003cspan class=\"divider\"\u003e|\u003c/span\u003e {{ end }}\r\n            {{ $topic }}\r\n        {{ end }}\r\n        \u003c/div\u003e\r\n      \u003c/div\u003e","example":"{{\u003c cardBanner\r\n    logo=\"不黑学长\"\r\n    avatar=\"/images/avatar.png\"\r\n    mainTitle=\"让完播率\u003e50% (3/3)\"\r\n    subtitle=\"6种文案公式\"\r\n    description=\"爆款拆解/爆款要素/文案结构\"\r\n    newTagText=\"全新整理\"\r\n    footerContent=\"运营技巧,爆款选题,文案写作,数据复盘\"\r\n/\u003e}}","asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/1743726636150-1.jpg","tags":["xhs","小红书"],"width":1080,"height":1440}]}

➜  mdfriday curl -X GET "http://127.0.0.1:1314/api/sc?type=ShortCode&status=&id=1"
{"data":[{"uuid":"98b64ab7-27a4-48d6-9083-da9ae9af093c","status":"public","namespace":"ShortCode","id":1,"slug":"cardbanner","hash":"","timestamp":1743726576000,"updated":1743726636122,"name":"cardBanner","template":"\u003cstyle\u003e\r\n.cardbanner {\r\n    font-family: Arial, sans-serif;\r\n            padding: 40px;\r\n            background-color: #f5f5f5;\r\n            max-width: 1080px;\r\n            margin: 0 auto;\r\n}\r\n\r\n.cardbanner .header {\r\n   display: flex;\r\n            justify-content: space-between;\r\n            align-items: flex-start;\r\n            margin-bottom: 100px;\r\n}\r\n\r\n.cardbanner .logo {\r\n    font-size: 24px;\r\n    font-weight: bold;\r\n}\r\n\r\n.cardbanner .avatar {\r\n    width: 60px;\r\n            height: 60px;\r\n            font-size: 40px;\r\n            border-radius: 50%; /* 让图片变成圆形 */\r\n            object-fit: cover; /* 确保图片填充整个圆形 */\r\n            display: block;\r\n}\r\n\r\n.cardbanner .main-title {\r\n    font-size: 52px;\r\n            font-weight: bold;\r\n            margin-bottom: 20px;\r\n            line-height: 1.2;\r\n}\r\n\r\n.cardbanner .subtitle {\r\n    font-size: 65px;\r\n            font-weight: bold;\r\n            background: linear-gradient(transparent 60%, #FFB6C1 40%);\r\n            display: inline-block;\r\n            margin-bottom: 15px;\r\n            letter-spacing: 15px;\r\n}\r\n\r\n.cardbanner .description {\r\n    font-size: 23px;\r\n            color: #666;\r\n            margin-bottom: 45px;\r\n}\r\n\r\n.cardbanner .new-label {\r\n    position: relative;\r\n            display: inline-block;\r\n            margin-top: 60px;\r\n            transform: rotate(-10deg);\r\n            width: 100%;\r\n}\r\n\r\n.cardbanner .new-tag {\r\n    background: #4169E1;\r\n            color: white;\r\n            padding: 10px 20px;\r\n            border-radius: 15px;\r\n            position: absolute;  /* 绝对定位 */\r\n            right: 0;  /* 让它紧贴 .new-label 右侧 */\r\n            top: 50%;  /* 垂直居中 */\r\n            transform: translateY(-50%) rotate(30deg);  /* 保持旋转但居中 */\r\n            display: inline-block;\r\n            font-weight: bold;\r\n            font-size: 28px;\r\n            box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);\r\n}\r\n\r\n.cardbanner .new-tag::after {\r\n    content: \"!!\";\r\n    color: white;\r\n    margin-left: 5px;\r\n}\r\n\r\n.cardbanner .footer {\r\n     margin-top: 174px;\r\n            display: flex;\r\n            justify-content: space-between;\r\n            font-size: 20px;\r\n            color: #333;\r\n}\r\n\r\n.cardbanner .footer span {\r\n    margin: 0 10px;\r\n}\r\n\r\n.cardbanner .divider {\r\n    color: #999;\r\n}\r\n      \u003c/style\u003e\r\n      \u003cdiv class=\"cardbanner\"\u003e\r\n        \u003cdiv class=\"header\"\u003e\r\n          \u003cdiv class=\"logo\"\u003e{{ .Get \"logo\" }}\u003c/div\u003e\r\n          \u003cdiv class=\"avatar\"\u003e\r\n           \u003cimg class=\"avatar\" src='{{ .Get \"avatar\" }}' alt=\"头像\"\u003e\r\n          \u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"main-title\"\u003e\r\n          {{ .Get \"mainTitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"subtitle\"\u003e\r\n          {{ .Get \"subtitle\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"description\"\u003e\r\n          {{ .Get \"description\" }}\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"new-label\"\u003e\r\n          \u003cdiv class=\"new-tag\"\u003e{{ .Get \"newTagText\" }}\u003c/div\u003e\r\n        \u003c/div\u003e\r\n\r\n        \u003cdiv class=\"footer\"\u003e\r\n        {{ $topics := split (.Get \"footerContent\") \",\" }}\r\n        {{ range $index, $topic := $topics }}\r\n            {{ if gt $index 0 }} \u003cspan class=\"divider\"\u003e|\u003c/span\u003e {{ end }}\r\n            {{ $topic }}\r\n        {{ end }}\r\n        \u003c/div\u003e\r\n      \u003c/div\u003e","example":"{{\u003c cardBanner\r\n    logo=\"不黑学长\"\r\n    avatar=\"/images/avatar.png\"\r\n    mainTitle=\"让完播率\u003e50% (3/3)\"\r\n    subtitle=\"6种文案公式\"\r\n    description=\"爆款拆解/爆款要素/文案结构\"\r\n    newTagText=\"全新整理\"\r\n    footerContent=\"运营技巧,爆款选题,文案写作,数据复盘\"\r\n/\u003e}}","asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/1743726636150-1.jpg","tags":["xhs","小红书"],"width":1080,"height":1440}]}

curl -X GET "http://localhost:1314/api/sc/hash?name=Test2"
{"data":[{"uuid":"4eb05195-5fd6-4978-a6e1-6ba84d833f13","status":"public","namespace":"ShortCode","id":2,"slug":"test2","hash":"32e6e1e134f9cc8f14b05925667c118d19244aebce442d6fecd2ac38cdc97649","timestamp":1744113507000,"updated":1744113687007,"name":"Test2","desc":"Test2 SC","template":"Test2 SC template","example":"SC example","tags":["SC2"],"asset":"/api/uploads/d66e65ad754f15723096c1156d043cbe/2025/04/2.jpg","width":1080,"height":1440}]}


#### 创建 MDFriday Preview

curl -X POST "http://127.0.0.1:1314/api/mdf/preview?type=MDFPreview" \
-F "type=MDFPreview" \
-F "id=-1" \
-F "name=abc" \
-F "size=12345" \
-F "asset=@/Users/weisun/Downloads/site.zip"

curl -X POST "http://127.0.0.1:1314/api/mdf/preview/deploy?type=MDFPreview&id=1" \
-F "type=MDFPreview" \
-F "host_name=MDFriday Preview" 
