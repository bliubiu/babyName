# frontend/Dockerfile
# ============================================================
# 起名项目前端 - Next.js 多阶段构建 (Standalone 模式)
# 产物: 精简运行时镜像 (~120MB)
# ============================================================

# ===== 基础镜像 =====
FROM node:20-alpine AS base

# 启用 Corepack 管理 pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# 设置工作目录
WORKDIR /app

# ===== 依赖安装阶段 =====
FROM base AS deps

# 复制包管理文件 (利用缓存层)
COPY package.json pnpm-lock.yaml* ./
# package.json 的 postinstall 指向 scripts/hoist-deps.mjs：
# 缺该文件时 pnpm install 直接失败（Cannot find module，docs/30 §六）
COPY scripts ./scripts

# 安装依赖 (包含 devDependencies 用于构建)
# --frozen-lockfile: 确保锁文件不被修改
# --prod=false: 安装 devDependencies (构建需要)
# 允许 sharp 运行构建脚本 (编译原生模块)
RUN pnpm approve-builds sharp && pnpm install --frozen-lockfile --prod=false

# ===== 构建阶段 =====
FROM base AS builder

# 从 deps 阶段复制 node_modules
COPY --from=deps /app/node_modules ./node_modules

# 复制源码
COPY . .

# 设置构建时环境变量
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

# 构建应用
# 输出: .next/standalone (server.js + 最小依赖) + .next/static (静态资源)
RUN pnpm approve-builds sharp && pnpm run build

# ===== 运行阶段 =====
FROM base AS runner

WORKDIR /app

# 运行时环境变量
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# 创建非 root 用户 (UID 1001)
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001

# 复制公共资源
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

# 复制独立输出 (包含 server.js 和精简 node_modules)
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./

# 复制静态资源 (不可变，可长期缓存)
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

# 切换到非 root 用户
USER nextjs

# 暴露端口
EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000 || exit 1

# 启动 Next.js 服务器
CMD ["node", "server.js"]