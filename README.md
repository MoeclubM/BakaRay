# BakaRay

Go + Gin 后端 + Vue 3 + Vite + Vuetify 前端的转发管理面板（Panel）。

## 部署方式

面板仅提供 Docker 部署。

### Docker 部署

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
- 规则：TCP/UDP 直接转发、入口/出口隧道规则、targets 管理
- 支付：套餐/订单/充值、epay（彩虹易支付）回调校验
- 管理后台：节点/节点组、用户/用户组、套餐、订单、支付配置、站点配置
