<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">套餐购买</h1>
      <v-spacer />
    </div>

    <v-overlay v-model="loading" contained class="align-center justify-center">
      <v-progress-circular indeterminate size="64" />
    </v-overlay>

    <v-row>
      <v-col v-for="pkg in packages" :key="pkg.id" cols="12" md="6" lg="4">
        <v-card class="h-100">
          <v-card-title class="text-center pt-6">
            <v-icon size="48" color="primary" class="mb-2">mdi-package-variant-closed</v-icon>
            <div>{{ pkg.name }}</div>
          </v-card-title>

          <v-card-text class="text-center">
            <div class="text-h3 font-weight-bold text-primary my-4">
              ¥{{ (pkg.price / 100).toFixed(2) }}
            </div>

            <v-divider class="my-4" />

            <v-list density="compact" class="bg-transparent">
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon color="success">mdi-arrow-down</v-icon>
                </template>
                <v-list-item-title>流量：{{ formatBytes(pkg.traffic) }}</v-list-item-title>
              </v-list-item>
              <v-list-item v-if="pkg.user_group_name">
                <template v-slot:prepend>
                  <v-icon color="info">mdi-account-group</v-icon>
                </template>
                <v-list-item-title>{{ pkg.user_group_name }}</v-list-item-title>
              </v-list-item>
              <v-list-item>
                <template v-slot:prepend>
                  <v-icon :color="pkg.renewable ? 'success' : 'warning'">
                    {{ pkg.renewable ? 'mdi-refresh' : 'mdi-lock' }}
                  </v-icon>
                </template>
                <v-list-item-title>
                  {{ pkg.renewable ? '可续费' : '一次性' }}
                </v-list-item-title>
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
              :disabled="!pkg.visible"
            >
              {{ pkg.visible ? '立即购买' : '已下架' }}
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="packages.length === 0" class="text-center py-12">
      <v-icon size="64" color="grey">mdi-package-variant-closed</v-icon>
      <div class="text-h6 mt-4 text-grey">暂无套餐</div>
    </div>

    <!-- 支付确认对话框 -->
    <v-dialog v-model="showPayDialog" max-width="400">
      <v-card>
        <v-card-title>确认购买</v-card-title>
        <v-card-text>
          <div class="text-center py-4">
            <div class="text-h6">{{ selectedPackage?.name }}</div>
            <div class="text-h5 font-weight-bold text-primary mt-2">
              ¥{{ (selectedPackage?.price / 100).toFixed(2) }}
            </div>
          </div>

          <v-alert
            v-if="!selectedPackage?.renewable && hasPurchased"
            type="warning"
            variant="tonal"
            density="compact"
            class="mt-4"
          >
            您已购买过此套餐，不可再次购买
          </v-alert>

          <v-alert
            v-if="userBalance < (selectedPackage?.price || 0)"
            type="warning"
            variant="tonal"
            density="compact"
            class="mt-4"
          >
            余额不足，请先<a href="/deposit">充值</a>
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showPayDialog = false">取消</v-btn>
          <v-btn
            color="primary"
            @click="createOrder"
            :loading="creating"
            :disabled="userBalance < (selectedPackage?.price || 0) || (!selectedPackage?.renewable && hasPurchased)"
          >
            确认购买
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { packageAPI, orderAPI } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useSnackbar } from '@/composables/useSnackbar'

const authStore = useAuthStore()
const { showSnackbar } = useSnackbar()

const packages = ref([])
const showPayDialog = ref(false)
const selectedPackage = ref(null)
const purchasing = ref(null)
const creating = ref(false)
const loading = ref(false)
const hasPurchased = ref(false)

const userBalance = computed(() => authStore.user?.balance || 0)

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function purchase(pkg) {
  selectedPackage.value = pkg

  // 检查是否已购买过不可续费的套餐
  if (!pkg.renewable) {
    checkPurchased(pkg.id)
  } else {
    hasPurchased.value = false
  }

  showPayDialog.value = true
}

async function checkPurchased(packageId) {
  try {
    const response = await orderAPI.list()
    const orders = response.data || []
    hasPurchased.value = orders.some(
      o => o.package_id === packageId && o.status === 'success'
    )
  } catch {
    hasPurchased.value = false
  }
}

async function createOrder() {
  if (!selectedPackage.value) return

  creating.value = true
  try {
    const orderRes = await orderAPI.create({
      package_id: selectedPackage.value.id,
      pay_type: 'balance'
    })

    if (orderRes.code === 0 && orderRes.data?.status === 'completed') {
      showPayDialog.value = false
      showSnackbar('购买成功！流量已到账', 'success')
      await authStore.fetchProfile()
      // 刷新套餐列表更新购买状态
      loadPackages()
    } else {
      showSnackbar(orderRes.message || '购买失败', 'error')
    }
  } catch (error) {
    console.error('Failed to create order:', error)
    showSnackbar(error.response?.data?.message || error.message || '购买失败', 'error')
  } finally {
    creating.value = false
  }
}

async function loadPackages() {
  loading.value = true
  try {
    const response = await packageAPI.list()
    packages.value = response.data || []
  } catch (error) {
    console.error('Failed to load packages:', error)
    showSnackbar('加载套餐失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadPackages()
})
</script>
