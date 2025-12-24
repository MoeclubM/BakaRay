<template>
  <div class="callback-container">
    <v-card max-width="500" class="text-center">
      <v-card-text class="py-8">
        <v-progress-circular
          v-if="loading"
          indeterminate
          color="primary"
          size="64"
          class="mb-4"
        />

        <v-icon
          v-else-if="success"
          size="64"
          color="success"
          class="mb-4"
        >
          mdi-check-circle
        </v-icon>

        <v-icon
          v-else
          size="64"
          color="error"
          class="mb-4"
        >
          mdi-close-circle
        </v-icon>

        <div class="text-h5 mb-2">
          {{ success ? '支付成功' : '支付失败' }}
        </div>

        <div class="text-grey">
          {{ message }}
        </div>

        <v-btn
          color="primary"
          variant="tonal"
          class="mt-6"
          to="/orders"
        >
          查看订单
        </v-btn>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { depositAPI } from '@/api'

const route = useRoute()
const loading = ref(true)
const success = ref(false)
const message = ref('正在处理...')

onMounted(async () => {
  try {
    const params = route.query
    const response = await depositAPI.callback(params)

    if (response.code === 0) {
      success.value = true
      message.value = response.message || '您的订单已支付成功'
    } else {
      success.value = false
      message.value = response.message || '支付处理失败'
    }
  } catch (error) {
    success.value = false
    message.value = error.message || '支付回调处理失败'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.callback-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}
</style>
