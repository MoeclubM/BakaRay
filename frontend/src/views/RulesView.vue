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

        <template v-slot:no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-traffic-light</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无规则</div>
          </div>
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

            <v-divider class="my-4" />

            <div class="d-flex align-center mb-2">
              <div class="text-subtitle-1 font-weight-bold">转发目标</div>
              <v-spacer />
              <v-btn size="small" variant="tonal" @click="addTarget">
                <v-icon start>mdi-plus</v-icon>
                添加目标
              </v-btn>
            </div>

            <v-alert type="info" variant="tonal" density="compact" class="mb-3">
              轮询（rr）与负载均衡（lb）支持多个目标；lb 会使用权重进行分流。
            </v-alert>

            <v-row v-for="(t, index) in form.targets" :key="index" class="mb-2">
              <v-col cols="12" md="5">
                <v-text-field
                  v-model="t.host"
                  label="目标地址"
                  :rules="[v => !!v || '请输入目标地址']"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="3">
                <v-text-field
                  v-model.number="t.port"
                  label="目标端口"
                  type="number"
                  :rules="[v => v > 0 || '请输入有效端口']"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="2">
                <v-text-field
                  v-model.number="t.weight"
                  label="权重"
                  type="number"
                  min="1"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="2" class="d-flex align-center">
                <v-switch v-model="t.enabled" label="启用" density="compact" hide-details class="mr-2" />
                <v-btn
                  icon
                  size="small"
                  variant="text"
                  color="error"
                  :disabled="form.targets.length <= 1"
                  @click="removeTarget(index)"
                >
                  <v-icon>mdi-delete</v-icon>
                </v-btn>
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

            <v-expand-transition>
              <div v-if="form.protocol === 'gost'">
                <v-divider class="my-4" />
                <div class="text-subtitle-1 font-weight-bold mb-2">Gost 配置</div>
                <v-row>
                  <v-col cols="12" md="4">
                    <v-select
                      v-model="form.gost_config.transport"
                      :items="['tcp', 'udp', 'quic']"
                      label="传输类型"
                    />
                  </v-col>
                  <v-col cols="12" md="4" class="d-flex align-center">
                    <v-switch v-model="form.gost_config.tls" label="TLS" color="primary" hide-details />
                  </v-col>
                  <v-col cols="12" md="4">
                    <v-text-field v-model.number="form.gost_config.timeout" label="超时(秒)" type="number" min="0" />
                  </v-col>
                </v-row>
                <v-text-field v-model="form.gost_config.chain" label="代理链(可选)" />
              </div>
            </v-expand-transition>

            <v-expand-transition>
              <div v-if="form.protocol === 'iptables'">
                <v-divider class="my-4" />
                <div class="text-subtitle-1 font-weight-bold mb-2">IPTables 配置</div>
                <v-row>
                  <v-col cols="12" md="4">
                    <v-select v-model="form.iptables_config.proto" :items="['tcp', 'udp']" label="协议" />
                  </v-col>
                  <v-col cols="12" md="4" class="d-flex align-center">
                    <v-switch v-model="form.iptables_config.snat" label="SNAT (MASQUERADE)" color="primary" hide-details />
                  </v-col>
                  <v-col cols="12" md="4">
                    <v-text-field v-model="form.iptables_config.iface" label="入站网卡(可选)" hint="例如 eth0" persistent-hint />
                  </v-col>
                </v-row>
              </div>
            </v-expand-transition>

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
import { useSnackbar } from '@/composables/useSnackbar'
import dayjs from 'dayjs'

const { showSnackbar } = useSnackbar()

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

function defaultForm() {
  return {
    name: '',
    node_id: null,
    protocol: 'gost',
    listen_port: 0,
    targets: [{ host: '', port: 0, weight: 1, enabled: true }],
    traffic_limit: 0,
    speed_limit: 0,
    mode: 'direct',
    enabled: true,
    gost_config: {
      transport: 'tcp',
      tls: false,
      chain: '',
      timeout: 0
    },
    iptables_config: {
      proto: 'tcp',
      snat: false,
      iface: ''
    }
  }
}

const form = ref(defaultForm())

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

