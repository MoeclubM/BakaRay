<template>
  <div>
    <h1 class="text-h4 mb-6">个人中心</h1>

    <v-row>
      <v-col cols="12" md="4">
        <v-card>
          <v-card-text class="text-center py-8">
            <v-avatar size="100" color="primary" class="mb-4">
              <span class="text-h4">{{ user?.username?.substring(0, 2).toUpperCase() }}</span>
            </v-avatar>
            <div class="text-h5">{{ user?.username }}</div>
            <div class="text-grey">{{ user?.email || '未设置邮箱' }}</div>

            <v-divider class="my-4" />

            <div class="text-h4 font-weight-bold text-primary">
              {{ formatBytes(user?.traffic_balance || 0) }}
            </div>
            <div class="text-grey">剩余流量</div>

            <div class="text-body-2 text-grey mt-2">
              余额：¥{{ (user?.balance || 0) / 100 }}
            </div>

            <v-btn color="primary" variant="tonal" class="mt-4" to="/packages">
              <v-icon start>mdi-cart-plus</v-icon>
              购买套餐
            </v-btn>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="8">
        <v-card>
          <v-card-title>基本信息</v-card-title>
          <v-card-text>
            <v-form ref="formRef" @submit.prevent="updateProfile">
              <v-text-field
                v-model="form.username"
                label="用户名"
                disabled
                hint="用户名不可修改"
                persistent-hint
              />

              <v-text-field
                v-model="form.email"
                label="邮箱"
                type="email"
                class="mt-4"
              />

              <v-text-field
                v-model="form.qq"
                label="QQ号码"
              />

              <v-btn
                color="primary"
                type="submit"
                :loading="saving"
                class="mt-4"
              >
                保存修改
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
                :rules="[v => v.length >= 6 || '密码至少6个字符']"
              />

              <v-text-field
                v-model="passwordForm.confirm_password"
                label="确认新密码"
                type="password"
                :rules="[v => v === passwordForm.new_password || '两次输入的密码不一致']"
              />

              <v-btn
                color="warning"
                type="submit"
                :loading="changingPassword"
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
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { userAPI } from '@/api'
import { useSnackbar } from '@/composables/useSnackbar'

const authStore = useAuthStore()
const { showSnackbar } = useSnackbar()

const user = computed(() => authStore.user)
const formRef = ref(null)
const passwordFormRef = ref(null)
const saving = ref(false)
const changingPassword = ref(false)

const form = ref({
  username: '',
  email: '',
  qq: ''
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

async function updateProfile() {
  saving.value = true
  try {
    await userAPI.updateProfile(form.value)
    await authStore.fetchProfile()
    showSnackbar('个人信息已更新', 'success')
  } catch (error) {
    showSnackbar(error.message || '更新失败', 'error')
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  const { valid } = await passwordFormRef.value.validate()
  if (!valid) return

  changingPassword.value = true
  try {
    await userAPI.changePassword({
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    showSnackbar('密码修改成功', 'success')
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (error) {
    showSnackbar(error.response?.data?.message || error.message || '修改失败', 'error')
  } finally {
    changingPassword.value = false
  }
}

onMounted(() => {
  if (user.value) {
    form.value = {
      username: user.value.username,
      email: user.value.email || '',
      qq: user.value.qq || ''
    }
  }
})
</script>
