<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">节点管理</h1>
      <v-spacer />
      <v-btn variant="tonal" prepend-icon="mdi-refresh" :loading="loading" @click="loadNodes">
        刷新列表
      </v-btn>
    </div>

    <v-row class="mb-4">
      <v-col cols="12" md="4">
        <v-card variant="tonal" color="primary">
          <v-card-text class="d-flex align-center justify-space-between">
            <div>
              <div class="text-caption text-medium-emphasis">节点总数</div>
              <div class="text-h4">{{ totalNodes }}</div>
            </div>
            <v-icon size="36">mdi-server-network</v-icon>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card variant="tonal" color="success">
          <v-card-text class="d-flex align-center justify-space-between">
            <div>
              <div class="text-caption text-medium-emphasis">在线节点</div>
              <div class="text-h4">{{ onlineNodes }}</div>
            </div>
            <v-icon size="36">mdi-lan-connect</v-icon>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card variant="tonal" color="error">
          <v-card-text class="d-flex align-center justify-space-between">
            <div>
              <div class="text-caption text-medium-emphasis">离线节点</div>
              <div class="text-h4">{{ offlineNodes }}</div>
            </div>
            <v-icon size="36">mdi-lan-disconnect</v-icon>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-alert
      type="info"
      variant="tonal"
      class="mb-4"
      icon="mdi-information-outline"
      text="节点会在安装脚本首次执行后自动注册，并按上报周期主动向面板拉取最新配置。"
    />

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

        <template v-slot:item.name="{ item }">
          <div class="d-flex flex-column">
            <span class="font-weight-medium">{{ item.name }}</span>
            <span class="text-caption text-medium-emphasis">倍率 {{ item.multiplier || 1 }}</span>
          </div>
        </template>

        <template v-slot:item.host="{ item }">
          <code>{{ item.host }}</code>
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
          <v-btn icon size="small" variant="text" color="error" @click="deleteNode(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>

        <template v-slot:no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-server-network-off</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无节点</div>
            <div class="text-body-2 mt-2 text-grey">节点会在安装脚本首次执行后自动注册</div>
          </div>
        </template>
      </v-data-table>
    </v-card>

    <!-- 编辑节点对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="600" persistent>
      <v-card>
        <v-card-title>编辑节点</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="节点名称"
              :rules="[v => !!v || '请输入节点名称']"
            />

            <v-row>
              <v-col cols="12">
                <v-text-field
                  v-model="form.host"
                  label="节点地址"
                  :rules="[v => !!v || '请输入节点地址']"
                />
              </v-col>
            </v-row>

            <v-combobox
              v-model="form.protocols"
              :items="['gost']"
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
import { computed, ref, onMounted } from 'vue'
import { adminAPI } from '@/api'
import { useSnackbar } from '@/composables/useSnackbar'
import dayjs from 'dayjs'

const { showSnackbar } = useSnackbar()

const nodes = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const editingNode = ref(null)
const deletingNode = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  host: '',
  protocols: ['gost'],
  region: '',
  multiplier: 1.0
})

const totalNodes = computed(() => nodes.value.length)
const onlineNodes = computed(() => nodes.value.filter((item) => item.status === 'online').length)
const offlineNodes = computed(() => totalNodes.value - onlineNodes.value)

const headers = [
  { title: '状态', key: 'status', width: 100 },
  { title: '名称', key: 'name' },
  { title: '接入地址', key: 'host' },
  { title: '协议', key: 'protocols' },
  { title: '地区', key: 'region' },
  { title: '最后活跃', key: 'last_seen' },
  { title: '操作', key: 'actions', width: 120 }
]

function formatDate(date) {
  if (!date) return '从未'
  return dayjs(date).fromNow()
}

function editNode(node) {
  editingNode.value = node
  form.value = {
    name: node.name,
    host: node.host,
    protocols: Array.isArray(node.protocols) ? node.protocols.filter((item) => item === 'gost') : ['gost'],
    region: node.region || '',
    multiplier: node.multiplier || 1
  }
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
    protocols: ['gost'],
    region: '',
    multiplier: 1.0
  }
}

async function saveNode() {
  if (!editingNode.value) return

  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    await adminAPI.nodes.update(editingNode.value.id, form.value)
    showSnackbar('节点更新成功', 'success')
    closeDialog()
    loadNodes()
  } catch (error) {
    console.error('Failed to save node:', error)
    showSnackbar(error.response?.data?.message || error.message || '保存失败', 'error')
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingNode.value) return

  deleting.value = true
  try {
    await adminAPI.nodes.delete(deletingNode.value.id)
    showSnackbar('节点删除成功', 'success')
    showDeleteDialog.value = false
    deletingNode.value = null
    loadNodes()
  } catch (error) {
    console.error('Failed to delete node:', error)
    showSnackbar(error.response?.data?.message || error.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}

async function loadNodes() {
  loading.value = true
  try {
    const response = await adminAPI.nodes.list()
    nodes.value = response.data || []
  } catch (error) {
    console.error('Failed to load nodes:', error)
    showSnackbar(error.response?.data?.message || error.message || '加载节点失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadNodes()
})
</script>
