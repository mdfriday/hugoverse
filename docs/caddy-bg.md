# 1. 后台启动
hugov caddy start

# 2. 添加多个站点
hugov caddy add -domain site1.test -path /web/site1
hugov caddy add -domain site2.test -path /web/site2

# 3. 查看状态
hugov caddy status

# 4. 导出配置（保存所有站点）
hugov caddy export -output /tmp/my-sites.json

# 5. 停止
hugov caddy stop

# 6. 下次启动时恢复所有配置
hugov caddy start -config /tmp/my-sites.json
# ✅ site1.test 和 site2.test 都会自动恢复


## Hosts

echo "127.0.0.1 site.test" | sudo tee -a /etc/hosts