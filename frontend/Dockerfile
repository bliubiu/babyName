# frontend/Dockerfile
# ============================================================
# 起名项目前端 - Next.js 静态导出构建（out/ → nginx 静态服务）
# 产物: 精简运行时镜像 (~120MB)
# ============================================================

# ===== 基础镜像 =====
FROM node:22-alpine AS base

# 启用 Corepack 管理 pnpm
# 版本必须固定：pnpm@latest 漂移到 12 后 YAML/锁文件行为变化会突然打破构建
# （2026.10.01 实测 12.4.2 拒绝锁文件外的配置变化）；固定为与开发机一致的
# 11.0.9（lockfileVersion 9.0）
RUN corepack enable && corepack prepare pnpm@11.0.9 --activate

# 设置工作目录
WORKDIR /app

# ===== 依赖安装阶段 =====
FROM base AS deps

# 复制包管理文件 (利用缓存层)
COPY package.json pnpm-lock.yaml* ./
# pnpm-workspace.yaml 携带 allowBuilds（sharp 构建脚本审批）：
# 不复制的话 pnpm 11 strictDepBuilds 下 install 直接失败
# （ERR_PNPM_IGNORED_BUILDS，docs/30 §六）
COPY pnpm-workspace.yaml ./
# package.json 的 postinstall 指向 scripts/hoist-deps.mjs：
# 缺该文件时 pnpm install 直接失败（Cannot find module，docs/30 §六）
COPY scripts ./scripts

# 安装依赖 (包含 devDependencies 用于构建)
# --frozen-lockfile: 确保锁文件不被修改
# --prod=false: 安装 devDependencies (构建需要)
# registry 走 npmmirror：部署环境（国内网络）直连 registry.npmjs.org 实测
# 仅 ~10KiB/s，471 个包要数小时（2026.10.01 容器实测）；与后端构建的
# GOPROXY=goproxy.cn 同一问题的同一解法
RUN pnpm install --frozen-lockfile --prod=false \
    --registry=https://registry.npmmirror.com

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
# 输出: out/（静态导出：HTML + _next/static 资源）
RUN pnpm run build

# ===== 运行阶段 =====
# ★ 项目前端是静态导出（next.config: output:'export', distDir:'out'），
#   产物为 out/ 下的纯静态文件——不存在 .next/standalone/server.js。
#   旧运行时阶段按 standalone 模式复制 /app/.next/standalone，构建必失败
#   （docs/30 §六）。此处仅用 nginx 提供静态服务（健康检查打 :3000），
#   API 请求由外层 nginx 反代到后端，前端走同源相对路径。
FROM nginx:alpine AS runner

COPY --from=builder /app/out /usr/share/nginx/html
COPY nginx-frontend.conf /etc/nginx/conf.d/default.conf

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000 || exit 1
