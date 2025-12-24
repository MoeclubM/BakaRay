#!/bin/bash

# BakaRay 安装脚本
# 支持 SQLite（默认）、MySQL、PostgreSQL

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}         BakaRay 安装程序${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 创建必要目录
mkdir -p data logs

# 数据库选择
echo -e "${YELLOW}请选择数据库类型:${NC}"
echo "  1) SQLite (内置，无需配置，推荐)"
echo "  2) MySQL / MariaDB (外部数据库)"
echo "  3) PostgreSQL (外部数据库)"
echo ""
read -p "请输入选项 [1-3]: " DB_OPTION

DB_TYPE="sqlite"
DB_PATH="data/bakaray.db"

case $DB_OPTION in
    1)
        echo -e "${GREEN}已选择 SQLite${NC}"
        ;;
    2)
        echo -e "${YELLOW}请输入 MySQL 连接信息:${NC}"
        read -p "  主机地址: " DB_HOST
        read -p "  端口 [3306]: " DB_PORT
        DB_PORT=${DB_PORT:-3306}
        read -p "  用户名: " DB_USER
        read -p "  密码: " DB_PASS
        read -p "  数据库名: " DB_NAME
        DB_TYPE="mysql"
        ;;
    3)
        echo -e "${YELLOW}请输入 PostgreSQL 连接信息:${NC}"
        read -p "  主机地址: " DB_HOST
        read -p "  端口 [5432]: " DB_PORT
        DB_PORT=${DB_PORT:-5432}
        read -p "  用户名: " DB_USER
        read -p "  密码: " DB_PASS
        read -p "  数据库名: " DB_NAME
        DB_TYPE="postgres"
        ;;
    *)
        echo -e "${RED}无效选项，使用默认 SQLite${NC}"
        ;;
esac

echo ""

# JWT 密钥
echo -e "${YELLOW}配置安全密钥:${NC}"
read -p "  JWT Secret (直接回车生成随机密钥): " JWT_SECRET
if [ -z "$JWT_SECRET" ]; then
    JWT_SECRET=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 32)
    echo -e "  ${GREEN}已生成: $JWT_SECRET${NC}"
fi

read -p "  Node Secret (直接回车生成随机密钥): " NODE_SECRET
if [ -z "$NODE_SECRET" ]; then
    NODE_SECRET=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 32)
    echo -e "  ${GREEN}已生成: $NODE_SECRET${NC}"
fi

echo ""

# 生成配置文件
echo -e "${YELLOW}生成配置文件...${NC}"

cat > config.yaml << EOF
# BakaRay 配置文件
# 由 install.sh 自动生成

server:
  host: "0.0.0.0"
  port: "8080"
  mode: "release"

database:
  type: "${DB_TYPE}"
EOF

if [ "$DB_TYPE" = "sqlite" ]; then
    cat >> config.yaml << EOF
  path: "${DB_PATH}"
EOF
else
    cat >> config.yaml << EOF
  host: "${DB_HOST}"
  port: ${DB_PORT}
  username: "${DB_USER}"
  password: "${DB_PASS}"
  name: "${DB_NAME}"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300
EOF
fi

cat >> config.yaml << EOF

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 10

site:
  name: "BakaRay"
  domain: "http://localhost:8080"
  node_secret: "${NODE_SECRET}"
  node_report_interval: 30

jwt:
  secret: "${JWT_SECRET}"
  expiration: 86400
EOF

echo -e "${GREEN}配置文件已生成: config.yaml${NC}"
echo ""

# 下载依赖
echo -e "${YELLOW}下载 Go 依赖...${NC}"
go mod download

# 构建
echo -e "${YELLOW}编译中...${NC}"
go build -ldflags="-s -w" -o bakaray ./cmd/server

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}安装完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "启动方式:"
echo "  ./bakaray"
echo ""
echo "配置文件: config.yaml"
if [ "$DB_TYPE" = "sqlite" ]; then
    echo "数据库文件: data/bakaray.db"
else
    echo "数据库: ${DB_TYPE}://${DB_HOST}:${DB_PORT}/${DB_NAME}"
fi
echo ""
