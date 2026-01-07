#!/bin/bash
# BakaRay 集成测试运行脚本
# 在 WSL/Docker 环境下运行此脚本

set -e

echo "========================================"
echo "  BakaRay 集成测试启动器"
echo "========================================"

# 检查是否在 Docker 容器内
if [ -f /.dockerenv ]; then
    echo "[INFO] 检测到 Docker 容器环境"
    IN_DOCKER=1
else
    echo "[INFO] 检测到本地环境"
    IN_DOCKER=0
fi

# 切换到脚本目录
cd "$(dirname "$0")"

# 检查 docker-compose
if ! command -v docker-compose &> /dev/null && ! command -v docker &> /dev/null; then
    echo "[ERROR] docker-compose 未安装"
    exit 1
fi

# 使用 docker compose (新版) 或 docker-compose
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# 停止现有容器
echo "[INFO] 停止现有容器..."
$DOCKER_COMPOSE -f docker-compose.integration.yml down -v 2>/dev/null || true

# 启动服务
echo "[INFO] 启动集成测试环境..."
$DOCKER_COMPOSE -f docker-compose.integration.yml up -d

# 等待服务就绪
echo "[INFO] 等待服务就绪..."
sleep 10

# 检查面板是否就绪
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:8500/api/auth/login | grep -q "code"; then
        echo "[INFO] 面板服务已就绪"
        break
    fi
    echo -n "."
    sleep 2
    ATTEMPT=$((ATTEMPT + 1))
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo "[ERROR] 面板服务未能在30秒内就绪"
    $DOCKER_COMPOSE -f docker-compose.integration.yml logs
    exit 1
fi

echo ""
echo "[INFO] 运行 Go 集成测试..."

# 编译并运行测试
go run gost_e2e_test.go

TEST_RESULT=$?

# 显示日志
echo ""
echo "[INFO] 面板日志:"
docker logs bakaray-integration-panel --tail 20 2>/dev/null || true

# 清理
echo ""
echo "[INFO] 清理测试环境..."
$DOCKER_COMPOSE -f docker-compose.integration.yml down -v

if [ $TEST_RESULT -eq 0 ]; then
    echo ""
    echo "========================================"
    echo "  集成测试通过！"
    echo "========================================"
else
    echo ""
    echo "========================================"
    echo "  集成测试失败"
    echo "========================================"
fi

exit $TEST_RESULT
