#!/bin/bash
set -e

# 配置路径
CONFIG_TEMPLATE="/app/config/config.yaml"
CONFIG_FILE="/app/config.yaml"

# 从模板生成配置文件（展开环境变量）
echo "Generating config from template..."
envsubst < "${CONFIG_TEMPLATE}" > "${CONFIG_FILE}"
chmod 644 "${CONFIG_FILE}"
export CONFIG_FILE

# 打印配置信息
echo "Configuration generated:"
echo "  Server mode: ${SERVER_MODE:-release}"
echo "  Database type: ${DB_TYPE:-sqlite}"
echo "  Database path: ${DB_PATH:-/app/data/bakaray.db}"
echo "  Redis host: ${REDIS_HOST:-bakaray-redis}"

# 确保数据目录存在
mkdir -p /app/data /app/logs

# 如果提供初始化账号参数，优先创建初始用户
if [[ -n "${INIT_USERNAME}" && -n "${INIT_PASSWORD}" ]]; then
    echo "Creating initial user ${INIT_USERNAME}..."
    INIT_CMD=(/app/init-account --username "${INIT_USERNAME}" --password "${INIT_PASSWORD}")
    [[ -n "${INIT_ROLE}" ]] && INIT_CMD+=(--role "${INIT_ROLE}")
    [[ -n "${INIT_GROUP}" ]] && INIT_CMD+=(--group "${INIT_GROUP}")
    "${INIT_CMD[@]}"
fi

# 启动服务
exec /app/panel
