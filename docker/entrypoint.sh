#!/bin/bash
set -e

# 配置路径
CONFIG_TEMPLATE="/app/config/config.yaml"
CONFIG_FILE="/app/config.yaml"

# 从模板生成配置文件（展开环境变量）
echo "Generating config from template..."
envsubst < "${CONFIG_TEMPLATE}" > "${CONFIG_FILE}"
chmod 644 "${CONFIG_FILE}"

# 打印配置信息
echo "Configuration generated:"
echo "  Server mode: ${SERVER_MODE:-release}"
echo "  Database type: ${DB_TYPE:-sqlite}"
echo "  Database path: ${DB_PATH:-/app/data/bakaray.db}"
echo "  Redis host: ${REDIS_HOST:-bakaray-redis}"

# 确保数据目录存在
mkdir -p /app/data /app/logs

# 初始化数据库（如果不存在）
DB_FILE="${DB_PATH:-/app/data/bakaray.db}"
if [ ! -f "$DB_FILE" ]; then
    echo "Creating new database..."
    # 使用 sqlite3 命令初始化
    if command -v sqlite3 &> /dev/null; then
        sqlite3 "$DB_FILE" < /app/migrations/001_init.sql
        echo "Database initialized."
    else
        echo "Warning: sqlite3 command not found, database initialization skipped."
    fi
else
    echo "Database already exists, skipping initialization."
fi

# 启动服务
exec /app/panel
