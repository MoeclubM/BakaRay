<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">套餐购买</h1>
      <v-spacer />
    </div>

    <v-row>
      <v-col v-for="pkg in packages" :key="pkg.id" cols="12" md="6" lg="4">
        <v-card class="h-100">
          <v-card-title class="text-center pt-6">
            <v-icon size="48" color="primary" class="mb-2">mdi-package-variant-closed</v-icon>
            <div>{{ pkg.name }}</div>
          </v-card-title>

          <v-card-text class="text-center">
            <div class="text-h3 font-weight-bold text-primary my-4">
              ¥{{ pkg.price / 100 }}
            </div>

            <v-divider class="my-4" />

            <v-list density="compact" class="bg-transparent">
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon color="success">mdi-arrow-down</v-icon>
                </template>
                <v-list-item-title>入站流量：{{ formatBytes(pkg.traffic) }}</v-list-item-title>
              </v-list-item>
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon color="info">mdi-arrow-up</v-icon>
                </template>
                <v-list-item-title>出站流量：{{ formatBytes(pkg.traffic) }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-card-text>

          <v-card-actions class="pa-4">
            <v-btn
              color="primary"
              variant="flat"
              block
              size="large"
              @click="purchase(pkg)"
              :loading="purchasing === pkg.id"
            >
              立即购买
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="packages.length === 0" class="text-center py-12">
      <v-icon size="64" color="grey">mdi-package-variant-closed</v-icon>
      <div class="text-h6 mt-4 text-grey">暂无套餐</div>
    </div>

    <!-- 支付对话框 -->
    <v-dialog v-model="showPayDialog" max-width="500">
      <v-card>
        <v-card-title>确认购买</v-card-title>
        <v-card-text>
          <div class="text-center py-4">
            <div class="text-h5">{{ selectedPackage?.name }}</div>
            <div class="text-h4 font-weight-bold text-primary mt-2">
              ¥{{ selectedPackage?.price / 100 }}
            </div>
          </div>

          <v-divider class="my-4" />

          <div class="text-subtitle-2 mb-2">选择支付方式</div>
          <v-radio-group v-model="payType">
            <v-radio
              v-for="payment in payments"
              :key="payment.id"
              :value="String(payment.id)"
              :label="payment.name"
            />
          </v-radio-group>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showPayDialog = false">取消</v-btn>
          <v-btn color="primary" @click="createOrder" :loading="creating">确认支付</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { packageAPI, paymentAPI, orderAPI, depositAPI } from '@/api'

const packages = ref([])
const payments = ref([])
const showPayDialog = ref(false)
const selectedPackage = ref(null)
const payType = ref(null)
const purchasing = ref(null)
const creating = ref(false)

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function purchase(pkg) {
  selectedPackage.value = pkg
  payType.value = payments.value[0]?.id != null ? String(payments.value[0].id) : null
  showPayDialog.value = true
}

async function createOrder() {
  if (!selectedPackage.value || !payType.value) return

  creating.value = true
  try {
    // 创建订单
    const orderRes = await orderAPI.create({
      package_id: selectedPackage.value.id,
      pay_type: payType.value
    })

    // 发起支付
    const payRes = await depositAPI.create({
      order_id: orderRes.data.order_id,
      pay_type: payType.value
    })

    // 如果有支付链接，跳转
    if (payRes.data.pay_url) {
      window.location.href = payRes.data.pay_url
    } else {
      showPayDialog.value = false
      // 刷新订单列表
    }
  } catch (error) {
    console.error('Failed to create order:', error)
  } finally {
    creating.value = false
  }
}

async function loadPackages() {
  try {
    const response = await packageAPI.list()
    packages.value = response.data || []
  } catch (error) {
    console.error('Failed to load packages:', error)
  }
}

async function loadPayments() {
  try {
    const response = await paymentAPI.list()
    payments.value = response.data || []
  } catch {
    payments.value = []
  }
}

onMounted(() => {
  loadPackages()
  loadPayments()
})
</script>
