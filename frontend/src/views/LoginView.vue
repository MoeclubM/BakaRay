<template>
  <div class="auth-container">
    <v-card class="auth-card" max-width="400">
      <v-card-title class="text-h5 text-center pt-6">
        <v-icon size="48" color="primary" class="mb-2">mdi-rocket-launch</v-icon>
        <div>登录 BakaRay</div>
      </v-card-title>

      <v-card-text>
        <v-form ref="formRef" @submit.prevent="handleLogin">
          <v-text-field
            v-model="form.username"
            label="用户名"
            prepend-inner-icon="mdi-account"
            :rules="[v => !!v || '请输入用户名']"
            variant="outlined"
            class="mb-2"
          />

          <v-text-field
            v-model="form.password"
            label="密码"
            :type="showPassword ? 'text' : 'password'"
            prepend-inner-icon="mdi-lock"
            :append-inner-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
            @click:append-inner="showPassword = !showPassword"
            :rules="[v => !!v || '请输入密码']"
            variant="outlined"
            class="mb-2"
          />

          <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
            {{ error }}
          </v-alert>

          <v-btn
            type="submit"
            color="primary"
            block
            size="large"
            :loading="loading"
          >
            登录
          </v-btn>
        </v-form>

        <div class="text-center mt-4">
          <span class="text-grey">还没有账号？</span>
          <router-link to="/register" class="ml-1">立即注册</router-link>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref(null)
const loading = ref(false)
const error = ref(null)
const showPassword = ref(false)

const form = ref({
  username: '',
  password: ''
})

async function handleLogin() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  loading.value = true
  error.value = null

  const success = await authStore.login(form.value.username, form.value.password)

  if (success) {
    const redirect = route.query.redirect || '/'
    router.push(redirect)
  } else {
    error.value = authStore.error
  }

  loading.value = false
}
</script>

<style scoped>
.auth-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.auth-card {
  width: 100%;
  max-width: 400px;
}
</style>
