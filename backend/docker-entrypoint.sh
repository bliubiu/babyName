#!/bin/sh
# ============================================================
# 起名项目后端容器入口脚本
# ============================================================
# 职责：把镜像内置的字库（/app/data-seed）播种到数据卷（/app/data），
#       并把内置配置（/app/config-seed）播种到配置目录（/app/config）。
#
# 为什么需要（docs/29 B12）：compose 把宿主机目录挂载到 /app/data 与 /app/config，
# 挂载点会遮蔽镜像内的同名目录。若镜像不内置字库，容器里的 data/ 就是空的 ——
# 汉字库、诗词典故、姓名统计全部缺失，服务能正常启动却生成不出名字，
# 且没有任何报错。播种用 `cp -n`（no-clobber）：目标已存在的文件一律保留，
# 用户手工更新字库/配置后重启依然生效。
# ============================================================

set -e

SEED_DIR="${DATA_SEED_DIR:-/app/data-seed}"
DATA_DIR="${DATA_DIR:-/app/data}"
CONFIG_SEED_DIR="${CONFIG_SEED_DIR:-/app/config-seed}"
CONFIG_DIR="${CONFIG_DIR:-/app/config}"

if [ -d "$SEED_DIR" ]; then
    mkdir -p "$DATA_DIR"
    # `-n` 不覆盖已存在文件；`|| true` 兜住 busybox cp 在只读卷/权限不足下的退出码
    cp -rn "$SEED_DIR/." "$DATA_DIR/" 2>/dev/null || true
    echo "[entrypoint] 字库已就绪: $DATA_DIR"
else
    echo "[entrypoint] 未找到内置字库目录 $SEED_DIR，跳过播种（如为自定义镜像请自行挂载字库）"
fi

# 配置播种：compose 把宿主 config 目录挂到 /app/config，会遮蔽镜像内文件。
# 宿主目录初始为空 → 配置文件缺失 → viper 全量回落默认值（读取响应头、
# 限流阈值等静默改变），排查时极难定位。故同样内置一份 application.yml 并播种。
if [ -d "$CONFIG_SEED_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    cp -rn "$CONFIG_SEED_DIR/." "$CONFIG_DIR/" 2>/dev/null || true
    echo "[entrypoint] 配置已就绪: $CONFIG_DIR"
else
    echo "[entrypoint] 未找到内置配置目录 $CONFIG_SEED_DIR，跳过配置播种"
fi

# 播种结果校验：字库缺失时服务照样能启动（端口正常、healthz 通过），
# 但生成接口会返回空结果 —— 这类故障最难定位。此处在启动前明确告警，
# 让问题在日志里一眼可见，而不是等到用户反馈「起不出名字」。
if [ ! -f "$DATA_DIR/namer.json" ]; then
    echo "[entrypoint] 警告: $DATA_DIR/namer.json 不存在，汉字库可能未就绪，生成结果将为空。"
    echo "[entrypoint] 提示: 请确认数据卷挂载正确（docker volume inspect），且宿主主机目录对容器用户（UID 1000）可写。"
fi

if [ ! -f "$CONFIG_DIR/application.yml" ]; then
    echo "[entrypoint] 警告: $CONFIG_DIR/application.yml 不存在，将使用内置默认配置。"
    echo "[entrypoint] 提示: 请确认配置目录挂载正确（docker volume inspect），且宿主主机目录对容器用户（UID 1000）可写。"
fi

cd /app
exec ./namer-server "$@"