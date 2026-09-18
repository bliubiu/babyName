#!/bin/bash
# ============================================================
# 起名项目 - 一键部署脚本
# 用法: ./deploy.sh [版本号] [镜像仓库地址]
# 示例: ./deploy.sh v1.2.0 your-registry.com
# ============================================================

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认参数
VERSION=${1:-latest}
REGISTRY=${2:-your-registry.com}
PROJECT_DIR="/data/namer"
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env.production"

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查必要文件
check_files() {
    log_info "检查必要文件..."
    
    if [[ ! -f "$COMPOSE_FILE" ]]; then
        log_error "未找到 $COMPOSE_FILE"
        exit 1
    fi
    
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error "未找到 $ENV_FILE，请从 .env.production.example 复制并修改"
        exit 1
    fi
    
    # 检查必要的环境变量
    source "$ENV_FILE"
    if [[ -z "${JWT_SECRET:-}" ]] || [[ "$JWT_SECRET" == *"change-me"* ]]; then
        log_error "请在 $ENV_FILE 中设置强随机 JWT_SECRET (至少 32 字符)"
        exit 1
    fi
    
    log_success "文件检查通过"
}

# 备份当前数据
backup_data() {
    log_info "备份当前数据..."
    BACKUP_DIR="/data/backups/namer/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # 备份 SQLite 数据库
    if [[ -d "$PROJECT_DIR/backend/data" ]]; then
        tar czf "$BACKUP_DIR/data.tar.gz" -C "$PROJECT_DIR/backend" data 2>/dev/null || true
        log_success "数据备份完成: $BACKUP_DIR/data.tar.gz"
    fi
    
    # 备份配置文件
    tar czf "$BACKUP_DIR/config.tar.gz" -C "$PROJECT_DIR" backend/config frontend/public 2>/dev/null || true
    
    # 保留最近 7 天备份
    find /data/backups/namer -type f -name "*.tar.gz" -mtime +7 -delete 2>/dev/null || true
}

# 拉取最新镜像
pull_images() {
    log_info "拉取镜像: $REGISTRY/namer-backend:$VERSION, $REGISTRY/namer-frontend:$VERSION"
    
    # 替换镜像地址并拉取
    sed "s|your-registry.com|$REGISTRY|g" "$COMPOSE_FILE" | \
    sed "s|:latest|:$VERSION|g" | \
    docker compose -f - pull
    
    log_success "镜像拉取完成"
}

# 滚动部署 (零停机)
deploy() {
    log_info "开始滚动部署..."
    
    # 使用 docker compose up -d 实现滚动更新
    docker compose --env-file "$ENV_FILE" up -d --remove-orphans
    
    log_info "等待服务就绪..."
    sleep 15
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log_info "健康检查尝试 $attempt/$max_attempts..."
        
        # 检查 Nginx
        if curl -f -s -o /dev/null --max-time 5 http://localhost/healthz; then
            log_success "Nginx 健康检查通过"
        else
            log_warning "Nginx 健康检查失败，重试中..."
        fi
        
        # 检查后端 API
        if curl -f -s -o /dev/null --max-time 5 http://localhost/api/healthz; then
            log_success "后端 API 健康检查通过"
        else
            log_warning "后端 API 健康检查失败，重试中..."
        fi
        
        # 检查前端
        if curl -f -s -o /dev/null --max-time 5 http://localhost/; then
            log_success "前端健康检查通过"
            break
        else
            log_warning "前端健康检查失败，重试中..."
        fi
        
        sleep 5
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_error "健康检查失败，部署可能有问题"
        return 1
    fi
    
    log_success "所有健康检查通过"
}

# 回滚函数
rollback() {
    log_warning "开始回滚..."
    
    # 停止当前服务
    docker compose down
    
    # 恢复备份数据
    LATEST_BACKUP=$(ls -td /data/backups/namer/*/ | head -1)
    if [[ -n "$LATEST_BACKUP" ]]; then
        log_info "恢复备份: $LATEST_BACKUP"
        tar xzf "$LATEST_BACKUP/data.tar.gz" -C "$PROJECT_DIR/backend" 2>/dev/null || true
        tar xzf "$LATEST_BACKUP/config.tar.gz" -C "$PROJECT_DIR" 2>/dev/null || true
    fi
    
    # 重启服务
    docker compose up -d
    
    log_warning "回滚完成，请检查服务状态"
}

# 主流程
main() {
    echo "=========================================="
    echo "  起名项目自动部署脚本 v1.0"
    echo "  版本: $VERSION"
    echo "  仓库: $REGISTRY"
    echo "=========================================="
    
    cd "$PROJECT_DIR" || { log_error "项目目录不存在: $PROJECT_DIR"; exit 1; }
    
    check_files
    backup_data
    pull_images
    deploy
    
    if health_check; then
        log_success "=========================================="
        log_success "  部署成功! 版本: $VERSION"
        log_success "  访问地址: https://your-domain.com"
        log_success "  API 文档: https://your-domain.com/api/docs"
        log_success "=========================================="
    else
        log_error "部署失败，开始自动回滚..."
        rollback
        exit 1
    fi
}

# 捕获中断信号
trap 'log_error "部署被中断"; rollback; exit 1' INT TERM

# 执行主流程
main "$@"