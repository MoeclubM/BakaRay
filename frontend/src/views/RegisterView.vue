<template>
  <div class="auth-container">
    <v-card class="auth-card" max-width="400">
      <v-card-title class="text-h5 text-center pt-6">
        <v-icon size="48" color="primary" class="mb-2">mdi-account-plus</v-icon>
        <div>注册 BakaRay</div>
      </v-card-title>

      <v-card-text>
        <v-form ref="formRef" @submit.prevent="handleRegister">
          <v-text-field
            v-model="form.username"
            label="用户名"
            prepend-inner-icon="mdi-account"
            :rules="usernameRules"
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
            :rules="passwordRules"
            variant="outlined"
            class="mb-2"
          />

          <v-text-field
            v-model="form.confirmPassword"
            label="确认密码"
            :type="showPassword ? 'text' : 'password'"
            prepend-inner-icon="mdi-lock-check"
            :rules="confirmPasswordRules"
            variant="outlined"
            class="mb-2"
          />

          <v-text-field
            v-model="form.inviteCode"
            label="邀请码（可选）"
            prepend-inner-icon="mdi-ticket"
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
            注册
          </v-btn>
        </v-form>

        <div class="text-center mt-4">
          <span class="text-grey">已有账号？</span>
          <router-link to="/login" class="ml-1">立即登录</router-link>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const formRef = ref(null)
const loading = ref(false)
const error = ref(null)
const showPassword = ref(false)

const form = ref({
  username: '',
  password: '',
  confirmPassword: '',
  inviteCode: ''
})

const usernameRules = [
  v => !!v || '请输入用户名',
  v => v.length >= 3 || '用户名至少3个字符',
  v => v.length <= 20 || '用户名最多20个字符',
  v => /^[a-zA-Z0-9_]+$/.test(v) || '用户名只能包含字母、数字和下划线'
]

const passwordRules = [
  v => !!v || '请输入密码',
  v => v.length >= 6 || '密码至少6个字符'
]

const confirmPasswordRules = [
  v => !!v || '请确认密码',
  v => v === form.value.password || '两次输入的密码不一致'
]

async function handleRegister() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  loading.value = true
  error.value = null

  const success = await authStore.register(
    form.value.username,
    form.value.password,
    form.value.inviteCode
  )

  if (success) {
    router.push('/login')
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
