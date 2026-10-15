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
# 且没有任何报错。播种不覆盖已存在文件：用户手工更新字库/配置后重启依然生效。
#
# ★ docs/30 §六 补正（2026.10.01 容器实测）：busybox cp 对
#   `cp -rn 目录/. 目标/` 这一组合会**静默不复制**（退出码 0、零文件落盘），
#   且 `2>/dev/null || true` 会把「目录不可写」的失败一并吞掉，
#   然后日志照印「已就绪」——故障被伪装成成功。改为逐项 glob 复制 +
#   逐项报错，并以实际文件存在性输出结论。
# ============================================================

set -e

SEED_DIR="${DATA_SEED_DIR:-/app/data-seed}"
DATA_DIR="${DATA_DIR:-/app/data}"
CONFIG_SEED_DIR="${CONFIG_SEED_DIR:-/app/config-seed}"
CONFIG_DIR="${CONFIG_DIR:-/app/config}"

# seed_dir 无覆盖地把 src 下每个条目复制到 dst（已存在则保留目标版本）
seed_dir() {
    src="$1"
    dst="$2"
    label="$3"
    if [ ! -d "$src" ]; then
        echo "[entrypoint] 未找到内置 $label 目录 $src，跳过播种"
        return 0
    fi
    mkdir -p "$dst"
    copied=0
    skipped=0
    failed=0
    for f in "$src"/*; do
        [ -e "$f" ] || continue
        name=$(basename "$f")
        if [ -e "$dst/$name" ]; then
            skipped=$((skipped + 1))
            continue
        fi
        if cp -r "$f" "$dst/" 2>/dev/null; then
            copied=$((copied + 1))
        else
            failed=$((failed + 1))
            echo "[entrypoint] 警告: $label 播种 $name 失败（目标目录不可写？宿主目录属主应为 UID 1000）"
        fi
    done
    echo "[entrypoint] $label 播种: 新增 $copied 项，保留已有 $skipped 项，失败 $failed 项"
}

seed_dir "$SEED_DIR" "$DATA_DIR" "字库"
seed_dir "$CONFIG_SEED_DIR" "$CONFIG_DIR" "配置"

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
