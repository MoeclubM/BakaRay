# BakaRay

Go + Gin 后端 + Vue 3 + Vite + Vuetify 前端的转发管理面板（Panel）。

## 部署方式

面板仅提供 Docker 部署。

### 一键安装 / 升级

Linux 服务器可直接执行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/MoeclubM/BakaRay/main/install.sh)
```

前置要求：系统已安装并启动 Docker 与 Docker Compose。

脚本默认执行以下动作：

- 获取最新 GitHub Release 版本
- 生成或复用 `/opt/bakaray/.env`
- 使用 `ghcr.io/moeclubm/bakaray/panel:<release>` 镜像启动面板
- 再次执行同一命令时自动升级到最新发布版本

常见自定义方式：

```bash
INSTALL_DIR=/opt/bakaray \
PANEL_PORT=8500 \
INIT_USERNAME=admin \
INIT_PASSWORD='StrongPassword123' \
bash <(curl -fsSL https://raw.githubusercontent.com/MoeclubM/BakaRay/main/install.sh)
```

如果需要固定版本，可指定 `BAKARAY_VERSION`：

```bash
BAKARAY_VERSION=0.1.1 bash <(curl -fsSL https://raw.githubusercontent.com/MoeclubM/BakaRay/main/install.sh)
```

### 手动 Docker 部署

```bash
git clone https://github.com/MoeclubM/BakaRay.git
cd BakaRay
docker compose up -d
```

如果需要外部 MySQL：

```bash
docker compose -f docker-compose.yml -f docker-compose.external.yml up -d
```

## 功能概览

- 用户：注册/登录/刷新 Token、个人信息、修改密码、流量统计
- 节点：节点列表/详情、节点心跳/上报、下发规则配置
- 规则：转发规则 CRUD（含 targets、gost 配置记录）
- 支付：套餐/订单/充值、epay（彩虹易支付）回调校验
- 管理后台：节点/节点组、用户/用户组、套餐、订单、支付配置、站点配置
