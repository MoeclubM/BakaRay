<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">节点管理</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        添加节点
      </v-btn>
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="nodes"
        :loading="loading"
      >
        <template v-slot:item.status="{ item }">
          <v-chip :color="item.status === 'online' ? 'success' : 'error'" size="small">
            {{ item.status === 'online' ? '在线' : '离线' }}
          </v-chip>
        </template>

        <template v-slot:item.region="{ item }">
          {{ item.region || '未知' }}
        </template>

        <template v-slot:item.protocols="{ item }">
          <v-chip v-for="proto in (item.protocols || [])" :key="proto" size="x-small" class="mr-1">
            {{ proto }}
          </v-chip>
        </template>

        <template v-slot:item.last_seen="{ item }">
          {{ formatDate(item.last_seen) }}
        </template>

        <template v-slot:item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editNode(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="info" @click="reloadNode(item)">
            <v-icon>mdi-refresh</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="error" @click="deleteNode(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>
      </v-data-table>
    </v-card>

    <!-- 创建/编辑节点对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="600" persistent>
      <v-card>
        <v-card-title>{{ editingNode ? '编辑节点' : '添加节点' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="节点名称"
              :rules="[v => !!v || '请输入节点名称']"
            />

            <v-row>
              <v-col cols="8">
                <v-text-field
                  v-model="form.host"
                  label="节点地址"
                  :rules="[v => !!v || '请输入节点地址']"
                />
              </v-col>
              <v-col cols="4">
                <v-text-field
                  v-model.number="form.port"
                  label="API端口"
                  type="number"
                  :rules="[v => v > 0 || '请输入有效端口']"
                />
              </v-col>
            </v-row>

            <v-text-field
              v-model="form.secret"
              label="认证密钥"
              :type="showSecret ? 'text' : 'password'"
              :append-inner-icon="showSecret ? 'mdi-eye-off' : 'mdi-eye'"
              @click:append-inner="showSecret = !showSecret"
              :rules="[v => !!v || '请输入认证密钥']"
              hint="用于节点与面板通信的密钥"
            />

            <v-select
              v-model="form.node_group_id"
              :items="nodeGroups"
              item-title="name"
              item-value="id"
              label="节点组"
            />

            <v-combobox
              v-model="form.protocols"
              :items="['gost', 'iptables', 'echo']"
              label="支持的协议"
              multiple
              chips
            />

            <v-text-field
              v-model="form.region"
              label="节点地区"
              hint="如：香港、日本、美国"
            />

            <v-row>
              <v-col cols="6">
                <v-text-field
                  v-model.number="form.multiplier"
                  label="倍率"
                  type="number"
                  step="0.1"
                  min="0.1"
                />
              </v-col>
            </v-row>
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="saveNode" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除节点 "{{ deletingNode?.name }}" 吗？
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

const nodes = ref([])
const nodeGroups = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const showSecret = ref(false)
const editingNode = ref(null)
const deletingNode = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  host: '',
  port: 8080,
  secret: '',
  node_group_id: null,
  protocols: ['gost'],
  region: '',
  multiplier: 1.0
})

const headers = [
  { title: '状态', key: 'status', width: 100 },
  { title: '名称', key: 'name' },
  { title: '地址', key: 'host' },
  { title: '协议', key: 'protocols' },
  { title: '地区', key: 'region' },
  { title: '最后活跃', key: 'last_seen' },
  { title: '操作', key: 'actions', width: 150 }
]

function formatDate(date) {
  if (!date) return '从未'
  return dayjs(date).fromNow()
}

function editNode(node) {
  editingNode.value = node
  form.value = { ...node }
  showCreateDialog.value = true
}

function deleteNode(node) {
  deletingNode.value = node
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingNode.value = null
  form.value = {
    name: '',
    host: '',
    port: 8080,
    secret: '',
    node_group_id: null,
    protocols: ['gost'],
    region: '',
    multiplier: 1.0
  }
}

async function saveNode() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    if (editingNode.value) {
      await adminAPI.nodes.update(editingNode.value.id, form.value)
    } else {
      await adminAPI.nodes.create(form.value)
    }
    closeDialog()
    loadNodes()
  } catch (error) {
    console.error('Failed to save node:', error)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingNode.value) return

  deleting.value = true
  try {
    await adminAPI.nodes.delete(deletingNode.value.id)
    showDeleteDialog.value = false
    deletingNode.value = null
    loadNodes()
  } catch (error) {
    console.error('Failed to delete node:', error)
  } finally {
    deleting.value = false
  }
}

async function reloadNode(node) {
  try {
    await adminAPI.nodes.reload(node.id)
  } catch (error) {
    console.error('Failed to reload node:', error)
  }
}

async function loadNodes() {
  loading.value = true
  try {
    const response = await adminAPI.nodes.list()
    nodes.value = response.data || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
  } finally {
    loading.value = false
  }
}

async function loadNodeGroups() {
  try {
    const response = await adminAPI.nodeGroups.list()
    nodeGroups.value = response.data || []
  } catch {
    nodeGroups.value = []
  }
}

onMounted(() => {
  loadNodes()
  loadNodeGroups()
})
</script>
