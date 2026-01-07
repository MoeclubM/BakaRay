<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">套餐管理</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        添加套餐
      </v-btn>
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="packages"
        :loading="loading"
      >
        <template v-slot:item.traffic="{ item }">
          {{ formatBytes(item.traffic) }}
        </template>

        <template v-slot:item.price="{ item }">
          ¥{{ item.price / 100 }}
        </template>

        <template v-slot:item.user_group_id="{ item }">
          {{ getGroupName(item.user_group_id) }}
        </template>

        <template v-slot:item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editPackage(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="error" @click="deletePackage(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>

        <template v-slot:no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-package-variant-closed</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无套餐</div>
          </div>
        </template>
      </v-data-table>
    </v-card>

    <!-- 创建/编辑套餐对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="500" persistent>
      <v-card>
        <v-card-title>{{ editingPackage ? '编辑套餐' : '添加套餐' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="套餐名称"
              :rules="[v => !!v || '请输入套餐名称']"
            />

            <v-text-field
              v-model.number="form.traffic"
              label="流量（GB）"
              type="number"
              :rules="[v => v > 0 || '请输入有效流量']"
            />

            <v-text-field
              v-model.number="form.price"
              label="价格（分）"
              type="number"
              :rules="[v => v >= 0 || '请输入有效价格']"
            />

            <v-select
              v-model="form.user_group_id"
              :items="userGroups"
              item-title="name"
              item-value="id"
              label="适用用户组"
              hint="留空则所有用户可用"
              clearable
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="savePackage" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除套餐 "{{ deletingPackage?.name }}" 吗？
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
import { useSnackbar } from '@/composables/useSnackbar'

const { showSnackbar } = useSnackbar()

const packages = ref([])
const userGroups = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const editingPackage = ref(null)
const deletingPackage = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  traffic: 0,
  price: 0,
  user_group_id: null
})

const headers = [
  { title: '名称', key: 'name' },
  { title: '流量', key: 'traffic' },
  { title: '价格', key: 'price' },
  { title: '适用用户组', key: 'user_group_id' },
  { title: '操作', key: 'actions', width: 120 }
]

function formatBytes(bytes) {
  if (!bytes) return '0 GB'
  const gb = bytes / (1024 * 1024 * 1024)
  return gb + ' GB'
}

function getGroupName(id) {
  if (!id) return '所有用户'
  const group = userGroups.value.find(g => g.id === id)
  return group?.name || '未知'
}

function editPackage(pkg) {
  editingPackage.value = pkg
  form.value = { ...pkg, traffic: pkg.traffic / (1024 * 1024 * 1024) }
  showCreateDialog.value = true
}

function deletePackage(pkg) {
  deletingPackage.value = pkg
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingPackage.value = null
  form.value = { name: '', traffic: 0, price: 0, user_group_id: null }
}

async function savePackage() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    const data = { ...form.value }
    data.traffic = data.traffic * 1024 * 1024 * 1024

    if (editingPackage.value) {
      await adminAPI.packages.update(editingPackage.value.id, data)
      showSnackbar('套餐更新成功', 'success')
    } else {
      await adminAPI.packages.create(data)
      showSnackbar('套餐创建成功', 'success')
    }
    closeDialog()
    loadPackages()
  } catch (error) {
    console.error('Failed to save package:', error)
    showSnackbar(error.response?.data?.message || error.message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingPackage.value) return

  deleting.value = true
  try {
    await adminAPI.packages.delete(deletingPackage.value.id)
    showSnackbar('套餐删除成功', 'success')
    showDeleteDialog.value = false
    deletingPackage.value = null
    loadPackages()
  } catch (error) {
    console.error('Failed to delete package:', error)
    showSnackbar(error.response?.data?.message || error.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}

async function loadPackages() {
  loading.value = true
  try {
    const response = await adminAPI.packages.list()
    packages.value = response.data || []
  } catch (error) {
    console.error('Failed to load packages:', error)
    showSnackbar(error.response?.data?.message || error.message || '加载套餐失败', 'error')
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
  loadPackages()
  loadUserGroups()
})
</script>
