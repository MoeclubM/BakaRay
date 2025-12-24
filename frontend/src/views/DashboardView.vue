<template>
  <div>
    <h1 class="text-h4 mb-6">仪表盘</h1>

    <!-- 欢迎卡片 -->
    <v-card class="mb-4" color="primary" variant="tonal">
      <v-card-text>
        <div class="d-flex align-center">
          <v-avatar size="64" color="primary" class="mr-4">
            <span class="text-h5">{{ user?.username?.substring(0, 2).toUpperCase() }}</span>
          </v-avatar>
          <div>
            <div class="text-h5">欢迎回来，{{ user?.username }}</div>
            <div class="text-grey">
              剩余流量：<span class="text-primary font-weight-bold">{{ formatBytes(user?.balance || 0) }}</span>
            </div>
          </div>
          <v-spacer />
          <v-btn color="primary" variant="flat" to="/packages">
            <v-icon start>mdi-cart-plus</v-icon>
            购买套餐
          </v-btn>
        </div>
      </v-card-text>
    </v-card>

    <!-- 流量统计 -->
    <v-row>
      <v-col cols="12" md="4">
        <v-card class="stat-card">
          <v-card-text>
            <v-icon size="40" color="success" class="mb-2">mdi-arrow-down-bold-circle</v-icon>
            <div class="stat-value">{{ formatBytes(trafficStats.bytes_in) }}</div>
            <div class="stat-label">本月入站流量</div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="stat-card">
          <v-card-text>
            <v-icon size="40" color="info" class="mb-2">mdi-arrow-up-bold-circle</v-icon>
            <div class="stat-value">{{ formatBytes(trafficStats.bytes_out) }}</div>
            <div class="stat-label">本月出站流量</div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="stat-card">
          <v-card-text>
            <v-icon size="40" color="warning" class="mb-2">mdi-chart-timeline-variant</v-icon>
            <div class="stat-value">{{ formatBytes(trafficStats.bytes_in + trafficStats.bytes_out) }}</div>
            <div class="stat-label">本月总流量</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 规则统计和节点状态 -->
    <v-row class="mt-4">
      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>转发规则</v-card-title>
          <v-card-text>
            <div class="d-flex justify-space-between mb-2">
              <span>活跃规则</span>
              <span class="font-weight-bold text-success">{{ rules.filter(r => r.enabled).length }}</span>
            </div>
            <v-progress-linear
              :model-value="rules.filter(r => r.enabled).length / Math.max(rules.length, 1) * 100"
              color="success"
              height="8"
              rounded
            />
            <div class="d-flex justify-space-between mt-2">
              <span>总规则数</span>
              <span class="font-weight-bold">{{ rules.length }}</span>
            </div>
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" color="primary" to="/rules">查看全部</v-btn>
          </v-card-actions>
        </v-card>
      </v-col>

      <v-col cols="12" md="6">
        <v-card>
          <v-card-title>节点状态</v-card-title>
          <v-card-text>
            <div class="d-flex justify-space-between mb-2">
              <span>在线节点</span>
              <span class="font-weight-bold text-success">{{ nodes.filter(n => n.status === 'online').length }}</span>
            </div>
            <v-progress-linear
              :model-value="nodes.filter(n => n.status === 'online').length / Math.max(nodes.length, 1) * 100"
              color="success"
              height="8"
              rounded
            />
            <div class="d-flex justify-space-between mt-2">
              <span>总节点数</span>
              <span class="font-weight-bold">{{ nodes.length }}</span>
            </div>
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" color="primary" to="/nodes">查看节点</v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <!-- 最近订单 -->
    <v-card class="mt-4">
      <v-card-title class="d-flex align-center">
        <v-icon start>mdi-receipt-text</v-icon>
        最近订单
        <v-spacer />
        <v-btn variant="text" color="primary" to="/orders">查看全部</v-btn>
      </v-card-title>
      <v-data-table
        :headers="orderHeaders"
        :items="recentOrders"
        :items-per-page="5"
        density="compact"
      >
        <template v-slot:item.status="{ item }">
          <v-chip :color="getStatusColor(item.status)" size="small">
            {{ getStatusText(item.status) }}
          </v-chip>
        </template>
        <template v-slot:item.amount="{ item }">
          ¥{{ item.amount / 100 }}
        </template>
        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>
      </v-data-table>
    </v-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { nodeAPI, ruleAPI, orderAPI, userAPI } from '@/api'
import dayjs from 'dayjs'

const authStore = useAuthStore()

const user = computed(() => authStore.user)
const nodes = ref([])
const rules = ref([])
const orders = ref([])

const trafficStats = ref({
  bytes_in: 0,
  bytes_out: 0
})

const recentOrders = computed(() => orders.value.slice(0, 5))

const orderHeaders = [
  { title: '订单号', key: 'trade_no' },
  { title: '金额', key: 'amount' },
  { title: '状态', key: 'status' },
  { title: '创建时间', key: 'created_at' }
]

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

function getStatusColor(status) {
  const colors = {
    pending: 'warning',
    success: 'success',
    failed: 'error'
  }
  return colors[status] || 'grey'
}

function getStatusText(status) {
  const texts = {
    pending: '待支付',
    success: '已完成',
    failed: '已失败'
  }
  return texts[status] || status
}

onMounted(async () => {
  try {
    const [nodesRes, rulesRes, ordersRes, trafficRes] = await Promise.all([
      nodeAPI.list(),
      ruleAPI.list(),
      orderAPI.list(),
      userAPI.getTrafficStats({ days: 30 })
    ])
    nodes.value = nodesRes.data || []
    rules.value = rulesRes.data || []
    orders.value = ordersRes.data || []
    trafficStats.value = trafficRes.data || { bytes_in: 0, bytes_out: 0 }
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  }
})
</script>

<style scoped>
.stat-card {
  text-align: center;
}
</style>
