<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">订单管理</h1>
      <v-spacer />
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="orders"
        :loading="loading"
        :items-per-page="20"
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

        <template v-slot:item.actions="{ item }">
          <v-menu>
            <template v-slot:activator="{ props }">
              <v-btn icon size="small" variant="text" v-bind="props">
                <v-icon>mdi-dots-vertical</v-icon>
              </v-btn>
            </template>
            <v-list density="compact">
              <v-list-item @click="viewOrder(item)">
                <v-list-item-title>查看详情</v-list-item-title>
              </v-list-item>
              <v-list-item v-if="item.status === 'pending'" @click="updateStatus(item, 'success')">
                <v-list-item-title>标记为已支付</v-list-item-title>
              </v-list-item>
              <v-list-item v-if="item.status === 'pending'" @click="updateStatus(item, 'failed')">
                <v-list-item-title>标记为已失败</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>
        </template>
      </v-data-table>
    </v-card>

    <!-- 订单详情对话框 -->
    <v-dialog v-model="showDetailDialog" max-width="500">
      <v-card v-if="selectedOrder">
        <v-card-title>订单详情</v-card-title>
        <v-card-text>
          <v-list density="compact">
            <v-list-item>
              <v-list-item-title>订单号</v-list-item-title>
              <v-list-item-subtitle>{{ selectedOrder.trade_no }}</v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title>金额</v-list-item-title>
              <v-list-item-subtitle>¥{{ selectedOrder.amount / 100 }}</v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title>状态</v-list-item-title>
              <v-list-item-subtitle>
                <v-chip :color="getStatusColor(selectedOrder.status)" size="small">
                  {{ getStatusText(selectedOrder.status) }}
                </v-chip>
              </v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title>支付方式</v-list-item-title>
              <v-list-item-subtitle>{{ selectedOrder.pay_type || '未知' }}</v-list-item-subtitle>
            </v-list-item>
            <v-list-item>
              <v-list-item-title>创建时间</v-list-item-title>
              <v-list-item-subtitle>{{ formatDate(selectedOrder.created_at) }}</v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showDetailDialog = false">关闭</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { adminAPI } from '@/api'
import dayjs from 'dayjs'

const orders = ref([])
const loading = ref(false)
const showDetailDialog = ref(false)
const selectedOrder = ref(null)

const headers = [
  { title: '订单号', key: 'trade_no' },
  { title: '套餐ID', key: 'package_id' },
  { title: '金额', key: 'amount' },
  { title: '状态', key: 'status' },
  { title: '创建时间', key: 'created_at' },
  { title: '操作', key: 'actions', width: 80 }
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

function viewOrder(order) {
  selectedOrder.value = order
  showDetailDialog.value = true
}

async function updateStatus(order, status) {
  try {
    await adminAPI.orders.updateStatus(order.id, { status })
    order.status = status
  } catch (error) {
    console.error('Failed to update order status:', error)
  }
}

async function loadOrders() {
  loading.value = true
  try {
    const response = await adminAPI.orders.list()
    orders.value = response.data?.list || response.data || []
  } catch (error) {
    console.error('Failed to load orders:', error)
  } finally {
    loading.value = false
  }
}

onMounted(loadOrders)
</script>
