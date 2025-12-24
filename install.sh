#!/bin/bash

# BakaRay 安装脚本
# 支持本地部署和 Docker 部署

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}         BakaRay 安装程序${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 生成随机密钥
generate_secret() {
    cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 32
}

# 安装类型选择
echo -e "${YELLOW}请选择安装方式:${NC}"
echo "  1) Docker 部署 (SQLite, 推荐)"
echo "  2) Docker 部署 (外部 MySQL)"
echo "  3) 本地编译部署 (SQLite)"
echo ""
read -p "请输入选项 [1-3]: " INSTALL_TYPE

case $INSTALL_TYPE in
    1|2)
        # Docker 部署
        if [ "$INSTALL_TYPE" = "2" ]; then
            echo ""
            echo -e "${YELLOW}请输入外部 MySQL 连接信息:${NC}"
            read -p "  MySQL 主机地址: " DB_HOST
            read -p "  端口 [3306]: " DB_PORT
            DB_PORT=${DB_PORT:-3306}
            read -p "  用户名: " DB_USER
            read -p "  密码: " DB_PASSWORD
            read -p "  数据库名: " DB_NAME
        fi

        echo ""
        echo -e "${YELLOW}配置安全密钥 (直接回车使用随机生成):${NC}"
        read -p "  JWT Secret: " JWT_SECRET
        [ -z "$JWT_SECRET" ] && JWT_SECRET=$(generate_secret)
        echo -e "  ${GREEN}JWT Secret: $JWT_SECRET${NC}"

        read -p "  Node Secret: " NODE_SECRET
        [ -z "$NODE_SECRET" ] && NODE_SECRET=$(generate_secret)
        echo -e "  ${GREEN}Node Secret: $NODE_SECRET${NC}"

        echo ""
        echo -e "${YELLOW}构建镜像并启动...${NC}"

        # 构建并启动
        if [ "$INSTALL_TYPE" = "2" ]; then
            # 外部 MySQL 模式
            DB_TYPE=mysql \
            DB_HOST="$DB_HOST" \
            DB_PORT="$DB_PORT" \
            DB_USER="$DB_USER" \
            DB_PASSWORD="$DB_PASSWORD" \
            DB_NAME="$DB_NAME" \
            JWT_SECRET="$JWT_SECRET" \
            NODE_SECRET="$NODE_SECRET" \
            docker-compose -f docker-compose.yml -f docker-compose.external.yml up -d --build
        else
            # SQLite 模式
            JWT_SECRET="$JWT_SECRET" \
            NODE_SECRET="$NODE_SECRET" \
            docker-compose up -d --build
        fi

        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}安装完成！${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo -e "${CYAN}访问地址: http://localhost:8500${NC}"
        echo ""
        echo "查看日志: docker-compose logs -f bakaray-panel"
        echo "停止服务: docker-compose down"
        ;;

    3)
        # 本地编译部署
        echo ""
        echo -e "${YELLOW}配置安全密钥 (直接回车使用随机生成):${NC}"
        read -p "  JWT Secret: " JWT_SECRET
        [ -z "$JWT_SECRET" ] && JWT_SECRET=$(generate_secret)
        echo -e "  ${GREEN}JWT Secret: $JWT_SECRET${NC}"

        read -p "  Node Secret: " NODE_SECRET
        [ -z "$NODE_SECRET" ] && NODE_SECRET=$(generate_secret)
        echo -e "  ${GREEN}Node Secret: $NODE_SECRET${NC}"

        # 创建目录
        mkdir -p data logs

        # 生成配置文件
        cat > config.yaml << EOF
# BakaRay 配置文件
# 由 install.sh 自动生成

server:
  host: "0.0.0.0"
  port: "8080"
  mode: "release"

database:
  type: "sqlite"
  path: "data/bakaray.db"

redis:
  host: "localhost"
  port: 6379
  password: "bakaray-redis-password"
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

        echo ""
        echo -e "${GREEN}配置文件已生成: config.yaml${NC}"

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
        echo -e "${CYAN}启动服务: ./bakaray${NC}"
        echo -e "${CYAN}访问地址: http://localhost:8080${NC}"
        echo ""
        echo "默认账号: admin / admin123"
        echo "配置文件: config.yaml"
        echo "数据库文件: data/bakaray.db"
        ;;
    *)
        echo -e "${RED}无效选项${NC}"
        exit 1
        ;;
esac

echo ""
