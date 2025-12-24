<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">转发规则</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        创建规则
      </v-btn>
    </div>

    <v-card>
      <v-data-table
        :headers="headers"
        :items="rules"
        :loading="loading"
      >
        <template v-slot:item.enabled="{ item }">
          <v-switch
            :model-value="item.enabled"
            color="success"
            hide-details
            density="compact"
            @update:model-value="toggleRule(item)"
          />
        </template>

        <template v-slot:item.protocol="{ item }">
          <v-chip size="small" :color="getProtocolColor(item.protocol)">
            {{ item.protocol }}
          </v-chip>
        </template>

        <template v-slot:item.traffic="{ item }">
          <div class="text-caption">
            <span class="text-success">↓ {{ formatBytes(item.traffic_used) }}</span>
            <span class="ml-2">/ {{ formatBytes(item.traffic_limit) }}</span>
          </div>
          <v-progress-linear
            :model-value="item.traffic_limit > 0 ? (item.traffic_used / item.traffic_limit) * 100 : 0"
            :color="getTrafficColor(item.traffic_limit > 0 ? item.traffic_used / item.traffic_limit : 0)"
            height="4"
            rounded
          />
        </template>

        <template v-slot:item.speed_limit="{ item }">
          {{ item.speed_limit ? item.speed_limit + ' Kbps' : '不限速' }}
        </template>

        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>

        <template v-slot:item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editRule(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn icon size="small" variant="text" color="error" @click="deleteRule(item)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>
      </v-data-table>
    </v-card>

    <!-- 创建/编辑规则对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="600" persistent>
      <v-card>
        <v-card-title>{{ editingRule ? '编辑规则' : '创建规则' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="规则名称"
              :rules="[v => !!v || '请输入规则名称']"
              class="mb-2"
            />

            <v-select
              v-model="form.protocol"
              :items="protocols"
              label="协议类型"
              :rules="[v => !!v || '请选择协议']"
              class="mb-2"
            />

            <v-select
              v-model="form.node_id"
              :items="nodes"
              item-title="name"
              item-value="id"
              label="选择节点"
              :rules="[v => !!v || '请选择节点']"
              class="mb-2"
            />

            <v-text-field
              v-model.number="form.listen_port"
              label="监听端口"
              type="number"
              :rules="[v => v > 0 || '请输入有效端口']"
              class="mb-2"
            />

            <v-row v-if="!editingRule">
              <v-col cols="8">
                <v-text-field
                  v-model="form.target_host"
                  label="目标地址"
                  :rules="[v => !!v || '请输入目标地址']"
                />
              </v-col>
              <v-col cols="4">
                <v-text-field
                  v-model.number="form.target_port"
                  label="目标端口"
                  type="number"
                  :rules="[v => v > 0 || '请输入有效端口']"
                />
              </v-col>
            </v-row>

            <v-row>
              <v-col cols="6">
                <v-text-field
                  v-model.number="form.traffic_limit"
                  label="流量限制 (GB)"
                  type="number"
                />
              </v-col>
              <v-col cols="6">
                <v-text-field
                  v-model.number="form.speed_limit"
                  label="限速 (Kbps)"
                  type="number"
                  hint="0 或留空表示不限速"
                />
              </v-col>
            </v-row>

            <v-select
              v-model="form.mode"
              :items="modes"
              label="转发模式"
            />

            <v-switch
              v-model="form.enabled"
              label="启用规则"
              color="success"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="saveRule" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除规则 "{{ deletingRule?.name }}" 吗？此操作不可恢复。
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
import { ruleAPI, nodeAPI } from '@/api'
import dayjs from 'dayjs'

const rules = ref([])
const nodes = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const editingRule = ref(null)
const deletingRule = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  node_id: null,
  protocol: 'gost',
  listen_port: 0,
  target_host: '',
  target_port: 0,
  traffic_limit: 0,
  speed_limit: 0,
  mode: 'direct',
  enabled: true
})

const headers = [
  { title: '状态', key: 'enabled', width: 80 },
  { title: '名称', key: 'name' },
  { title: '协议', key: 'protocol' },
  { title: '端口', key: 'listen_port' },
  { title: '流量', key: 'traffic' },
  { title: '限速', key: 'speed_limit' },
  { title: '创建时间', key: 'created_at' },
  { title: '操作', key: 'actions', width: 100 }
]

const protocols = ['gost', 'iptables']
const modes = [
  { title: '直连', value: 'direct' },
  { title: '轮询', value: 'rr' },
  { title: '负载均衡', value: 'lb' }
]

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(date) {
  return dayjs(date).format('YYYY-MM-DD HH:mm')
}

function getProtocolColor(protocol) {
  const colors = { gost: 'blue', iptables: 'orange', echo: 'purple' }
  return colors[protocol] || 'grey'
}

function getTrafficColor(ratio) {
  if (ratio > 0.9) return 'error'
  if (ratio > 0.7) return 'warning'
  return 'success'
}

function editRule(rule) {
  editingRule.value = rule
  form.value = {
    ...form.value,
    ...rule,
    traffic_limit: rule.traffic_limit ? rule.traffic_limit / (1024 * 1024 * 1024) : 0,
    target_host: '',
    target_port: 0
  }
  showCreateDialog.value = true
}

function deleteRule(rule) {
  deletingRule.value = rule
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingRule.value = null
  form.value = {
    name: '',
    node_id: null,
    protocol: 'gost',
    listen_port: 0,
    target_host: '',
    target_port: 0,
    traffic_limit: 0,
    speed_limit: 0,
    mode: 'direct',
    enabled: true
  }
}

async function saveRule() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    const data = { ...form.value }
    if (!data.node_id) delete data.node_id
    data.traffic_limit = data.traffic_limit * 1024 * 1024 * 1024

    if (editingRule.value) {
      delete data.target_host
      delete data.target_port
      await ruleAPI.update(editingRule.value.id, data)
    } else {
      data.targets = [
        {
          host: form.value.target_host,
          port: form.value.target_port,
          weight: 1,
          enabled: true
        }
      ]
      delete data.target_host
      delete data.target_port
      await ruleAPI.create(data)
    }
    closeDialog()
    loadRules()
  } catch (error) {
    console.error('Failed to save rule:', error)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingRule.value) return

  deleting.value = true
  try {
    await ruleAPI.delete(deletingRule.value.id)
    showDeleteDialog.value = false
    deletingRule.value = null
    loadRules()
  } catch (error) {
    console.error('Failed to delete rule:', error)
  } finally {
    deleting.value = false
  }
}

async function toggleRule(rule) {
  try {
    await ruleAPI.update(rule.id, { enabled: !rule.enabled })
    rule.enabled = !rule.enabled
  } catch (error) {
    console.error('Failed to toggle rule:', error)
  }
}

async function loadRules() {
  loading.value = true
  try {
    const response = await ruleAPI.list()
    rules.value = response.data || []
  } catch (error) {
    console.error('Failed to load rules:', error)
  } finally {
    loading.value = false
  }
}

async function loadNodes() {
  try {
    const response = await nodeAPI.list()
    nodes.value = response.data || []
  } catch {
    nodes.value = []
  }
}

onMounted(() => {
  loadRules()
  loadNodes()
})
</script>
