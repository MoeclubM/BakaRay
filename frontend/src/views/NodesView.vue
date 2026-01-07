<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">节点列表</h1>
      <v-spacer />
    </div>

    <v-overlay v-model="loading" contained class="align-center justify-center">
      <v-progress-circular indeterminate size="64" />
    </v-overlay>

    <v-row>
      <v-col v-for="node in nodes" :key="node.id" cols="12" md="6" lg="4">
        <v-card :class="{ 'border-opacity-50': node.status !== 'online' }">
          <v-card-title class="d-flex align-center">
            <v-icon
              :color="node.status === 'online' ? 'success' : 'error'"
              class="mr-2 pulse"
            >
              mdi-circle
            </v-icon>
            {{ node.name }}
            <v-spacer />
            <v-chip :color="node.status === 'online' ? 'success' : 'error'" size="small">
              {{ node.status === 'online' ? '在线' : '离线' }}
            </v-chip>
          </v-card-title>

          <v-card-text>
            <v-list density="compact" class="bg-transparent">
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon>mdi-map-marker</v-icon>
                </template>
                <v-list-item-title>地区：{{ node.region || '未知' }}</v-list-item-title>
              </v-list-item>
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon>mdi-protocol</v-icon>
                </template>
                <v-list-item-title>协议：{{ (node.protocols || []).join(', ') }}</v-list-item-title>
              </v-list-item>
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon>mdi-clock-outline</v-icon>
                </template>
                <v-list-item-title>最后活跃：{{ formatDate(node.last_seen) }}</v-list-item-title>
              </v-list-item>
            </v-list>

            <!-- 节点探针数据 -->
            <v-expand-transition>
              <div v-if="expandedNode === node.id" class="probe-card">
                <v-divider class="my-2" />
                <div class="text-subtitle-2 mb-2">节点探针</div>
                <div v-if="node.probe" class="probe-content">
                  <!-- CPU -->
                  <div class="probe-item">
                    <span>CPU</span>
                    <v-progress-linear
                      :model-value="node.probe.cpu?.usage_percent || 0"
                      color="primary"
                      height="24"
                      rounded
                      style="width: 140px"
                    >
                      <template v-slot:default>
                        {{ (node.probe.cpu?.usage_percent || 0).toFixed(1) }}%
                      </template>
                    </v-progress-linear>
                  </div>
                  <!-- Memory -->
                  <div class="probe-item">
                    <span>内存</span>
                    <v-progress-linear
                      :model-value="node.probe.memory?.usage_percent || 0"
                      color="info"
                      height="24"
                      rounded
                      style="width: 140px"
                    >
                      <template v-slot:default>
                        {{ (node.probe.memory?.usage_percent || 0).toFixed(1) }}%
                      </template>
                    </v-progress-linear>
                  </div>
                  <!-- Network -->
                  <template v-if="node.probe.network && node.probe.network.length">
                    <div v-for="iface in node.probe.network" :key="iface.name" class="probe-item">
                      <span>{{ iface.name }}</span>
                      <div class="text-caption">
                        <span class="text-success">↓ {{ formatSpeed(iface.rx_speed) }}</span>
                        <span class="ml-2 text-info">↑ {{ formatSpeed(iface.tx_speed) }}</span>
                      </div>
                    </div>
                  </template>
                </div>
                <div v-else class="text-grey text-caption">暂无探针数据</div>
              </div>
            </v-expand-transition>
          </v-card-text>

          <v-card-actions>
            <v-btn
              variant="tonal"
              size="small"
              @click="toggleProbe(node.id)"
            >
              <v-icon start>{{ expandedNode === node.id ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
              {{ expandedNode === node.id ? '收起' : '探针' }}
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="nodes.length === 0" class="text-center py-12">
      <v-icon size="64" color="grey">mdi-server-network-off</v-icon>
      <div class="text-h6 mt-4 text-grey">暂无节点</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { nodeAPI } from '@/api'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const nodes = ref([])
const expandedNode = ref(null)
const loading = ref(false)
let refreshInterval = null

function formatDate(date) {
  if (!date) return '未知'
  return dayjs(date).fromNow()
}

function formatSpeed(bytes) {
  if (!bytes) return '0 B/s'
  const k = 1024
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function toggleProbe(nodeId) {
  expandedNode.value = expandedNode.value === nodeId ? null : nodeId
}

async function loadNodes() {
  try {
    const response = await nodeAPI.list()
    nodes.value = response.data || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
  }
}

onMounted(async () => {
  loading.value = true
  await loadNodes()
  loading.value = false

  // 每秒自动刷新探针数据
  refreshInterval = setInterval(loadNodes, 1000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<style scoped>
.pulse {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.probe-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
}
</style>
