<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">个人中心</h1>
      <v-spacer />
      <v-btn color="primary" variant="tonal" to="/packages">
        <v-icon start>mdi-cart-plus</v-icon>
        购买套餐
      </v-btn>
    </div>

    <v-overlay v-model="loading" contained class="align-center justify-center">
      <v-progress-circular indeterminate size="64" />
    </v-overlay>

    <v-row>
      <v-col cols="12" md="4">
        <v-card>
          <v-card-text class="text-center py-8">
            <v-avatar size="96" color="primary" class="mb-4">
              <span class="text-h4">{{ user?.username?.substring(0, 2).toUpperCase() }}</span>
            </v-avatar>
            <div class="text-h5">{{ user?.username || '未登录' }}</div>
            <div class="text-caption text-medium-emphasis mt-1">
              用户组：{{ user?.user_group_name || '未分配' }}
            </div>

            <v-divider class="my-4" />

            <div class="text-h4 font-weight-bold text-primary">
              {{ formatBytes(user?.traffic_balance || 0) }}
            </div>
            <div class="text-medium-emphasis">剩余流量</div>

            <div class="text-body-2 mt-3">
              余额：¥{{ ((user?.balance || 0) / 100).toFixed(2) }}
            </div>
            <div class="text-body-2 mt-1">
              角色：{{ user?.role === 'admin' ? '管理员' : '普通用户' }}
            </div>
            <div class="text-body-2 mt-1">
              注册时间：{{ formatDate(user?.created_at) }}
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="8">
        <v-card>
          <v-card-title>基本信息</v-card-title>
          <v-card-text>
            <v-form ref="profileFormRef" @submit.prevent="updateProfile">
              <v-text-field
                v-model="profileForm.username"
                label="用户名"
                :rules="[v => !!String(v || '').trim() || '请输入用户名']"
              />

              <v-text-field
                :model-value="user?.user_group_name || '未分配'"
                label="用户组"
                readonly
                class="mt-2"
              />

              <v-text-field
                :model-value="user?.role === 'admin' ? '管理员' : '普通用户'"
                label="账户角色"
                readonly
                class="mt-2"
              />

              <v-btn
                color="primary"
                type="submit"
                :loading="savingProfile"
                class="mt-4"
              >
                保存资料
              </v-btn>
            </v-form>
          </v-card-text>
        </v-card>

        <v-card class="mt-4">
          <v-card-title>修改密码</v-card-title>
          <v-card-text>
            <v-form ref="passwordFormRef" @submit.prevent="changePassword">
              <v-text-field
                v-model="passwordForm.old_password"
                label="当前密码"
                type="password"
                :rules="[v => !!v || '请输入当前密码']"
              />

              <v-text-field
                v-model="passwordForm.new_password"
                label="新密码"
                type="password"
                :rules="[v => String(v || '').length >= 6 || '密码至少 6 位']"
                class="mt-2"
              />

              <v-text-field
                v-model="passwordForm.confirm_password"
                label="确认新密码"
                type="password"
                :rules="[v => v === passwordForm.new_password || '两次输入的密码不一致']"
                class="mt-2"
              />

              <v-btn
                color="warning"
                type="submit"
                :loading="savingPassword"
                class="mt-4"
              >
                修改密码
              </v-btn>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import dayjs from 'dayjs'
import { useAuthStore } from '@/stores/auth'
import { userAPI } from '@/api'
import { useSnackbar } from '@/composables/useSnackbar'

const authStore = useAuthStore()
const { showSnackbar } = useSnackbar()

const user = computed(() => authStore.user)
const loading = ref(false)
const savingProfile = ref(false)
const savingPassword = ref(false)
const profileFormRef = ref(null)
const passwordFormRef = ref(null)

const profileForm = ref({
  username: ''
})

const passwordForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(date) {
  if (!date) return '未知'
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

async function loadProfile() {
  loading.value = true
  try {
    await authStore.fetchProfile()
    profileForm.value.username = user.value?.username || ''
  } finally {
    loading.value = false
  }
}

async function updateProfile() {
  const { valid } = await profileFormRef.value.validate()
  if (!valid) return

  savingProfile.value = true
  try {
    const username = profileForm.value.username.trim()
    const response = await userAPI.updateProfile({ username })
    if (response.code !== 0) throw new Error(response.message || '更新失败')
    await authStore.fetchProfile()
    profileForm.value.username = user.value?.username || username
    showSnackbar('个人资料已更新', 'success')
  } catch (error) {
    showSnackbar(error.response?.data?.message || error.message || '更新失败', 'error')
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  const { valid } = await passwordFormRef.value.validate()
  if (!valid) return

  savingPassword.value = true
  try {
    const response = await userAPI.changePassword({
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    if (response.code !== 0) throw new Error(response.message || '修改失败')
    passwordForm.value = {
      old_password: '',
      new_password: '',
      confirm_password: ''
    }
    showSnackbar('密码修改成功', 'success')
  } catch (error) {
    showSnackbar(error.response?.data?.message || error.message || '修改失败', 'error')
  } finally {
    savingPassword.value = false
  }
}

onMounted(() => {
  loadProfile()
})
</script>