const protocols = [
  { title: 'TCP 转发 (forward)', value: 'forward' },
  { title: 'SOCKS5 代理 (socks5)', value: 'socks5' },
  { title: 'HTTP 代理 (http)', value: 'http' },
  { title: 'Shadowsocks (ss)', value: 'ss' },
  { title: 'QUIC', value: 'quic' },
  { title: 'WebSocket (ws)', value: 'ws' },
  { title: 'WebSocket Secure (wss)', value: 'wss' },
  { title: 'HTTP/2 (http2)', value: 'http2' }
]
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
  loadRuleDetail(rule.id)
}

function deleteRule(rule) {
  deletingRule.value = rule
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingRule.value = null
  form.value = defaultForm()
}

function addTarget() {
  form.value.targets.push({ host: '', port: 0, weight: 1, enabled: true })
}

function removeTarget(index) {
  if (form.value.targets.length <= 1) return
  form.value.targets.splice(index, 1)
}

async function loadRuleDetail(id) {
  loading.value = true
  try {
    const body = await ruleAPI.get(id)
    const detail = body?.code === 0 ? body.data : null
    if (!detail?.rule) throw new Error('Invalid rule detail')

    editingRule.value = detail.rule
    form.value = {
      ...defaultForm(),
      name: detail.rule.name,
      node_id: detail.rule.node_id,
      protocol: detail.rule.protocol,
      listen_port: detail.rule.listen_port,
      traffic_limit: detail.rule.traffic_limit ? detail.rule.traffic_limit / (1024 * 1024 * 1024) : 0,
      speed_limit: detail.rule.speed_limit || 0,
      mode: detail.rule.mode || 'direct',
      enabled: !!detail.rule.enabled,
      targets: (detail.targets && detail.targets.length > 0) ? detail.targets.map((t) => ({
        host: t.host,
        port: t.port,
        weight: t.weight ?? 1,
        enabled: t.enabled !== false
      })) : [{ host: '', port: 0, weight: 1, enabled: true }],
      gost_config: detail.gost_config ? {
        transport: detail.gost_config.transport || 'tcp',
        tls: !!detail.gost_config.tls,
        chain: detail.gost_config.chain || '',
        timeout: detail.gost_config.timeout || 0
      } : defaultForm().gost_config,
      iptables_config: detail.iptables_config ? {
        proto: detail.iptables_config.proto || 'tcp',
        snat: !!detail.iptables_config.snat,
        iface: detail.iptables_config.iface || ''
      } : defaultForm().iptables_config
    }

    showCreateDialog.value = true
  } catch (error) {
    console.error('Failed to load rule detail:', error)
    showSnackbar(error.message || '加载规则失败', 'error')
  } finally {
    loading.value = false
  }
}

async function saveRule() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    const data = {
      name: form.value.name,
      node_id: form.value.node_id,
      protocol: form.value.protocol,
      listen_port: form.value.listen_port,
      traffic_limit: Math.max(0, Math.round((Number(form.value.traffic_limit) || 0) * 1024 * 1024 * 1024)),
      speed_limit: Math.max(0, Number(form.value.speed_limit) || 0),
      mode: form.value.mode,
      enabled: form.value.enabled,
      targets: (form.value.targets || []).map((t) => ({
        host: (t.host || '').trim(),
        port: Number(t.port) || 0,
        weight: Math.max(1, Number(t.weight) || 1),
        enabled: t.enabled !== false
      })).filter((t) => t.host && t.port > 0),
      gost_config: form.value.protocol === 'gost' ? form.value.gost_config : null,
      iptables_config: form.value.protocol === 'iptables' ? form.value.iptables_config : null
    }

    if (!data.node_id) delete data.node_id
    if (!data.targets || data.targets.length === 0) {
      showSnackbar('请至少添加一个有效目标', 'error')
      return
    }

    if (editingRule.value) {
      await ruleAPI.update(editingRule.value.id, data)
      showSnackbar('规则已更新', 'success')
    } else {
      await ruleAPI.create(data)
      showSnackbar('规则已创建', 'success')
    }
    closeDialog()
    loadRules()
  } catch (error) {
    console.error('Failed to save rule:', error)
    showSnackbar(error.response?.data?.message || error.message || '保存失败', 'error')
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
