#!/usr/bin/env bash
# =============================================================
# oh-my-stock 一键部署
# 用法:
#   ./deploy.sh start     # 首次启动（含构建）
#   ./deploy.sh stop      # 停止
#   ./deploy.sh restart   # 重启
#   ./deploy.sh status    # 查看状态
#   ./deploy.sh logs      # 拉日志
#   ./deploy.sh rebuild   # 重建镜像并重启
#   ./deploy.sh frontend  # 仅重新构建前端（dist 挂载进容器，无需重建镜像）
# =============================================================
set -euo pipefail
cd "$(dirname "$0")"

cmd="${1:-start}"

ensure_env() {
  if [[ ! -f .env ]]; then
    cp .env.example .env
    echo "→ 已生成 .env（请修改密码后重新启动）"
  fi
}

case "$cmd" in
  start)
    ensure_env
    echo "→ 构建前端 dist（容器内 yarn，复用 yarn.lock）"
    docker run --rm -v "$(pwd)/front:/src" -w /src node:20-alpine \
      sh -c "corepack enable && yarn install --frozen-lockfile && yarn build"
    echo "→ 启动所有服务"
    docker compose up -d
    echo "→ 等数据库就绪"
    for i in {1..30}; do
      if docker compose exec -T postgres pg_isready -U "${DB_USER:-postgres}" >/dev/null 2>&1; then
        echo "✅ postgres ready"
        break
      fi
      sleep 1
    done
    echo "→ 初始化物化视图"
    docker compose up init-mv
    echo ""
    echo "✅ 部署完成"
    echo "  前端: http://localhost:5173"
    echo "  后端: http://localhost:3003  /healthz"
    echo "  Swagger: http://localhost:3003/swagger/index.html  （如启用）"
    ;;
  stop)
    echo "→ 停止所有服务"
    docker compose down
    ;;
  restart)
    echo "→ 重启服务"
    docker compose restart
    ;;
  status)
    docker compose ps
    echo "---"
    curl -fsS http://localhost:3003/healthz && echo " backend ok"
    curl -fsS -o /dev/null -w "frontend HTTP %{http_code}\n" http://localhost:5173/
    ;;
  logs)
    docker compose logs -f --tail=200
    ;;
  rebuild)
    echo "→ 强制重建并启动"
    docker compose build --no-cache
    docker compose up -d
    docker compose up init-mv
    ;;
  frontend)
    echo "→ 仅重新构建前端（容器内 yarn）并让 nginx 重新加载"
    docker run --rm -v "$(pwd)/front:/src" -w /src node:20-alpine \
      sh -c "corepack enable && yarn install --frozen-lockfile && yarn build"
    docker compose exec frontend nginx -s reload >/dev/null 2>&1 || docker compose restart frontend
    echo "✅ 前端已更新"
    ;;
  *)
    echo "用法: $0 {start|stop|restart|status|logs|rebuild|frontend}"
    exit 1
    ;;
esac
