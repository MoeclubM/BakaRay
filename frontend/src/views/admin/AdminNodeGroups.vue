<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">节点组管理</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        添加节点组
      </v-btn>
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="groups"
        :loading="loading"
      >
        <template v-slot:item.type="{ item }">
          <v-chip :color="item.type === 'entry' ? 'primary' : 'info'" size="small">
            {{ item.type === 'entry' ? '转发入口' : '转发目标' }}
          </v-chip>
        </template>

        <template v-slot:item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editGroup(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="error" @click="deleteGroup(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>

        <template v-slot:no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-lan-disconnect</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无节点组</div>
          </div>
        </template>
      </v-data-table>
    </v-card>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="500" persistent>
      <v-card>
        <v-card-title>{{ editingGroup ? '编辑节点组' : '添加节点组' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="组名"
              :rules="[v => !!v || '请输入组名']"
            />

            <v-select
              v-model="form.type"
              :items="types"
              label="类型"
              :rules="[v => !!v || '请选择类型']"
            />

            <v-textarea
              v-model="form.description"
              label="描述"
              rows="2"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="saveGroup" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除节点组 "{{ deletingGroup?.name }}" 吗？
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

const groups = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const editingGroup = ref(null)
const deletingGroup = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  type: 'entry',
  description: ''
})

const types = [
  { title: '转发入口', value: 'entry' },
  { title: '转发目标', value: 'target' }
]

const headers = [
  { title: '名称', key: 'name' },
  { title: '类型', key: 'type' },
  { title: '描述', key: 'description' },
  { title: '操作', key: 'actions', width: 120 }
]

function editGroup(group) {
  editingGroup.value = group
  form.value = { ...group }
  showCreateDialog.value = true
}

function deleteGroup(group) {
  deletingGroup.value = group
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingGroup.value = null
  form.value = { name: '', type: 'entry', description: '' }
}

async function saveGroup() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    if (editingGroup.value) {
      await adminAPI.nodeGroups.update(editingGroup.value.id, form.value)
      showSnackbar('节点组更新成功', 'success')
    } else {
      await adminAPI.nodeGroups.create(form.value)
      showSnackbar('节点组创建成功', 'success')
    }
    closeDialog()
    loadGroups()
  } catch (error) {
    console.error('Failed to save group:', error)
    showSnackbar(error.response?.data?.message || error.message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingGroup.value) return

  deleting.value = true
  try {
    await adminAPI.nodeGroups.delete(deletingGroup.value.id)
    showSnackbar('节点组删除成功', 'success')
    showDeleteDialog.value = false
    deletingGroup.value = null
    loadGroups()
  } catch (error) {
    console.error('Failed to delete group:', error)
    showSnackbar(error.response?.data?.message || error.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}

async function loadGroups() {
  loading.value = true
  try {
    const response = await adminAPI.nodeGroups.list()
    groups.value = response.data || []
  } catch (error) {
    console.error('Failed to load groups:', error)
    showSnackbar(error.response?.data?.message || error.message || '加载节点组失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(loadGroups)
</script>
