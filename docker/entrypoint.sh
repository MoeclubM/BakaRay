#!/bin/bash
set -e

# 确保数据目录存在
mkdir -p /app/data /app/logs

# 打印配置信息
echo "Starting BakaRay panel with environment configuration:"
echo "  Database type: ${DB_TYPE:-sqlite}"
echo "  Redis host: ${REDIS_HOST:-localhost}"

# 如果提供初始化账号参数，优先创建初始用户
if [[ -n "${INIT_USERNAME}" && -n "${INIT_PASSWORD}" ]]; then
    echo "Creating initial user ${INIT_USERNAME}..."
    INIT_CMD=(/app/init-account --username "${INIT_USERNAME}" --password "${INIT_PASSWORD}")
    [[ -n "${INIT_ROLE}" ]] && INIT_CMD+=(--role "${INIT_ROLE}")
    [[ -n "${INIT_GROUP}" ]] && INIT_CMD+=(--group "${INIT_GROUP}")
    "${INIT_CMD[@]}"
fi

# 启动后端服务（后台运行）
echo "Starting BakaRay panel..."
/app/panel &
PANEL_PID=$!

# 等待后端服务启动
sleep 2

# 启动 nginx
echo "Starting nginx..."
nginx -g 'daemon off;' &
NGINX_PID=$!

# trap 退出信号，清理进程
cleanup() {
    echo "Shutting down..."
    kill $NGINX_PID 2>/dev/null || true
    kill $PANEL_PID 2>/dev/null || true
    exit 0
}
trap cleanup SIGTERM SIGINT

# 等待进程
wait
