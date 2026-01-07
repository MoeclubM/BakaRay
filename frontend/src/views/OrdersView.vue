<template>
  <div>
    <h1 class="text-h4 mb-6">我的订单</h1>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="orders"
        :loading="loading"
      >
        <template v-slot:item.status="{ item }">
          <v-chip :color="getStatusColor(item.status)" size="small">
            {{ getStatusText(item.status) }}
          </v-chip>
        </template>

        <template v-slot:item.amount="{ item }">
          ¥{{ item.amount / 100 }}
        </template>

        <template v-slot:item.package_id="{ item }">
          {{ getPackageName(item.package_id) }}
        </template>

        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>

        <template v-slot:no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-receipt-text-outline</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无订单</div>
          </div>
        </template>
      </v-data-table>
    </v-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { orderAPI, packageAPI } from '@/api'
import dayjs from 'dayjs'

const orders = ref([])
const packages = ref([])
const loading = ref(false)

const headers = [
  { title: '订单号', key: 'trade_no' },
  { title: '套餐', key: 'package_id' },
  { title: '金额', key: 'amount' },
  { title: '状态', key: 'status' },
  { title: '创建时间', key: 'created_at' }
]

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

function getStatusColor(status) {
  const colors = { pending: 'warning', success: 'success', failed: 'error' }
  return colors[status] || 'grey'
}

function getStatusText(status) {
  const texts = { pending: '待支付', success: '已完成', failed: '已失败' }
  return texts[status] || status
}

function getPackageName(id) {
  const pkg = packages.value.find(p => p.id === id)
  return pkg?.name || '未知套餐'
}

async function loadOrders() {
  loading.value = true
  try {
    const response = await orderAPI.list()
    orders.value = response.data || []
  } catch (error) {
    console.error('Failed to load orders:', error)
  } finally {
    loading.value = false
  }
}

async function loadPackages() {
  try {
    const response = await packageAPI.list()
    packages.value = response.data || []
  } catch {
    packages.value = []
  }
}

onMounted(() => {
  loadOrders()
  loadPackages()
})
</script>
