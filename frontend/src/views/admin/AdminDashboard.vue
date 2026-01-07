<template>
  <div>
    <h1 class="text-h4 mb-6">概览</h1>

    <v-row>
      <v-col cols="12" sm="6" md="3">
        <v-card>
          <v-card-text class="d-flex align-center">
            <v-avatar color="primary" size="48" class="mr-4">
              <v-icon>mdi-account-group</v-icon>
            </v-avatar>
            <div>
              <div class="text-h5">{{ stats.user_count }}</div>
              <div class="text-body-2 text-medium-emphasis">用户总数</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card>
          <v-card-text class="d-flex align-center">
            <v-avatar color="success" size="48" class="mr-4">
              <v-icon>mdi-server-network</v-icon>
            </v-avatar>
            <div>
              <div class="text-h5">{{ stats.node_count }}</div>
              <div class="text-body-2 text-medium-emphasis">节点总数</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card>
          <v-card-text class="d-flex align-center">
            <v-avatar color="info" size="48" class="mr-4">
              <v-icon>mdi-receipt</v-icon>
            </v-avatar>
            <div>
              <div class="text-h5">{{ stats.order_count }}</div>
              <div class="text-body-2 text-medium-emphasis">订单总数</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card>
          <v-card-text class="d-flex align-center">
            <v-avatar color="warning" size="48" class="mr-4">
              <v-icon>mdi-currency-cny</v-icon>
            </v-avatar>
            <div>
              <div class="text-h5">¥{{ stats.total_revenue }}</div>
              <div class="text-body-2 text-medium-emphasis">总收入</div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row class="mt-4">
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>最近订单</v-card-title>
          <v-card-text>
            <v-list>
              <v-list-item v-for="order in recentOrders" :key="order.trade_no">
                <template v-slot:prepend>
                  <v-icon :color="order.status === 'success' ? 'success' : 'warning'">
                    {{ order.status === 'success' ? 'mdi-check-circle' : 'mdi-clock' }}
                  </v-icon>
                </template>
                <v-list-item-title>{{ order.package_name || '套餐购买' }}</v-list-item-title>
                <v-list-item-subtitle>{{ order.created_at }}</v-list-item-subtitle>
                <template v-slot:append>
                  <span class="text-primary">¥{{ order.amount }}</span>
                </template>
              </v-list-item>
              <v-list-item v-if="recentOrders.length === 0">
                <v-list-item-title class="text-center text-medium-emphasis">暂无订单</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>节点状态</v-card-title>
          <v-card-text>
            <v-list>
              <v-list-item v-for="node in nodes" :key="node.id">
                <template v-slot:prepend>
                  <v-icon :color="node.status === 'online' ? 'success' : 'error'">
                    mdi-circle
                  </v-icon>
                </template>
                <v-list-item-title>{{ node.name }}</v-list-item-title>
                <v-list-item-subtitle>{{ node.host }}</v-list-item-subtitle>
                <template v-slot:append>
                  <v-chip size="small" :color="node.status === 'online' ? 'success' : 'error'">
                    {{ node.status === 'online' ? '在线' : '离线' }}
                  </v-chip>
                </template>
              </v-list-item>
              <v-list-item v-if="nodes.length === 0">
                <v-list-item-title class="text-center text-medium-emphasis">暂无节点</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'

const stats = ref({
  user_count: 0,
  node_count: 0,
  order_count: 0,
  total_revenue: 0
})

const recentOrders = ref([])
const nodes = ref([])

onMounted(async () => {
  try {
    const [statsRes, ordersRes, nodesRes] = await Promise.all([
      api.adminAPI.stats.overview(),
      api.adminAPI.orders.list({ limit: 5 }),
      api.nodeAPI.list({ limit: 5 })
    ])

    if (statsRes.code === 0) {
      stats.value = statsRes.data || {}
    }
    if (ordersRes.code === 0) {
      recentOrders.value = ordersRes.data || []
    }
    if (nodesRes.code === 0) {
      nodes.value = nodesRes.data || []
    }
  } catch (error) {
    console.error('获取数据失败:', error)
  }
})
</script>
