#!/usr/bin/env bash

set -euo pipefail

REPO_OWNER="MoeclubM"
REPO_NAME="BakaRay"
DEFAULT_INSTALL_DIR="/opt/bakaray"
DEFAULT_PANEL_IMAGE="ghcr.io/moeclubm/bakaray/panel"
DEFAULT_PANEL_PORT="8500"
DEFAULT_SERVER_MODE="release"
DEFAULT_DB_TYPE="sqlite"
DEFAULT_DB_PATH="/app/data/bakaray.db"
DEFAULT_REDIS_PASSWORD="bakaray-redis-password"
DEFAULT_NODE_REPORT_INTERVAL="30"
DEFAULT_INIT_USERNAME="admin"
DEFAULT_INIT_ROLE="admin"
DEFAULT_INIT_GROUP="0"

blue='\033[0;34m'
green='\033[0;32m'
yellow='\033[1;33m'
red='\033[0;31m'
nc='\033[0m'

if [[ "${OSTYPE:-}" != linux* ]]; then
    echo -e "${red}此脚本仅支持 Linux。${nc}" >&2
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    echo -e "${red}缺少 curl，请先安装。${nc}" >&2
    exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
    echo -e "${red}缺少 Docker，请先安装并启动 Docker。${nc}" >&2
    exit 1
fi

if docker compose version >/dev/null 2>&1; then
    compose_cmd=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
    compose_cmd=(docker-compose)
else
    echo -e "${red}缺少 Docker Compose，请先安装 docker compose 插件或 docker-compose。${nc}" >&2
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    echo -e "${red}Docker 守护进程不可用，请先启动 Docker。${nc}" >&2
    exit 1
fi

generate_secret() {
    od -An -N20 -tx1 /dev/urandom | tr -d ' \n'
}

resolve_latest_release_tag() {
    curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" \
        | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n 1
}

has_env_key() {
    grep -q "^$1=" "$env_file"
}

set_env_value() {
    local key="$1"
    local value="$2"
    local escaped

    escaped=$(printf '%s' "$value" | sed -e 's/[\/&|]/\\&/g')
    if has_env_key "$key"; then
        sed -i "s|^${key}=.*|${key}=${escaped}|" "$env_file"
    else
        printf '%s=%s\n' "$key" "$value" >>"$env_file"
    fi
}

ensure_env_value() {
    local key="$1"
    local default_value="$2"

    if [[ ${!key+x} ]]; then
        set_env_value "$key" "${!key}"
        return
    fi

    if ! has_env_key "$key"; then
        set_env_value "$key" "$default_value"
    fi
}

read_container_env() {
    local key="$1"

    docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' bakaray-panel 2>/dev/null \
        | sed -n "s/^${key}=//p" \
        | head -n 1
}

read_panel_port_from_container() {
    docker port bakaray-panel 80/tcp 2>/dev/null \
        | head -n 1 \
        | sed -E 's/.*:([0-9]+)$/\1/'
}

write_compose_file() {
    cat >"$compose_file" <<'EOF'
services:
  bakaray-panel:
    image: ${PANEL_IMAGE}:${PANEL_IMAGE_TAG}
    container_name: bakaray-panel
    ports:
      - "${PANEL_PORT:-8500}:80"
    environment:
      - DB_TYPE=${DB_TYPE:-sqlite}
      - DB_PATH=${DB_PATH:-/app/data/bakaray.db}
      - DB_HOST=${DB_HOST:-localhost}
      - DB_PORT=${DB_PORT:-3306}
      - DB_USERNAME=${DB_USERNAME:-root}
      - DB_PASSWORD=${DB_PASSWORD:-}
      - DB_NAME=${DB_NAME:-bakaray}
      - REDIS_HOST=bakaray-redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD:-bakaray-redis-password}
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - SERVER_MODE=${SERVER_MODE:-release}
      - JWT_SECRET=${JWT_SECRET}
      - NODE_SECRET=${NODE_SECRET}
      - NODE_REPORT_INTERVAL=${NODE_REPORT_INTERVAL:-30}
      - INIT_USERNAME=${INIT_USERNAME:-admin}
      - INIT_PASSWORD=${INIT_PASSWORD:-}
      - INIT_ROLE=${INIT_ROLE:-admin}
      - INIT_GROUP=${INIT_GROUP:-0}
    volumes:
      - bakaray_data:/app/data
      - bakaray_logs:/app/logs
    depends_on:
      bakaray-redis:
        condition: service_healthy
    networks:
      - bakaray_net
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://localhost/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  bakaray-redis:
    image: redis:7-alpine
    container_name: bakaray-redis
    command: ["redis-server", "--appendonly", "yes", "--requirepass", "${REDIS_PASSWORD:-bakaray-redis-password}"]
    volumes:
      - bakaray_redis_data:/data
    networks:
      - bakaray_net
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD:-bakaray-redis-password}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

networks:
  bakaray_net:
    name: bakaray_net
    driver: bridge

volumes:
  bakaray_data:
  bakaray_redis_data:
  bakaray_logs:
EOF
}

wait_for_panel_healthy() {
    local attempt

    for attempt in $(seq 1 60); do
        status=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' bakaray-panel 2>/dev/null || true)
        if [[ "$status" == "healthy" ]]; then
            return
        fi
        if [[ "$status" == "unhealthy" ]]; then
            docker logs --tail 100 bakaray-panel || true
            echo -e "${red}面板容器健康检查失败。${nc}" >&2
            exit 1
        fi
        sleep 2
    done

    docker logs --tail 100 bakaray-panel || true
    echo -e "${red}等待面板启动超时。${nc}" >&2
    exit 1
}

