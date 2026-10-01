# backend/Dockerfile
# ============================================================
# 起名项目后端 - 多阶段构建
# 产物: 纯静态链接二进制 (~15MB)
# ============================================================

# ===== 构建阶段 =====
FROM golang:1.25-alpine AS builder

# 配置 Go 代理（使用国内镜像加速下载）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

# 安装构建依赖
RUN apk add --no-cache git make gcc musl-dev ca-certificates tzdata

WORKDIR /app

# 利用缓存层：先复制依赖文件下载模块
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建：静态链接、精简二进制
# - CGO_ENABLED=0: 禁用 CGO，生成纯 Go 静态二进制
# - -ldflags="-s -w": 剥离符号表和调试信息
# - -extldflags=-static: 静态链接 libc
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags=-static" \
    -o namer-server ./cmd/server/main.go

# ===== 运行阶段 =====
FROM alpine:3.20 AS runtime

# 安装运行时依赖
# - ca-certificates: HTTPS 证书验证
# - tzdata: 时区数据 (Asia/Shanghai)
# - sqlite-libs: SQLite 动态库 (如使用 CGO SQLite)
RUN apk add --no-cache ca-certificates tzdata sqlite-libs

# 创建非 root 用户 (UID 1000)
RUN addgroup -g 1000 -S namer && \
    adduser -u 1000 -S namer -G namer

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/namer-server .

# 配置文件（播种到 /app/config-seed，由入口脚本铺到 /app/config）
#
# ★ docs/29 B12：原为 `COPY --from=builder /app/config ./config`，但构建上下文
# （backend/）下根本没有 config/ 目录 —— 配置文件实际是 backend/application.yml。
# 该 COPY 会让 `docker build` 直接失败（source path not found），镜像根本构建不出来。
#
# ★ docs/29 B12 补正：直接 COPY 到 ./config/application.yml 仍然是无效的 ——
# compose 用宿主目录挂载 /app/config，挂载点会整体遮蔽镜像内该目录，
# 新部署时宿主 config 目录是空的，配置文件根本不存在，viper 全量回落默认值。
# 故把 application.yml 烘焙为 /app/config-seed/application.yml，
# 由 docker-entrypoint.sh 以 `cp -n` 播种到 /app/config（已有文件不覆盖），
# 与字库播种同一条路径；用户手工改过的配置同样能在重启后保留。
COPY --from=builder /app/application.yml ./config-seed/application.yml

# 内置字库（诗词/汉字/统计数据等 JSON）播种到 /app/data-seed
#
# ★ docs/29 B12：运行阶段原先只复制二进制与配置，/app/data 是空的。
# 而 compose 把宿主机目录挂载到 /app/data —— 挂载会遮蔽镜像内文件，
# 于是容器里 data/ 永远为空，汉字库/诗词/统计数据全部缺失，
# 服务能启动但生成不出名字。改为：字库烘焙进 /app/data-seed，
# 由 docker-entrypoint.sh 首次启动时播种到 /app/data（已存在则保留卷内版本，
# 以支持用户手工更新字库后重启生效）。
COPY --from=builder /app/data ./data-seed

# 复制启动脚本（负责播种字库）
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# 创建必要目录并赋权
RUN mkdir -p /app/data /app/config /app/logs /app/tmp && \
    chown -R namer:namer /app

# 切换到非 root 用户
USER namer

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# 暴露端口
EXPOSE 8080

# 启动命令（经入口脚本播种字库后启动服务）
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]