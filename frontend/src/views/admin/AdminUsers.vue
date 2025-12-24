<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">用户管理</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        添加用户
      </v-btn>
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="users"
        :loading="loading"
        :items-per-page="20"
      >
        <template v-slot:item.is_admin="{ item }">
          <v-chip v-if="item.is_admin" color="warning" size="small">管理员</v-chip>
        </template>

        <template v-slot:item.balance="{ item }">
          ¥{{ (item.balance || 0) / 100 }}
        </template>

        <template v-slot:item.user_group_id="{ item }">
          {{ getGroupName(item.user_group_id) }}
        </template>

        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>

        <template v-slot:item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editUser(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" @click="adjustBalance(item)">
            <v-icon>mdi-cash-plus</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="error" @click="deleteUser(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>
      </v-data-table>
    </v-card>

    <!-- 创建/编辑用户对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="500" persistent>
      <v-card>
        <v-card-title>{{ editingUser ? '编辑用户' : '添加用户' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.username"
              label="用户名"
              :disabled="editingUser"
              :rules="[v => !!v || '请输入用户名']"
            />

            <v-text-field
              v-model="form.password"
              :label="editingUser ? '新密码（留空不修改）' : '密码'"
              type="password"
              :rules="editingUser ? [] : [v => v.length >= 6 || '密码至少6个字符']"
            />

            <v-text-field
              v-model.number="form.balance"
              label="余额（分）"
              type="number"
            />

            <v-select
              v-model="form.user_group_id"
              :items="userGroups"
              item-title="name"
              item-value="id"
              label="用户组"
            />

            <v-switch
              v-model="form.is_admin"
              label="管理员权限"
              color="warning"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="saveUser" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 调整余额对话框 -->
    <v-dialog v-model="showBalanceDialog" max-width="400">
      <v-card>
        <v-card-title>调整余额</v-card-title>
        <v-card-text>
          <v-form ref="balanceFormRef">
            <v-text-field
              v-model.number="balanceForm.amount"
              label="金额（分，正数增加，负数减少）"
              type="number"
              :rules="[v => v !== 0 || '请输入非零金额']"
            />
            <v-text-field
              v-model="balanceForm.remark"
              label="备注"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showBalanceDialog = false">取消</v-btn>
          <v-btn color="primary" @click="confirmAdjustBalance" :loading="adjusting">确认</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除用户 "{{ deletingUser?.username }}" 吗？
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showDeleteDialog = false">取消</v-btn>
          <v-btn color="error" @click="confirmDelete" :loading="deleting">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { adminAPI } from '@/api'
import dayjs from 'dayjs'

const users = ref([])
const userGroups = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const adjusting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const showBalanceDialog = ref(false)
const editingUser = ref(null)
const deletingUser = ref(null)
const adjustingUser = ref(null)
const formRef = ref(null)
const balanceFormRef = ref(null)

const form = ref({
  username: '',
  password: '',
  balance: 0,
  user_group_id: null,
  is_admin: false
})

const balanceForm = ref({
  amount: 0,
  remark: ''
})

const headers = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '用户名', key: 'username' },
  { title: '余额', key: 'balance' },
  { title: '用户组', key: 'user_group_id' },
  { title: '管理员', key: 'is_admin', width: 100 },
  { title: '创建时间', key: 'created_at' },
  { title: '操作', key: 'actions', width: 180 }
]

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

function getGroupName(id) {
  const group = userGroups.value.find(g => g.id === id)
  return group?.name || '默认组'
}

function editUser(user) {
  editingUser.value = user
  form.value = { ...user, password: '' }
  showCreateDialog.value = true
}

function adjustBalance(user) {
  adjustingUser.value = user
  balanceForm.value = { amount: 0, remark: '' }
  showBalanceDialog.value = true
}

function deleteUser(user) {
  deletingUser.value = user
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingUser.value = null
  form.value = {
    username: '',
    password: '',
    balance: 0,
    user_group_id: null,
    is_admin: false
  }
}

async function saveUser() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    if (editingUser.value) {
      await adminAPI.users.update(editingUser.value.id, form.value)
    } else {
      await adminAPI.users.create(form.value)
    }
    closeDialog()
    loadUsers()
  } catch (error) {
    console.error('Failed to save user:', error)
  } finally {
    saving.value = false
  }
}

async function confirmAdjustBalance() {
  const { valid } = await balanceFormRef.value.validate()
  if (!valid) return

  adjusting.value = true
  try {
    await adminAPI.users.adjustBalance(adjustingUser.value.id, balanceForm.value)
    showBalanceDialog.value = false
    loadUsers()
  } catch (error) {
    console.error('Failed to adjust balance:', error)
  } finally {
    adjusting.value = false
  }
}

async function confirmDelete() {
  if (!deletingUser.value) return

  deleting.value = true
  try {
    await adminAPI.users.delete(deletingUser.value.id)
    showDeleteDialog.value = false
    deletingUser.value = null
    loadUsers()
  } catch (error) {
    console.error('Failed to delete user:', error)
  } finally {
    deleting.value = false
  }
}

async function loadUsers() {
  loading.value = true
  try {
    const response = await adminAPI.users.list()
    users.value = response.data?.list || response.data || []
  } catch (error) {
    console.error('Failed to load users:', error)
  } finally {
    loading.value = false
  }
}

async function loadUserGroups() {
  try {
    const response = await adminAPI.userGroups.list()
    userGroups.value = response.data || []
  } catch {
    userGroups.value = []
  }
}

onMounted(() => {
  loadUsers()
  loadUserGroups()
})
</script>
