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

# 复制配置文件
COPY --from=builder /app/config ./config

# 创建必要目录并赋权
RUN mkdir -p /app/data /app/logs /app/tmp && \
    chown -R namer:namer /app

# 切换到非 root 用户
USER namer

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# 暴露端口
EXPOSE 8080

# 启动命令
ENTRYPOINT ["./namer-server"]