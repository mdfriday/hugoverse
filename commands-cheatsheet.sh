#!/bin/bash
# Colima + Hugoverse 快速命令参考

cat << 'EOF'
╔══════════════════════════════════════════════════════════════╗
║          Colima + Hugoverse 命令速查表                      ║
╚══════════════════════════════════════════════════════════════╝

┌─────────────────────────────────────────────────────────────┐
│ 1️⃣  安装 Colima                                             │
└─────────────────────────────────────────────────────────────┘

  brew install colima docker docker-compose

┌─────────────────────────────────────────────────────────────┐
│ 2️⃣  Colima 管理                                             │
└─────────────────────────────────────────────────────────────┘

  colima start                          # 启动（默认配置）
  colima start --cpu 4 --memory 8      # 启动（自定义资源）
  colima stop                           # 停止
  colima restart                        # 重启
  colima status                         # 查看状态
  colima delete                         # 删除（清理所有）
  colima logs                           # 查看日志
  colima ssh                            # SSH 进入虚拟机

┌─────────────────────────────────────────────────────────────┐
│ 3️⃣  Hugoverse 测试                                          │
└─────────────────────────────────────────────────────────────┘

  ./verify-local.sh                     # 快速验证（不启动）
  ./test-docker-local.sh                # 完整测试（启动服务）
  
  make verify-local                     # Makefile 方式
  make test-local                       # Makefile 方式

┌─────────────────────────────────────────────────────────────┐
│ 4️⃣  服务管理（使用 .env.local）                             │
└─────────────────────────────────────────────────────────────┘

  # 启动所有服务
  docker compose --env-file .env.local up -d
  
  # 停止所有服务
  docker compose --env-file .env.local down
  
  # 查看服务状态
  docker compose --env-file .env.local ps
  
  # 重启服务
  docker compose --env-file .env.local restart
  
  # 查看日志（所有服务）
  docker compose --env-file .env.local logs -f
  
  # 查看特定服务日志
  docker compose --env-file .env.local logs -f hugoverse
  docker compose --env-file .env.local logs -f caddy
  docker compose --env-file .env.local logs -f couchdb
  
  # 完全清理（包括数据卷）
  docker compose --env-file .env.local down -v

┌─────────────────────────────────────────────────────────────┐
│ 5️⃣  Makefile 快捷命令                                       │
└─────────────────────────────────────────────────────────────┘

  make up-local                         # 启动服务
  make down-local                       # 停止服务
  make logs-local                       # 查看日志
  make clean-local                      # 完全清理
  make restart                          # 重启服务

┌─────────────────────────────────────────────────────────────┐
│ 6️⃣  Docker 命令                                             │
└─────────────────────────────────────────────────────────────┘

  docker ps                             # 查看运行的容器
  docker ps -a                          # 查看所有容器
  docker images                         # 查看镜像
  docker system df                      # 查看磁盘使用
  docker system prune -a --volumes      # 清理所有未使用资源

┌─────────────────────────────────────────────────────────────┐
│ 7️⃣  进入容器调试                                            │
└─────────────────────────────────────────────────────────────┘

  # 进入 Hugoverse 容器
  docker compose --env-file .env.local exec hugoverse sh
  
  # 进入 Caddy 容器
  docker compose --env-file .env.local exec caddy sh
  
  # 进入 CouchDB 容器
  docker compose --env-file .env.local exec couchdb bash

┌─────────────────────────────────────────────────────────────┐
│ 8️⃣  查看生成的 License                                      │
└─────────────────────────────────────────────────────────────┘

  docker compose --env-file .env.local logs hugoverse | grep "License Key"

┌─────────────────────────────────────────────────────────────┐
│ 9️⃣  重新构建镜像                                            │
└─────────────────────────────────────────────────────────────┘

  # 重新构建所有镜像
  docker compose --env-file .env.local build
  
  # 重新构建特定镜像
  docker compose --env-file .env.local build hugoverse
  docker compose --env-file .env.local build caddy
  
  # 无缓存构建
  docker compose --env-file .env.local build --no-cache

┌─────────────────────────────────────────────────────────────┐
│ 🔟 访问地址                                                 │
└─────────────────────────────────────────────────────────────┘

  Admin 面板:    http://localhost:8080/admin
  API 健康检查:  http://localhost:8080/api/health
  CouchDB UI:    http://localhost:8080/_utils
  
  登录信息:
    Email:     admin@localhost
    Password:  test123456

┌─────────────────────────────────────────────────────────────┐
│ 1️⃣1️⃣  故障排查                                             │
└─────────────────────────────────────────────────────────────┘

  # Colima 状态
  colima status
  
  # Docker daemon 状态
  docker info
  
  # 服务健康检查
  curl http://localhost:8080/api/health
  
  # 查看最近 50 行日志
  docker compose --env-file .env.local logs --tail=50 hugoverse
  
  # 检查端口占用
  lsof -i :8080
  
  # 重启 Colima（解决大部分问题）
  colima restart

┌─────────────────────────────────────────────────────────────┐
│ 1️⃣2️⃣  完整重置（清理所有）                                 │
└─────────────────────────────────────────────────────────────┘

  # 停止并删除所有容器和数据
  docker compose --env-file .env.local down -v
  
  # 删除所有镜像
  docker rmi $(docker images -q 'hugoverse*') 2>/dev/null
  
  # 清理 Docker 系统
  docker system prune -a --volumes
  
  # 重启 Colima
  colima restart
  
  # 重新测试
  ./test-docker-local.sh

┌─────────────────────────────────────────────────────────────┐
│ 1️⃣3️⃣  一行命令完整流程                                     │
└─────────────────────────────────────────────────────────────┘

  # 从零开始
  brew install colima docker docker-compose && \
  colima start --cpu 2 --memory 4 && \
  cd /Users/weisun/github/mdfriday/hugoverse && \
  ./test-docker-local.sh

┌─────────────────────────────────────────────────────────────┐
│ 📚 文档快速访问                                             │
└─────────────────────────────────────────────────────────────┘

  cat COLIMA-QUICKSTART.md             # 一页纸快速开始
  cat DOCKER-CLI.md                    # 完整 Docker CLI 指南
  cat DOCKER-CLI-SUMMARY.md            # 方案总结
  cat START-HERE.md                    # 总体开始指南
  cat TEST-LOCAL.md                    # 本地测试详细说明

╔══════════════════════════════════════════════════════════════╗
║  提示：复制这些命令到你的终端直接使用！                     ║
╚══════════════════════════════════════════════════════════════╝

EOF
