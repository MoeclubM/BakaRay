<template>
  <div>
    <h1 class="text-h4 mb-6">仪表盘</h1>

    <v-overlay v-model="loading" contained class="align-center justify-center">
      <v-progress-circular indeterminate size="64" />
    </v-overlay>

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
              剩余流量：<span class="text-primary font-weight-bold">{{ formatBytes(user?.traffic_balance || 0) }}</span>
            </div>
            <div v-if="userGroupName" class="text-grey text-caption mt-1">
              用户组：<span class="text-info">{{ userGroupName }}</span>
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
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { nodeAPI, ruleAPI, userAPI } from '@/api'
import { useSnackbar } from '@/composables/useSnackbar'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const authStore = useAuthStore()
const { showSnackbar } = useSnackbar()

const user = computed(() => authStore.user)
const userGroupName = computed(() => {
  const groupId = user.value?.user_group_id
  if (!groupId) return '未分配'
  // 简单处理，实际应从API获取用户组名称
  return groupId === 1 ? '测试用户组' : `用户组 #${groupId}`
})

const nodes = ref([])
const rules = ref([])
const loading = ref(false)

const trafficStats = ref({
  bytes_in: 0,
  bytes_out: 0
})

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(async () => {
  loading.value = true
  try {
    await authStore.fetchProfile()
    const [nodesRes, rulesRes, trafficRes] = await Promise.all([
      nodeAPI.list(),
      ruleAPI.list(),
      userAPI.getTrafficStats({ days: 30 })
    ])
    nodes.value = nodesRes.data || []
    rules.value = rulesRes.data || []
    trafficStats.value = trafficRes.data || { bytes_in: 0, bytes_out: 0 }
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
    showSnackbar(error.response?.data?.message || error.message || '加载数据失败', 'error')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.stat-card {
  text-align: center;
}
</style>
