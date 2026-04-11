<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">充值中心</h1>
    </div>

    <v-row>
      <!-- 账户余额卡片 -->
      <v-col cols="12" md="4">
        <v-card>
          <v-card-text class="text-center py-8">
            <v-icon size="64" color="primary">mdi-wallet</v-icon>
            <div class="text-h3 mt-4 font-weight-bold">{{ formatCents(userBalance) }}</div>
            <div class="text-grey mt-2">当前余额</div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- 充值金额选择 -->
      <v-col cols="12" md="8">
        <v-card>
          <v-card-title>选择充值金额</v-card-title>
          <v-card-text>
            <v-row>
              <v-col v-for="amount in presetAmounts" :key="amount" cols="6" sm="4" md="3">
                <v-btn
                  block
                  variant="outlined"
                  :color="selectedAmount === amount ? 'primary' : undefined"
                  :class="{ 'bg-primary-lighten-5': selectedAmount === amount }"
                  @click="selectedAmount = amount"
                  size="large"
                >
                  {{ formatYuan(amount) }}
                </v-btn>
              </v-col>
              <v-col cols="6" sm="4" md="3">
                <v-text-field
                  v-model.number="customAmount"
                  label="自定义"
                  prefix="¥"
                  type="number"
                  min="1"
                  hide-details
                  @focus="selectedAmount = null"
                />
              </v-col>
            </v-row>

            <v-divider class="my-6" />

            <div class="text-subtitle-1 mb-4">选择支付方式</div>

            <v-alert
              v-if="paymentMethods.length === 0"
              type="warning"
              variant="tonal"
            >
              暂无可用支付渠道，请先在后台启用支付配置。
            </v-alert>

            <v-row v-else>
              <v-col v-for="method in paymentMethods" :key="method.id" cols="6" sm="4">
                <v-card
                  variant="outlined"
                  :class="{ 'border-primary': selectedPaymentId === method.id }"
                  @click="selectedPaymentId = method.id"
                  class="pa-4 cursor-pointer text-center"
                >
                  <v-icon size="32" color="primary">mdi-credit-card-outline</v-icon>
                  <div class="mt-2 text-body-2">{{ method.name }}</div>
                  <div class="text-caption text-grey mt-1">{{ method.pay_type || method.provider }}</div>
                </v-card>
              </v-col>
            </v-row>

            <v-btn
              color="primary"
              size="large"
              block
              class="mt-6"
              :loading="submitting"
              :disabled="!canSubmit"
              @click="handleSubmit"
            >
              确认支付 {{ formatYuan(actualAmount) }}
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- 充值记录 -->
    <v-card class="mt-6">
      <v-card-title>充值记录</v-card-title>
      <v-data-table
        :headers="headers"
        :items="paymentHistory"
        :loading="loading"
      >
        <template v-slot:item.amount="{ item }">
          <span class="text-success font-weight-bold">+{{ formatCents(item.amount) }}</span>
        </template>

        <template v-slot:item.status="{ item }">
          <v-chip :color="getStatusColor(item.status)" size="small">
            {{ getStatusText(item.status) }}
          </v-chip>
        </template>

        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>

        <template v-slot:no-data>
          <div class="text-center py-8 text-grey">
            <v-icon size="48">mdi-receipt-text-outline</v-icon>
            <div class="mt-2">暂无充值记录</div>
          </div>
        </template>
      </v-data-table>
    </v-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { paymentAPI, depositAPI, orderAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useSnackbar } from '@/composables/useSnackbar'
import dayjs from 'dayjs'

const authStore = useAuthStore()
const { showSnackbar } = useSnackbar()

const loading = ref(false)
const submitting = ref(false)
const selectedAmount = ref(10)
const customAmount = ref(null)
const selectedPaymentId = ref(null)
const paymentHistory = ref([])
const paymentMethods = ref([])

const presetAmounts = [5, 10, 20, 50, 100, 200]
const userBalance = computed(() => authStore.user?.balance || 0)

const headers = [
  { title: '订单号', key: 'trade_no', width: 180 },
  { title: '金额', key: 'amount', width: 100 },
  { title: '支付方式', key: 'pay_type', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', key: 'created_at', width: 180 }
]

const actualAmount = computed(() => {
  if (customAmount.value !== null && customAmount.value > 0) {
    return customAmount.value
  }
  return selectedAmount.value || 0
})

const canSubmit = computed(() => {
  return actualAmount.value > 0 && selectedPaymentId.value
})

function formatYuan(amount) {
  if (!amount) return '¥0.00'
  return '¥' + Number(amount).toFixed(2)
}

function formatCents(amount) {
  if (!amount) return '¥0.00'
  return '¥' + (Number(amount) / 100).toFixed(2)
}

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

function getStatusColor(status) {
  const colors = { pending: 'warning', success: 'success', failed: 'error' }
  return colors[status] || 'grey'
}

function getStatusText(status) {
  const texts = { pending: '待支付', success: '已支付', failed: '失败' }
  return texts[status] || status
}

async function handleSubmit() {
  if (!canSubmit.value) return

  submitting.value = true
  try {
    const res = await depositAPI.create({
      amount: Math.round(Number(actualAmount.value) * 100),
      payment_id: selectedPaymentId.value
    })

    if (res.code === 0 && res.data?.pay_url) {
      // 跳转到支付页面
      window.open(res.data.pay_url, '_blank')
      showSnackbar('正在前往支付...', 'success')
    } else {
      showSnackbar(res.message || '发起支付失败', 'error')
    }
  } catch (error) {
    console.error('Deposit failed:', error)
    showSnackbar(error.message || '支付失败', 'error')
  } finally {
    submitting.value = false
  }
}

async function loadData() {
  loading.value = true
  try {
    await authStore.fetchProfile()
    const [methodsRes, ordersRes] = await Promise.all([
      paymentAPI.list(),
      orderAPI.list()
    ])
    paymentMethods.value = methodsRes.data || []
    if (!selectedPaymentId.value && paymentMethods.value.length > 0) {
      selectedPaymentId.value = paymentMethods.value[0].id
    }
    paymentHistory.value = (ordersRes.data || []).filter((item) => !item.package_id)
  } catch (error) {
    console.error('Failed to load data:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>
