<template>
  <div>
    <h1 class="text-h4 mb-6">站点设置</h1>

    <v-overlay v-model="loading" contained class="align-center justify-center">
      <v-progress-circular indeterminate size="64" />
    </v-overlay>

    <v-card>
      <v-card-text>
        <v-form ref="formRef" @submit.prevent="saveSettings">
          <div class="text-subtitle-1 font-weight-bold mb-4">基本信息</div>

          <v-text-field
            v-model="form.site_name"
            label="站点名称"
            :rules="[v => !!v || '请输入站点名称']"
            class="mb-4"
          />

          <v-text-field
            v-model="form.site_domain"
            label="站点域名"
            hint="用于生成节点配置链接"
            persistent-hint
            class="mb-4"
          />

          <v-divider class="my-6" />

          <div class="text-subtitle-1 font-weight-bold mb-4">节点配置</div>

          <v-text-field
            v-model="form.node_secret"
            label="节点认证密钥"
            :type="showSecret ? 'text' : 'password'"
            :append-inner-icon="showSecret ? 'mdi-eye-off' : 'mdi-eye'"
            @click:append-inner="showSecret = !showSecret"
            hint="节点连接面板时使用的密钥"
            persistent-hint
            class="mb-4"
          />

          <v-text-field
            v-model.number="form.node_report_interval"
            label="节点上报频率（秒）"
            type="number"
            min="10"
            max="300"
            hint="建议值：30-60秒"
            persistent-hint
            class="mb-4"
          />

          <v-divider class="my-6" />

          <div class="d-flex justify-end">
            <v-btn
              color="primary"
              type="submit"
              :loading="saving"
              size="large"
            >
              保存设置
            </v-btn>
          </div>
        </v-form>
      </v-card-text>
    </v-card>

    <!-- 一键配置链接 -->
    <v-card class="mt-4">
      <v-card-title>节点配置模板</v-card-title>
      <v-card-text>
        <v-alert type="info" variant="tonal" class="mb-4">
          将以下配置发送给节点管理员（需要将 node_id 替换为对应节点 ID）。
        </v-alert>

        <v-textarea
          :model-value="nodeConfigYaml"
          label="config.yaml"
          readonly
          auto-grow
          rows="10"
          append-inner-icon="mdi-content-copy"
          @click:append-inner="copyNodeConfigYaml"
        />
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { adminAPI } from '@/api'
import { useSnackbar } from '@/composables/useSnackbar'

const formRef = ref(null)
const saving = ref(false)
const showSecret = ref(false)
const loading = ref(false)
const { showSnackbar } = useSnackbar()

const form = ref({
  site_name: 'BakaRay',
  site_domain: '',
  node_secret: '',
  node_report_interval: 30
})

const panelURL = computed(() => {
  const raw = (form.value.site_domain || '').trim()
  if (!raw) return window.location.origin
  if (raw.startsWith('http://') || raw.startsWith('https://')) return raw
  return `https://${raw}`
})

const nodeConfigYaml = computed(() => {
  const secret = form.value.node_secret || ''
  const reportInterval = Number(form.value.node_report_interval) || 30
  return `# BakaRay-Node 配置文件\npanel:\n  url: \"${panelURL.value}\"\n  node_id: 1\n  secret: \"${secret}\"\n\nnode:\n  report_interval: ${reportInterval}\n  probe_interval: 5\n  http_port: 8081\n  listen_ports:\n    - 8080\n    - 8090\n    - 8100\n\nlogger:\n  level: \"info\"\n  output: \"stdout\"\n`
})

async function copyNodeConfigYaml() {
  try {
    await navigator.clipboard.writeText(nodeConfigYaml.value)
    showSnackbar('已复制 config.yaml', 'success')
  } catch (error) {
    console.error('Failed to copy config yaml:', error)
    showSnackbar('复制失败', 'error')
  }
}

async function loadSettings() {
  loading.value = true
  try {
    const response = await adminAPI.site.get()
    if (response.data) {
      form.value = { ...form.value, ...response.data }
    }
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    await adminAPI.site.update(form.value)
  } catch (error) {
    console.error('Failed to save settings:', error)
  } finally {
    saving.value = false
  }
}

onMounted(loadSettings)
</script>
