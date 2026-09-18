# 起名项目 Docker 构建部署指南

## 目录

1. [项目架构概览](#项目架构概览)
2. [Dockerfile 说明](#dockerfile-说明)
3. [Docker Compose 配置](#docker-compose-配置)
4. [构建与部署流程](#构建与部署流程)
4. [环境变量配置](#环境变量配置)
5. [Nginx 反向代理配置](#nginx-反向代理配置)
6. [常用运维命令](#常用运维命令)
7. [故障排查](#故障排查)

---

## 项目架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                     用户访问                                 │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│  Nginx (反向代理 + SSL终结 + 静态资源缓存)                   │
│  - 端口 80/443                                              │
│  - /api/* → Backend:8080                                    │
│  - /_next/static/* → 缓存 1 年                              │
│  - /* → Frontend:3000                                       │
└─────────────────────┬───────────────────────────────────────┘
                      ▼
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌───────────────┐           ┌───────────────┐
│ Frontend      │           │ Backend       │
│ Next.js:3000  │           │ Go Gin:8080   │
│ 独立输出模式   │           │ 纯静态二进制   │
└───────────────┘           └───────────────┘
        │                           │
        │                    ┌──────┴──────┐
        │                    ▼             ▼
        │              ┌─────────┐   ┌─────────┐
        │              │ SQLite  │   │ 数据目录  │
        │              │ (WAL模式)│   │ /app/data │
        │              └─────────┘   └─────────┘
        └─────────────────────────────────┘
```

**技术栈：**
- **后端**: Go 1.23 + Gin + SQLite (WAL模式) + 纯 Go 实现
- **前端**: Next.js 16.2 + React 19.2 + TypeScript + Tailwind CSS
- **数据持久化**: SQLite (默认) / 内存环形缓存 (冷热数据分离)
- **反向代理**: Nginx (SSL终结、负载均衡、静态资源缓存)

---

## Dockerfile 说明

### 后端 Dockerfile (`backend/Dockerfile`)

**多阶段构建特点：**
- **构建阶段**: 使用 `golang:1.23-alpine`，启用 `CGO_ENABLED=0` 静态编译
- **运行阶段**: 使用 `alpine:3.20`，仅包含二进制文件和必要数据文件
- **镜像大小**: ~15MB (极简)
- **安全性**: 非 root 用户 (UID 1000)、只读根文件系统兼容

```dockerfile
# 关键构建参数
CGO_ENABLED=0 GOOS=linux GOARCH=amd64
go build -ldflags="-s -w -extldflags=-static" -o namer-server
```

**健康检查:**
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1
```

### 前端 Dockerfile (`frontend/Dockerfile`)

**Next.js 独立输出模式:**
- 启用 `output: 'standalone'` 生成精简的 `server.js` + 最小依赖
- **构建阶段**: 完整 Node.js 环境编译
- **运行阶段**: 仅包含 `server.js` + `.next/static` + 最小 `node_modules`
- **镜像大小**: ~120MB

```javascript
// next.config.js 关键配置
output: 'standalone',
```

---

## Docker Compose 配置

### 文件结构
```
├── docker-compose.yml          # 生产环境 (含 Nginx)
├── docker-compose.dev.yml      # 开发环境 (热重载)
├── .env.production             # 生产环境变量 (不提交 Git)
├── .env.development            # 开发环境变量
└── nginx/
    ├── nginx.conf              # 主配置
    └── conf.d/                 # 站点配置
```

### 生产环境 (`docker-compose.yml`)

**服务拓扑:**
```
nginx:80/443 (反向代理)
    ├── frontend:3000 (Next.js)
    └── backend:8080 (Gin API)
```

**关键配置点:**
- **健康检查**: 每服务独立配置，`depends_on` 配合 `condition: service_healthy`
- **数据持久化**: `./backend/data:/app/data` (SQLite WAL 模式需文件锁支持)
- **日志管理**: `json-file` 驱动，单文件 10MB，保留 3 个
- **重启策略**: `unless-stopped` (容器异常退出自动重启)

### 开发环境 (`docker-compose.dev.yml`)

**热重载支持:**
```yaml
backend:
  volumes:
    - ./backend:/app          # 源码挂载
    - /app/tmp                # 避免覆盖容器内编译产物
  command: air -c .air.toml   # Go 热重载

frontend:
  volumes:
    - ./frontend:/app
    - /app/node_modules       # 保护依赖目录
    - /app/.next              # 保护构建缓存
  command: pnpm run dev       # Next.js 热重载
```

**启动开发环境:**
```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

---

## 构建与部署流程

### 1. 本地开发构建

```bash
# 后端
cd backend
docker build -t namer-backend:dev .

# 前端
cd frontend
docker build -t namer-frontend:dev .

# 验证镜像
docker images | grep namer
```

### 2. 生产环境多架构构建推送

```bash
# 创建 buildx 构建器 (首次)
docker buildx create --name multiarch --use --bootstrap

# 后端多架构构建推送
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t your-registry.com/namer-backend:v1.0.0 \
  -t your-registry.com/namer-backend:latest \
  --push \
  ./backend

# 前端多架构构建推送
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t your-registry.com/namer-frontend:v1.0.0 \
  -t your-registry.com/namer-frontend:latest \
  --push \
  ./frontend
```

### 3. 一键部署脚本

```bash
#!/bin/bash
# deploy.sh
set -e

VERSION=${1:-latest}
REGISTRY=${2:-your-registry.com}

echo "🚀 部署版本: $VERSION"

# 1. 拉取最新镜像
docker compose pull

# 2. 数据库迁移 (如需要)
# docker compose run --rm backend ./namer-server migrate

# 3. 滚动重启 (零停机)
docker compose up -d --remove-orphans

# 4. 健康检查等待
echo "⏳ 等待服务就绪..."
sleep 10

# 5. 验证部署
curl -f http://localhost/healthz || exit 1
curl -f http://localhost/api/healthz || exit 1

echo "✅ 部署成功: $VERSION"
```

**使用方法:**
```bash
chmod +x deploy.sh
./deploy.sh v1.2.0 your-registry.com
```

### 4. CI/CD 集成 (GitHub Actions)

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    tags: ['v*']

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ secrets.REGISTRY_URL }}
          username: ${{ secrets.REGISTRY_USER }}
          password: ${{ secrets.REGISTRY_PASSWORD }}
      
      - name: Build and Push Backend
        uses: docker/build-push-action@v5
        with:
          context: ./backend
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ secrets.REGISTRY_URL }}/namer-backend:${{ github.ref_name }}
            ${{ secrets.REGISTRY_URL }}/namer-backend:latest
      
      - name: Build and Push Frontend
        uses: docker/build-push-action@v5
        with:
          context: ./frontend
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ${{ secrets.REGISTRY_URL }}/namer-frontend:${{ github.ref_name }}
            ${{ secrets.REGISTRY_URL }}/namer-frontend:latest
      
      - name: Deploy to Server
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.SERVER_HOST }}
          username: ${{ secrets.SERVER_USER }}
          key: ${{ secrets.SERVER_SSH_KEY }}
          script: |
            cd /opt/namer
            ./deploy.sh ${{ github.ref_name }} ${{ secrets.REGISTRY_URL }}
```

---

## 环境变量配置

### 生产环境 (`.env.production`)

```bash
# ===== 通用 =====
COMPOSE_PROJECT_NAME=namer
TZ=Asia/Shanghai

# ===== 后端 =====
GIN_MODE=release
PORT=8080
DATA_DIR=/app/data
LOG_LEVEL=info
LOG_DIR=/app/logs

# 数据库 (SQLite 默认，可选 PostgreSQL)
# DSN=postgres://namer:password@postgres:5432/namer?sslmode=disable

# Redis (可选，用于缓存/会话)
# REDIS_ADDR=redis:6379
# REDIS_PASSWORD=
# REDIS_DB=0

# JWT 认证
JWT_SECRET=your-32-character-secret-key-here
JWT_EXPIRE=24h

# 邮件/通知 (可选)
# SMTP_HOST=smtp.example.com
# SMTP_PORT=587
# SMTP_USER=namer@example.com
# SMTP_PASSWORD=

# ===== 前端 =====
NODE_ENV=production
NEXT_PUBLIC_API_BASE=https://api.your-domain.com
NEXT_PUBLIC_WS_BASE=wss://api.your-domain.com
NEXT_PUBLIC_APP_NAME=起名工具
NEXT_PUBLIC_APP_VERSION=1.0.0

# ===== Nginx =====
NGINX_SSL_EMAIL=admin@your-domain.com
NGINX_DOMAIN=your-domain.com
```

### 开发环境 (`.env.development`)

```bash
COMPOSE_PROJECT_NAME=namer-dev
GIN_MODE=debug
PORT=8080
DATA_DIR=/app/data
LOG_LEVEL=debug

NODE_ENV=development
NEXT_PUBLIC_API_BASE=http://localhost:8080
NEXT_PUBLIC_WS_BASE=ws://localhost:8080
```

**使用方式:**
```bash
# 生产环境
docker compose --env-file .env.production up -d

# 开发环境
docker compose --env-file .env.development -f docker-compose.yml -f docker-compose.dev.yml up -d
```

---

## Nginx 反向代理配置

### 核心功能
- **SSL 终结**: Let's Encrypt 证书自动续期
- **HTTP → HTTPS 强制跳转**
- **API 代理**: `/api/*` → `backend:8080`
- **WebSocket 支持**: `/ws` 升级协议
- **静态资源缓存**: `/_next/static/*` 缓存 1 年
- **安全头**: CSP、HSTS、X-Frame-Options 等
- **限流**: API 限流 (可选)

### 证书自动续期 (Certbot)

```bash
# 手动申请证书
docker run --rm -it \
  -v ./certbot/conf:/etc/letsencrypt \
  -v ./certbot/www:/var/www/certbot \
  certbot/certbot certonly --webroot \
  --webroot-path=/var/www/certbot \
  -d your-domain.com -d www.your-domain.com \
  --email admin@your-domain.com --agree-tos --no-eff-email

# 自动续期 (cron)
0 3 * * * docker run --rm -v ./certbot/conf:/etc/letsencrypt -v ./certbot/www:/var/www/certbot certbot/certbot renew --quiet
```

---

## 常用运维命令

### 服务管理
```bash
# 启动/停止/重启
docker compose up -d                    # 后台启动
docker compose down                     # 停止并移除容器
docker compose restart backend          # 重启单个服务
docker compose up -d --scale backend=2  # 扩容 (需共享存储)

# 查看状态
docker compose ps                       # 服务状态
docker compose ps -a                    # 包含停止的容器
```

### 日志查看
```bash
# 实时日志
docker compose logs -f backend --tail=100
docker compose logs -f frontend --tail=100
docker compose logs -f nginx --tail=50

# 查看特定时间范围
docker compose logs --since="2024-01-15T10:00:00" backend

# 导出日志
docker compose logs backend > backend_$(date +%Y%m%d).log
```

### 容器调试
```bash
# 进入容器 Shell
docker compose exec backend sh
docker compose exec frontend sh
docker compose exec nginx sh

# 查看进程
docker compose top backend

# 查看资源使用
docker stats --no-stream
```

### 数据备份与恢复

```bash
# 备份 SQLite 数据库 (含 WAL/SHM)
docker compose exec backend tar czf - /app/data | gzip > backup_$(date +%Y%m%d_%H%M%S).tar.gz

# 备份配置文件
tar czf config_backup_$(date +%Y%m%d).tar.gz backend/config frontend/public

# 恢复数据
gunzip -c backup_20240115_120000.tar.gz | docker compose exec -T backend tar xzf - -C /
docker compose restart backend
```

### 版本回滚

```bash
# 查看镜像历史
docker images your-registry.com/namer-backend

# 标记回滚版本
docker tag your-registry.com/namer-backend:v1.1.0 your-registry.com/namer-backend:latest

# 重新部署
docker compose up -d backend

# 验证
curl -f http://localhost/api/healthz
```

### 清理资源

```bash
# 清理未使用镜像/容器/网络
docker system prune -f

# 深度清理 (含卷)
docker system prune -af --volumes

# 清理构建缓存
docker builder prune -af
```

---

## 故障排查

### 1. 后端启动失败

**现象**: `backend` 容器反复重启

**排查步骤:**
```bash
# 1. 查看日志
docker compose logs backend --tail=50

# 2. 常见原因:
# - 端口冲突: lsof -i :8080
# - 数据目录权限: ls -la backend/data/
# - 配置文件缺失: ls backend/config/
# - 数据库锁定: 确保无其他进程占用 SQLite 文件

# 3. 进入容器调试
docker compose run --rm backend sh -c "ls -la /app/data && ./namer-server -version"
```

### 2. 前端构建失败

**现象**: `frontend` 构建阶段报错

**排查步骤:**
```bash
# 1. 查看构建日志
docker compose logs frontend --tail=100

# 2. 常见原因:
# - 依赖安装失败: 检查 pnpm-lock.yaml 是否提交
# - 类型检查失败: 运行 pnpm run lint / pnpm run typecheck
# - 内存不足: 增加 Docker 内存限制 (Settings > Resources > Memory > 4GB)

# 3. 本地验证构建
cd frontend && pnpm run build
```

### 3. Nginx 代理错误

**现象**: 502 Bad Gateway / 504 Gateway Timeout

**排查步骤:**
```bash
# 1. 检查上游服务
docker compose ps
curl -f http://backend:8080/healthz  # 在 nginx 容器内测试

# 2. 检查 Nginx 配置语法
docker compose exec nginx nginx -t

# 3. 查看 Nginx 错误日志
docker compose logs nginx | grep error

# 4. 常见问题:
# - upstream 名称拼写错误 (backend vs frontend)
# - 端口不匹配 (8080 vs 3000)
# - 网络不通 (需同一 docker network)
```

### 4. 数据库锁定/损坏

**现象**: `database is locked` / `disk I/O error`

**排查步骤:**
```bash
# 1. 检查 WAL 模式文件
ls -la backend/data/
# 应存在: namer.db, namer.db-wal, namer.db-shm

# 2. 强制清理锁文件 (仅在确无其他进程时)
docker compose down
rm -f backend/data/namer.db-wal backend/data/namer.db-shm
docker compose up -d backend

# 3. 检查磁盘空间
df -h /opt/namer/backend/data

# 4. SQLite 完整性检查
docker compose exec backend sqlite3 /app/data/namer.db "PRAGMA integrity_check;"
```

### 5. SSL 证书问题

**现象**: 浏览器提示证书无效 / 过期

**排查步骤:**
```bash
# 1. 检查证书有效期
docker compose exec nginx openssl x509 -in /etc/letsencrypt/live/your-domain.com/fullchain.pem -text -noout | grep "Not After"

# 2. 手动续期
docker run --rm -it \
  -v $(pwd)/certbot/conf:/etc/letsencrypt \
  -v $(pwd)/certbot/www:/var/www/certbot \
  certbot/certbot renew --force-renewal

# 3. 重新加载 Nginx
docker compose exec nginx nginx -s reload
```

### 6. 性能问题

**现象**: 响应缓慢 / CPU/内存占用高

**排查步骤:**
```bash
# 1. 资源监控
docker stats --no-stream

# 2. Go pprof 分析 (后端)
# 开启: 在代码中 import _ "net/http/pprof"
# 访问: http://localhost:8080/debug/pprof/
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# 3. Node.js 性能分析 (前端)
# 启动时添加: NODE_OPTIONS="--inspect=0.0.0.0:9229"
# Chrome DevTools 连接: chrome://inspect

# 4. 数据库慢查询
# SQLite: EXPLAIN QUERY PLAN SELECT ...
# 添加索引优化
```

---

## 附录：完整文件清单

```
项目根目录/
├── Docker部署指南.md              # 本文档
├── docker-compose.yml             # 生产环境编排
├── docker-compose.dev.yml         # 开发环境编排
├── .env.production                # 生产环境变量 (gitignore)
├── .env.development               # 开发环境变量
├── deploy.sh                      # 一键部署脚本
├── backend/
│   ├── Dockerfile                 # 后端多阶段构建
│   ├── .dockerignore              # 构建忽略文件
│   ├── cmd/server/main.go         # 服务入口
│   ├── config/                    # 配置文件目录
│   └── data/                      # SQLite 数据目录 (运行时挂载)
├── frontend/
│   ├── Dockerfile                 # 前端多阶段构建
│   ├── .dockerignore              # 构建忽略文件
│   ├── next.config.js             # Next.js 配置 (output: standalone)
│   ├── package.json
│   └── pnpm-lock.yaml
└── nginx/
    ├── nginx.conf                 # Nginx 主配置
    ├── conf.d/
    │   └── default.conf           # 站点配置
    └── certbot/                   # Let's Encrypt 证书目录
        ├── conf/
        └── www/
```

---

## 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| 1.0.0 | 2024-09-16 | 初始版本：完整 Docker 构建部署指南 |

---

*文档维护: 请在每次部署流程变更时同步更新本文档*