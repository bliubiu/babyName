# Docker 部署快速开始

## 📋 前置要求

- Docker 20.10+
- Docker Compose 2.0+ (或 `docker compose` 插件)
- Docker Buildx (多架构构建)
- 4GB+ 可用内存

## 🚀 快速开始

### 1. 克隆项目
```bash
git clone <repository-url>
cd name
```

### 2. 配置环境变量
```bash
# 复制示例配置
cp .env.example .env.production

# 编辑配置 (必须修改域名等)
vim .env.production
```

**必须修改的关键配置：**
- `NGINX_DOMAIN`: 您的域名
- `NGINX_SSL_EMAIL`: Let's Encrypt 证书申请邮箱
- `NEXT_PUBLIC_API_BASE`: 前端调用的 API 地址

> ⚠️ **安全提示（docs/29 B14）**：当前版本后端**没有任何鉴权**（无 JWT、无
> Session、无 API Key），`/api/*` 全部接口公开可读写。`JWT_SECRET` /
> `JWT_EXPIRE` 是历史遗留的死配置，代码里没有任何引用，设置它们**不会**产生
> 任何保护作用。
>
> 因此：**请勿将后端端口直接暴露到公网**。如需公网访问，必须在 Nginx 层
> （`nginx/conf.d/`）或云网关/WAF 上自行加访问控制（IP 白名单、Basic Auth、
> mTLS 等）。若要把鉴权做进应用，请先确认需求范围。

### 3. 申请 SSL 证书 (首次部署)
```bash
# 启动临时 Nginx 验证域名
docker compose up -d nginx

# 申请证书
docker run --rm -it \
  -v $(pwd)/certbot/conf:/etc/letsencrypt \
  -v $(pwd)/certbot/www:/var/www/certbot \
  certbot/certbot certonly --webroot \
  --webroot-path=/var/www/certbot \
  -d your-domain.com -d www.your-domain.com \
  --email admin@your-domain.com --agree-tos --no-eff-email

# 重新加载 Nginx
docker compose exec nginx nginx -s reload
```

### 4. 构建并启动
```bash
# 构建镜像
docker compose build

# 启动服务
docker compose --env-file .env.production up -d

# 查看状态
docker compose ps
docker compose logs -f
```

### 5. 验证部署
```bash
# 健康检查
curl https://your-domain.com/healthz
curl https://your-domain.com/api/healthz

# 访问应用
open https://your-domain.com
```

## 🛠️ 常用命令

| 操作 | 命令 |
|------|------|
| 查看日志 | `docker compose logs -f backend` |
| 重启服务 | `docker compose restart backend` |
| 进入容器 | `docker compose exec backend sh` |
| 查看资源 | `docker stats` |
| 备份数据 | `docker compose exec backend tar czf - /app/data \| gzip > backup_$(date +%Y%m%d).tar.gz` |
| 回滚版本 | `docker tag registry/app:v1.1 registry/app:latest && docker compose up -d` |

## 🔧 开发环境

```bash
# 启动开发环境 (热重载)
docker compose -f docker-compose.yml -f docker-compose.dev.yml --env-file .env.development up -d

# 查看前端热重载日志
docker compose logs -f frontend
```

## 📦 数据备份与恢复

### 备份
```bash
# 完整备份
docker compose exec backend tar czf - /app/data | gzip > backup_$(date +%Y%m%d_%H%M%S).tar.gz

# 仅备份数据库
docker compose exec backend sqlite3 /app/data/namer.db ".backup /app/data/backup_$(date +%Y%m%d).db"
```

### 恢复
```bash
# 停止服务
docker compose down

# 恢复数据
gunzip -c backup_20240115_120000.tar.gz | docker compose exec -T backend tar xzf - -C /

# 重启服务
docker compose up -d
```

## 🚨 故障排查

| 问题 | 排查步骤 |
|------|----------|
| 后端启动失败 | `docker compose logs backend --tail=50` 检查端口冲突、权限、配置 |
| 前端构建失败 | `docker compose logs frontend --tail=100` 检查依赖、内存 |
| 502/504 错误 | 检查上游服务 `curl http://backend:8080/healthz` |
| SSL 证书过期 | `docker run --rm -v ./certbot/conf:/etc/letsencrypt certbot/certbot renew` |
| 数据库锁定 | `rm -f backend/data/namer.db-wal backend/data/namer.db-shm` |

## 📊 监控与维护

### 日志查看
```bash
# 实时日志
docker compose logs -f backend --tail=100

# 特定时间范围
docker compose logs --since="2024-01-15T10:00:00" backend
```

### 性能监控
```bash
# 资源使用
docker stats --no-stream

# Go pprof (后端开启后)
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

### 清理资源
```bash
# 清理未使用资源
docker system prune -f

# 深度清理 (含数据卷，谨慎使用)
docker system prune -af --volumes
```

## 🔄 CI/CD 集成

### GitHub Actions 示例
```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    tags: ['v*']
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ${{ secrets.REGISTRY_URL }}
          username: ${{ secrets.REGISTRY_USER }}
          password: ${{ secrets.REGISTRY_PASSWORD }}
      - uses: docker/build-push-action@v5
        with:
          context: ./backend
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ secrets.REGISTRY_URL }}/namer-backend:${{ github.ref_name }}
      # ... 前端同理
      - name: Deploy
        uses: appleboy/ssh-action@v1
        with:
          script: cd /opt/namer && ./deploy.sh ${{ github.ref_name }} ${{ secrets.REGISTRY_URL }}
```

## 📁 目录结构

```
.
├── docker-compose.yml          # 生产环境编排
├── docker-compose.dev.yml      # 开发环境编排
├── .env.example                # 环境变量示例
├── .env.production             # 生产环境配置 (勿提交)
├── .env.development            # 开发环境配置
├── deploy.sh                   # 一键部署脚本
├── Docker部署指南.md           # 完整部署文档
├── nginx/
│   ├── nginx.conf              # Nginx 主配置
│   └── conf.d/                 # 站点配置
├── certbot/                    # Let's Encrypt 证书
│   ├── conf/
│   └── www/
├── backend/
│   ├── Dockerfile              # 后端构建
│   └── .dockerignore
└── frontend/
    ├── Dockerfile              # 前端构建
    ├── .dockerignore
    └── next.config.js          # output: 'standalone'
```

## 🆘 获取帮助

- 查看完整文档: `Docker部署指南.md`
- 查看服务状态: `docker compose ps`
- 查看实时日志: `docker compose logs -f`
- 进入容器调试: `docker compose exec backend sh`