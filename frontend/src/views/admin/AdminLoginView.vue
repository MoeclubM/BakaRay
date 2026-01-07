<template>
  <div class="admin-auth-container">
    <v-card class="admin-auth-card" max-width="420">
      <v-card-title class="text-h5 text-center pt-6">
        <v-icon size="48" color="error" class="mb-2">mdi-shield-crown</v-icon>
        <div class="text-error">BakaRay Admin</div>
        <div class="text-subtitle-2 text-grey mt-1">管理后台登录</div>
      </v-card-title>

      <v-card-text>
        <v-form ref="formRef" @submit.prevent="handleLogin">
          <v-text-field
            v-model="form.username"
            label="管理员用户名"
            prepend-inner-icon="mdi-account"
            :rules="[v => !!v || '请输入用户名']"
            variant="outlined"
            density="comfortable"
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
            density="comfortable"
            class="mb-2"
          />

          <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
            {{ error }}
          </v-alert>

          <v-btn
            type="submit"
            color="error"
            block
            size="large"
            :loading="loading"
          >
            管理后台登录
          </v-btn>
        </v-form>

        <div class="text-center mt-4">
          <v-btn variant="text" to="/login" size="small" color="primary">
            <v-icon left>mdi-arrow-left</v-icon>
            返回前台
          </v-btn>
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
    if (!authStore.isAdmin) {
      error.value = '权限不足，仅管理员可访问'
      authStore.logout()
      loading.value = false
      return
    }
    const redirect = route.query.redirect || '/admin'
    router.push(redirect)
  } else {
    error.value = authStore.error || '登录失败'
  }

  loading.value = false
}
</script>

<style scoped>
.admin-auth-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d1f1f 50%, #1a1a1a 100%);
}

.admin-auth-card {
  width: 100%;
  max-width: 420px;
  border-top: 4px solid rgb(var(--v-theme-error));
}
</style>
