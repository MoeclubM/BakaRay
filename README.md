# BakaRay

Go + Gin 后端 + Vue 3 + Vite + Vuetify 前端的转发管理面板（Panel）。

## 功能概览

- 用户：注册/登录/刷新 Token、个人信息、修改密码、流量统计
- 节点：节点列表/详情、节点心跳/上报、下发规则配置
- 规则：转发规则 CRUD（含 targets、gost/iptables 配置记录）
- 支付：套餐/订单/充值、epay（彩虹易支付）回调校验
- 管理后台：节点/节点组、用户/用户组、套餐、订单、支付配置、站点配置

## 本地运行

1. 复制并编辑环境变量：`cp .env.example .env`
2. 启动后端：`go run ./cmd/server`
3. 启动前端：
   - `cd frontend`
   - `npm run dev`

## Docker

- `docker compose up -d`

Panel 镜像会在构建阶段执行 `frontend` 的 `npm install && npm run build`，并将 `dist` 产物打包到容器中。