requested_version="${BAKARAY_VERSION:-latest}"
install_dir="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
release_tag=""
image_tag=""
existing_container=0
env_created=0
imported_existing_config=0

echo -e "${blue}========================================${nc}"
echo -e "${blue}      BakaRay 一键安装与升级脚本${nc}"
echo -e "${blue}========================================${nc}"

if [[ "$requested_version" == "latest" ]]; then
    release_tag=$(resolve_latest_release_tag)
    if [[ -z "$release_tag" ]]; then
        echo -e "${red}无法获取最新发布版本，请检查网络或 GitHub API 访问情况。${nc}" >&2
        exit 1
    fi
    image_tag="${release_tag#v}"
else
    if [[ "$requested_version" == v* ]]; then
        release_tag="$requested_version"
        image_tag="${requested_version#v}"
    else
        release_tag="v${requested_version}"
        image_tag="$requested_version"
    fi
fi

compose_file="${install_dir}/docker-compose.yml"
env_file="${install_dir}/.env"

mkdir -p "$install_dir"

if docker inspect bakaray-panel >/dev/null 2>&1; then
    existing_container=1
fi

if [[ ! -f "$env_file" ]]; then
    : >"$env_file"
    env_created=1
fi

if [[ "$env_created" == "1" && "$existing_container" == "1" ]]; then
    for key in DB_TYPE DB_PATH DB_HOST DB_PORT DB_USERNAME DB_PASSWORD DB_NAME REDIS_PASSWORD SERVER_MODE JWT_SECRET NODE_SECRET NODE_REPORT_INTERVAL INIT_USERNAME INIT_PASSWORD INIT_ROLE INIT_GROUP; do
        value="$(read_container_env "$key")"
        if [[ -n "$value" ]]; then
            set_env_value "$key" "$value"
        fi
    done

    panel_port="$(read_panel_port_from_container)"
    if [[ -n "$panel_port" ]]; then
        set_env_value "PANEL_PORT" "$panel_port"
    fi

    imported_existing_config=1
fi

set_env_value "PANEL_IMAGE_TAG" "$image_tag"
set_env_value "PANEL_RELEASE_TAG" "$release_tag"
ensure_env_value "PANEL_IMAGE" "$DEFAULT_PANEL_IMAGE"
ensure_env_value "PANEL_PORT" "$DEFAULT_PANEL_PORT"
ensure_env_value "DB_TYPE" "$DEFAULT_DB_TYPE"
ensure_env_value "DB_PATH" "$DEFAULT_DB_PATH"
ensure_env_value "DB_HOST" "localhost"
ensure_env_value "DB_PORT" "3306"
ensure_env_value "DB_USERNAME" "root"
ensure_env_value "DB_PASSWORD" ""
ensure_env_value "DB_NAME" "bakaray"
ensure_env_value "REDIS_PASSWORD" "$DEFAULT_REDIS_PASSWORD"
ensure_env_value "SERVER_MODE" "$DEFAULT_SERVER_MODE"
ensure_env_value "JWT_SECRET" "$(generate_secret)"
ensure_env_value "NODE_SECRET" "$(generate_secret)"
ensure_env_value "NODE_REPORT_INTERVAL" "$DEFAULT_NODE_REPORT_INTERVAL"
ensure_env_value "INIT_USERNAME" "$DEFAULT_INIT_USERNAME"
ensure_env_value "INIT_PASSWORD" "$(generate_secret)"
ensure_env_value "INIT_ROLE" "$DEFAULT_INIT_ROLE"
ensure_env_value "INIT_GROUP" "$DEFAULT_INIT_GROUP"

write_compose_file

echo -e "${yellow}安装目录:${nc} $install_dir"
echo -e "${yellow}目标版本:${nc} $release_tag"

panel_image="$(sed -n 's/^PANEL_IMAGE=//p' "$env_file" | head -n 1)"
echo -e "${yellow}面板镜像:${nc} ${panel_image}:${image_tag}"

"${compose_cmd[@]}" --project-name bakaray --env-file "$env_file" -f "$compose_file" pull
"${compose_cmd[@]}" --project-name bakaray --env-file "$env_file" -f "$compose_file" up -d --remove-orphans

wait_for_panel_healthy

panel_port="$(sed -n 's/^PANEL_PORT=//p' "$env_file" | head -n 1)"
init_username="$(sed -n 's/^INIT_USERNAME=//p' "$env_file" | head -n 1)"
init_password="$(sed -n 's/^INIT_PASSWORD=//p' "$env_file" | head -n 1)"

echo ""
echo -e "${green}安装完成，当前版本 ${release_tag} 已启动。${nc}"
echo -e "${green}访问地址:${nc} http://<服务器IP>:${panel_port}"
echo -e "${green}配置文件:${nc} ${env_file}"
echo -e "${green}查看日志:${nc} cd ${install_dir} && ${compose_cmd[*]} --project-name bakaray --env-file .env -f docker-compose.yml logs -f bakaray-panel"

if [[ "$env_created" == "1" && "$imported_existing_config" == "0" ]]; then
    echo ""
    echo -e "${yellow}首次安装已生成初始管理员账号，请尽快登录后修改密码。${nc}"
    echo "用户名: ${init_username}"
    echo "密码: ${init_password}"
fi

if [[ "$existing_container" == "1" ]]; then
    echo ""
    echo -e "${green}检测到已有部署，已按当前发布版本完成升级。${nc}"
fi
