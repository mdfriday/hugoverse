
- 初始化用户本地环境
	- 帮助用户生成一个新的系统用户
	- 用户创建站点
		- 帮助用户生成Author
			- 一个是主题Author: mdfriday.com。 生成前查询是否已经创建。
			- 一个是用户自己：随机first name, last name。创建后记录在配置信息中。
		- 注册用户选择的主题
			- 查询主题是否已经存在
		- 创建站点

### 创建用户

curl -X POST http://127.0.0.1:1314/api/user \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=abc@qq.com&password=123456"

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=me@sunwei.xyz&password=123456"

curl -X POST http://127.0.0.1:1314/api/login \
-H "Content-Type: application/x-www-form-urlencoded" \
-d "email=user_625216@mdfriday.com&password=123456"

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



#### Search

curl -X GET "http://127.0.0.1:1314/api/search?type=Site&q=title:Dooring" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTA6NDg6MzYuMTQ0MTUyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.cDTG9kJQoXcM00YdwuEXDqfPuH2XQNXvzS86WN7Gc-w"

curl -X GET "http://127.0.0.1:1314/api/search2?type=Language" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTA6MTY6MDAuMTY5NjQ2KzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.p9puX1tjZ5WpaUKtCclzkB9W6qCWVbFmmKJtAlBRV6Y"

curl -X GET "http://127.0.0.1:1314/api/content?type=Image&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTA6NDg6MzYuMTQ0MTUyKzA4OjAwIiwiaWF0IjpudWxsLCJpc3MiOm51bGwsImp0aSI6bnVsbCwibmJmIjpudWxsLCJzdWIiOm51bGwsInVzZXIiOiJtZUBzdW53ZWkueHl6In0.cDTG9kJQoXcM00YdwuEXDqfPuH2XQNXvzS86WN7Gc-w"


curl -X GET "http://127.0.0.1:1314/api/content?type=Image&id=2" \
-H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOm51bGwsImV4cCI6IjIwMjUtMDQtMjdUMTM6MzM6MjkuMzU4OTQrMDg6MDAiLCJpYXQiOm51bGwsImlzcyI6bnVsbCwianRpIjpudWxsLCJuYmYiOm51bGwsInN1YiI6bnVsbCwidXNlciI6ImFiY0BzdW53ZWkueHl6In0.-33W-Z0Epz9Ve8d0n_oLtOW9dw5FzBHZqzAQE2y6EkQ"
