# 需求

实现 license 管理系统，可以支持用户创建 sync 服务和 publish 服务。

用户可以在前端激活 license ，就可以知道 license 的类型和有效期。
license 有 Free，Starter，Creator，Pro, Enterprise 不同的类型，有效期通常都是一年。

用户的使用流程是，在前端输入 license , 会发送一个 activate 的请求到后端，验证是否合法，有没有过期， 如果是合法有效的，则会在后端创建一个 couchDB 的数据库。
用户就可以用这个账号进行文件同步了。
同时，我们也会为用户生成一个专属的 WEB SERVER 文件夹，里面可以放用户分享的单篇文章，或者是整个站点，而且每一个都是以独立文件夹存在的，所以我们还可以绑定自定义域名。
