<template>
  <div>
    <h1 class="text-h4 mb-6">数据概览</h1>

    <v-row>
      <v-col cols="12" sm="6" md="3">
        <v-card class="stat-card" color="primary" variant="tonal">
          <v-card-text>
            <div class="d-flex align-center">
              <v-icon size="40" color="primary">mdi-account-group</v-icon>
              <div class="ml-4">
                <div class="text-h4">{{ stats.users }}</div>
                <div class="text-grey">用户总数</div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card class="stat-card" color="success" variant="tonal">
          <v-card-text>
            <div class="d-flex align-center">
              <v-icon size="40" color="success">mdi-server</v-icon>
              <div class="ml-4">
                <div class="text-h4">{{ stats.nodes }}</div>
                <div class="text-grey">节点总数</div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card class="stat-card" color="info" variant="tonal">
          <v-card-text>
            <div class="d-flex align-center">
              <v-icon size="40" color="info">mdi-routes</v-icon>
              <div class="ml-4">
                <div class="text-h4">{{ stats.rules }}</div>
                <div class="text-grey">转发规则</div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card class="stat-card" color="warning" variant="tonal">
          <v-card-text>
            <div class="d-flex align-center">
              <v-icon size="40" color="warning">mdi-cart</v-icon>
              <div class="ml-4">
                <div class="text-h4">{{ stats.orders }}</div>
                <div class="text-grey">今日订单</div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row class="mt-4">
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>节点状态分布</v-card-title>
          <v-card-text>
            <div class="d-flex justify-space-around text-center">
              <div>
                <v-progress-circular
                  :model-value="nodeOnlinePercent"
                  color="success"
                  size="80"
                  width="8"
                >
                  {{ nodeOnlineCount }}
                </v-progress-circular>
                <div class="mt-2">在线</div>
              </div>
              <div>
                <v-progress-circular
                  :model-value="100 - nodeOnlinePercent"
                  color="error"
                  size="80"
                  width="8"
                >
                  {{ nodeOfflineCount }}
                </v-progress-circular>
                <div class="mt-2">离线</div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>快捷操作</v-card-title>
          <v-card-text>
            <v-row>
              <v-col cols="6">
                <v-btn block variant="tonal" color="primary" to="/admin/nodes">
                  <v-icon start>mdi-server-plus</v-icon>
                  添加节点
                </v-btn>
              </v-col>
              <v-col cols="6">
                <v-btn block variant="tonal" color="success" to="/admin/users">
                  <v-icon start>mdi-account-plus</v-icon>
                  添加用户
                </v-btn>
              </v-col>
              <v-col cols="6">
                <v-btn block variant="tonal" color="info" to="/admin/packages">
                  <v-icon start>mdi-package-variant</v-icon>
                  添加套餐
                </v-btn>
              </v-col>
              <v-col cols="6">
                <v-btn block variant="tonal" color="warning" to="/admin/settings">
                  <v-icon start>mdi-cog</v-icon>
                  系统设置
                </v-btn>
              </v-col>
            </v-row>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { adminAPI } from '@/api'

const stats = ref({
  users: 0,
  nodes: 0,
  rules: 0,
  orders: 0
})

const nodes = ref([])

const nodeOnlineCount = computed(() => nodes.value.filter(n => n.status === 'online').length)
const nodeOfflineCount = computed(() => nodes.value.length - nodeOnlineCount.value)
const nodeOnlinePercent = computed(() => {
  if (!nodes.value.length) return 0
  return (nodeOnlineCount.value / nodes.value.length) * 100
})

onMounted(async () => {
  try {
    const [usersRes, nodesRes, rulesRes, ordersRes] = await Promise.all([
      adminAPI.users.list({ limit: 1 }),
      adminAPI.nodes.list(),
      adminAPI.rules.count(),
      adminAPI.orders.list({ limit: 1 })
    ])

    stats.value = {
      users: usersRes.total || 0,
      nodes: (nodesRes.data || []).length,
      rules: rulesRes.data || 0,
      orders: ordersRes.total || 0
    }
    nodes.value = nodesRes.data || []
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
})
</script>

<style scoped>
.stat-card {
  height: 100%;
}
</style>
