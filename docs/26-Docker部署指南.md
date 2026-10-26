# 起名项目 Docker 构建部署指南

> 适用版本：2026.10.26 起（SSR 全栈，已移除 Next.js 前端容器）
>
> 本文档只描述当前真实部署链路。历史 Next.js / pnpm / `frontend/` 相关内容
> 已全部废弃，不要再按旧流程构建或排查。

## 目录

1. [架构概览](#架构概览)
2. [Dockerfile 说明](#dockerfile-说明)
3. [Docker Compose 配置](#docker-compose-配置)
4. [构建与部署](#构建与部署)
5. [环境变量](#环境变量)
6. [Nginx 反向代理](#nginx-反向代理)
7. [局域网部署](#局域网部署)
8. [常用运维命令](#常用运维命令)
9. [故障排查](#故障排查)
10. [文件清单](#文件清单)

---

## 架构概览

```
用户访问
   │
   ▼
Nginx（反向代理 / TLS 终结 / 限流 / 安全头）
   │  所有页面与 API
   ▼
Backend（Go 1.25 + Gin）
   ├── SSR 页面：html/template（templates/ 内嵌）
   ├── JSON API：/api/v1/*
   └── 静态资源：/assets/tailwind.css（go:embed）
   │
   ├── SQLite（WAL，/app/data/namer.db）
   └── 字库（/app/data 下 JSON + 康熙笔画 CSV）
```

- **没有独立前端容器**：页面由 Go 服务端模板直接渲染，无 Node/pnpm 构建产物。
- **Tailwind CSS 自托管**：构建产物 `backend/internal/infrastructure/web/static/tailwind.css`
  通过 `go:embed` 编入二进制，运行期无 CDN 依赖。
- **单容器后端**：SSR 页面与 JSON API 共用同一服务、同一数据卷。
- 旧 `/api/v1/*` JSON 接口保留，供移动端或外部系统调用。

---

## Dockerfile 说明

### 构建上下文必须是仓库根

`backend/Dockerfile` 现在依赖仓库根 `data/` 字库作为镜像内置种子，
因此 `docker build` / compose 的构建上下文必须是仓库根：

```bash
# 正确
docker build -f backend/Dockerfile -t namer-backend:dev .

# 错误（拿不到根 data/，运行阶段 COPY /app/data 会失败）
docker build -t namer-backend:dev ./backend
```

镜像内路径：

| 阶段 | 路径 | 说明 |
| --- | --- | --- |
| builder | `/app` | Go 源码 + `/app/data` 字库 |
| runtime | `/app/namer-server` | 静态链接二进制 |
| runtime | `/app/config-seed/application.yml` | 配置种子 |
| runtime | `/app/data-seed/` | 字库种子 |
| runtime | `/app/config`、`/app/data`、`/app/logs` | 运行期挂载点 |

### 多阶段构建

- **builder**：`golang:1.25.10-alpine`，`CGO_ENABLED=0` 静态编译。
- **runtime**：`alpine:3.20`，仅含二进制、证书、时区与非 root 用户（UID 1000）。
- 构建参数：`-ldflags="-s -w -extldflags=-static"`。

### 启动时播种

`backend/docker-entrypoint.sh` 在启动前执行：

1. `/app/data-seed/*` → `/app/data/`（已存在则保留卷内版本）；
2. `/app/config-seed/*` → `/app/config/`（已有配置不覆盖）；
3. 校验 `namer.json` 与 `application.yml`，缺失时打印明确警告；
4. `exec ./namer-server`。

compose 把宿主目录挂到 `/app/data`、`/app/config`，**挂载会遮蔽镜像内目录**，
因此必须依赖播种机制；若宿主目录属主不是 UID 1000，播种会失败并打印警告。

---

## Docker Compose 配置

### 生产环境（`docker-compose.yml`）

服务拓扑：

```
nginx（80/443）
   └── backend:8080（SSR + API）
          ├── /app/data   ← /data/namer/backend/data
          ├── /app/logs   ← /data/namer/backend/logs
          └── /app/config ← /data/namer/backend/config
```

关键点：

- `backend.build.context: .`、`dockerfile: backend/Dockerfile`；
- `depends_on` 只等待 backend 健康；
- `NAME_MODE=api`：SSR 不需要静态目录，避免容器内创建已废弃的 `frontend/out`；
- `NAME_DATABASE_PATH=/app/data/namer.db`：数据库必须落在挂载卷内；
- `NAME_SERVER_HOST=0.0.0.0`：默认 `localhost` 会导致 nginx 502；
- 所有配置环境变量使用 `NAME_` 前缀（viper `AutomaticEnv`）。

### 开发环境（`docker-compose.dev.yml`）

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml \
  --env-file .env.development up -d
```

- 使用 builder 阶段镜像（含 Go 工具链）容器内 `go run` 运行；
- 源码挂载 `./backend:/app`，字库挂载 `./data:/app/data`；
- `CONFIG_DIR=/app`：直接读取源码目录下的 `backend/application.yml`；
- 已无 `frontend` 服务与 `:3000` 端口。

---

## 构建与部署

### 一键部署（推荐）

```bash
./deploy.sh
```

脚本行为：检查文件 → 备份 `/data/namer/backend/data` 与 `config` →
`docker compose up -d --build --remove-orphans` → 轮询
`/healthz`、`/readyz`、`/` → 失败自动回滚。

### 手动部署

```bash
# 生产环境变量（从 .env.example 复制后修改）
cp .env.example .env.production

# 构建并启动
docker compose --env-file .env.production up -d --build

# 验证
curl -f http://localhost/healthz
curl -f http://localhost/readyz
curl -f http://localhost/ | head
```

### 只重建后端

```bash
docker compose --env-file .env.production up -d --build backend
```

### 镜像分发（可选）

若采用「构建机推镜像、服务器拉镜像」模式，请自行在 compose 中增加
`image:` 并用 `docker buildx build --platform linux/amd64,linux/arm64`
推送。注意构建上下文仍必须是仓库根。

---

## 环境变量

`.env.example`（生产）：

```bash
COMPOSE_PROJECT_NAME=namer
TZ=Asia/Shanghai

GIN_MODE=release
PORT=8080
DATA_DIR=/app/data
LOG_LEVEL=info
LOG_DIR=/app/logs
```

- `PORT` / `LOG_LEVEL` / `LOG_DIR` / `DATA_DIR` 由 compose 转成 `NAME_*` 传给后端；
- `NAME_DATABASE_PATH` 固定为 `/app/data/namer.db`，不要改到容器可写层；
- 不再有 `NEXT_PUBLIC_*` 构建期变量。

---

## Nginx 反向代理

主配置：`nginx/nginx.conf`

| 路径 | 行为 |
| --- | --- |
| `/api/` | 代理 backend，100 r/s，突发 200 |
| `/api/names/generate` | 生成接口单独限流 10 r/min |
| `/assets/` | 代理 backend 的自托管 Tailwind CSS，不记访问日志 |
| `/healthz`、`/readyz` | 代理 backend，5s 超时，不记日志 |
| `/ws` | WebSocket 升级 |
| `/` | SSR 页面，代理 backend |
| 其他 | HTTP→HTTPS 跳转、ACME、隐藏文件拒绝 |

安全头与 CSP：

```nginx
script-src 'self' 'unsafe-inline';
style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
font-src 'self' data: https://fonts.gstatic.com;
```

- Tailwind 已自托管，CSP 不再放行 CDN，也不再需要 `'unsafe-eval'`；
- 字体仍从 Google Fonts 外链，若需完全离线，需改为本地字体并同步收紧 CSP。

### 证书自动续期（Certbot）

```bash
docker run --rm -it \
  -v /data/certbot/conf:/etc/letsencrypt \
  -v /data/certbot/www:/var/www/certbot \
  certbot/certbot certonly --webroot \
  --webroot-path=/var/www/certbot \
  -d your-domain.com -d www.your-domain.com \
  --email admin@your-domain.com --agree-tos --no-eff-email

# 每日续期
0 3 * * * docker run --rm -v /data/certbot/conf:/etc/letsencrypt \
  -v /data/certbot/www:/var/www/certbot \
  certbot/certbot renew --quiet
```

---

## 局域网部署

### HTTP（IP 直访）

```bash
docker compose --env-file .env.production \
  -f docker-compose.yml -f docker-compose.lan.yml up -d --build
```

- 纯 HTTP :80，`server_name _` + `default_server`；
- 无证书/ACME/HSTS；
- 页面、API、`/assets/` 全部反代 backend；
- 只有 nginx 发布宿主端口，backend 8080 仅在容器网络内。

访问：

```text
页面   http://<虚拟机IP>/
API    http://<虚拟机IP>/api/v1/...
探活   http://<虚拟机IP>/healthz
就绪   http://<虚拟机IP>/readyz
```

### HTTPS（自签名）

```bash
mkdir -p nginx/lan-certs
openssl req -x509 -nodes -newkey rsa:2048 -days 825 \
  -keyout nginx/lan-certs/server.key -out nginx/lan-certs/server.crt \
  -subj "/CN=namer.lan" \
  -addext "subjectAltName=IP:192.168.10.180,DNS:namer.lan"

docker compose --env-file .env.production \
  -f docker-compose.yml -f docker-compose.lan-https.yml up -d --build
```

证书文件不入 Git；换机后重新生成即可。

---

## 常用运维命令

```bash
# 状态与日志
docker compose ps
docker compose logs -f backend --tail=200
docker compose logs -f nginx --tail=100

# 重启 / 重建
docker compose restart backend
docker compose --env-file .env.production up -d --build backend

# 进入容器
docker compose exec backend sh
docker compose exec nginx sh

# 资源占用
docker stats --no-stream
```

### 数据备份与恢复

```bash
# 备份（宿主路径与 compose 卷一致）
tar czf namer-data-$(date +%Y%m%d).tar.gz -C /data/namer/backend data config

# 恢复
docker compose down
tar xzf namer-data-20261026.tar.gz -C /data/namer/backend
docker compose --env-file .env.production up -d
```

SQLite 使用 WAL，备份时建议先停后端，或同时打包
`namer.db`、`namer.db-wal`、`namer.db-shm`。

---

## 故障排查

### 1. 构建失败：`COPY /app/data: not found`

构建上下文不是仓库根。使用：

```bash
docker build -f backend/Dockerfile .
# 或
docker compose build backend
```

### 2. 后端反复重启

```bash
docker compose logs backend --tail=100
```

重点检查：

- `/app/data` 权限：宿主目录属主应为 UID 1000；
- `/app/config/application.yml` 是否存在（入口脚本会播种）；
- SQLite 是否可写；
- 物料缺失会 **拒绝启动**（fail-fast），不会静默返回空结果。

### 3. 启动后收藏/历史为空

- 确认 `NAME_DATABASE_PATH=/app/data/namer.db` 且 `/app/data` 已挂卷；
- 显式配置 `database.driver=memory` 时数据只在内存，重启即丢（日志会警告）；
- SQLite 打开失败现在会直接拒绝启动，不再静默降级。

### 4. Nginx 502 / 404

```bash
docker compose exec nginx nginx -t
docker compose exec nginx wget -qO- http://backend:8080/healthz
```

常见原因：

- 后端未监听 `0.0.0.0`；
- nginx 配置仍残留 `upstream frontend`（当前版本必须只反代 backend）；
- backend 健康检查未通过导致 nginx 未启动。

### 5. Tailwind 样式缺失

```bash
# 模板新增 Tailwind 类后重建 CSS（仓库根执行）
npx --yes tailwindcss@3.4.3 \
  -c scripts/tailwind.config.js \
  -i scripts/tailwind.input.css \
  -o backend/internal/infrastructure/web/static/tailwind.css --minify

# 确认页面引用的是自托管地址
curl -I http://localhost/assets/tailwind.css
```

### 6. SQLite 锁 / I/O 错误

```bash
# 停服务后检查卷内文件
ls -la /data/namer/backend/data
# 应存在 namer.db、namer.db-wal、namer.db-shm
```

不要在容器内用 `sqlite3` CLI（运行镜像未安装）；可在宿主机对卷内数据库执行
完整性检查：

```bash
sqlite3 /data/namer/backend/data/namer.db "PRAGMA integrity_check;"
```

### 7. SSL 证书问题

```bash
docker compose exec nginx openssl x509 \
  -in /etc/letsencrypt/live/your-domain.com/fullchain.pem -text -noout | grep "Not After"
docker compose exec nginx nginx -s reload
```

---

## 文件清单

```
项目根/
├── .dockerignore                 # 构建上下文规则（保留 backend/ 与 data/）
├── docker-compose.yml            # 生产编排：nginx + backend
├── docker-compose.dev.yml        # 开发编排：backend 热重载
├── docker-compose.lan.yml        # 局域网 HTTP 变体
├── docker-compose.lan-https.yml  # 局域网 HTTPS 变体
├── deploy.sh                     # 一键部署
├── .env.example
├── data/                         # 字库（构建时烘焙为 /app/data-seed）
├── backend/
│   ├── Dockerfile                # 需仓库根构建上下文
│   ├── docker-entrypoint.sh      # 字库/配置播种
│   ├── application.yml           # 本地配置（容器由环境变量覆盖）
│   └── internal/infrastructure/web/static/tailwind.css
└── nginx/
    ├── nginx.conf                # 公网 HTTPS 主配置
    ├── nginx-lan.conf            # 局域网 HTTP
    ├── nginx-lan-https.conf      # 局域网 HTTPS
    └── conf.d/                   # 站点扩展（compose 挂载）
```

---

## 版本历史

| 版本 | 日期 | 变更说明 |
| --- | --- | --- |
| 2026.10.26 | 2026-10-26 | SSR 全栈部署：移除 frontend 容器、构建上下文改为仓库根、CSP 收紧、SQLite 失败 fail-fast、自托管 Tailwind |
| 1.0.0 | 2024-09-16 | 初始版本（Next.js 前后端分离，已废弃） |

---

*文档维护：每次部署链路变更时同步更新本文档。*
