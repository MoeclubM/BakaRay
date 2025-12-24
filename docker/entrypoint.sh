#!/bin/bash
set -e

# 配置路径
CONFIG_TEMPLATE="/app/config/config.yaml"
CONFIG_FILE="/app/config.yaml"

# 从模板生成配置文件（展开环境变量）
echo "Generating config from template..."
envsubst < "${CONFIG_TEMPLATE}" > "${CONFIG_FILE}"
chmod 644 "${CONFIG_FILE}"

# 打印配置信息（不显示敏感信息）
echo "Configuration generated:"
echo "  Server mode: ${SERVER_MODE:-release}"
echo "  Database host: ${DB_HOST:-localhost}"
echo "  Redis host: ${REDIS_HOST:-localhost}"

# 启动服务
exec /app/panel
