<template>
  <div>
    <div class="d-flex align-center mb-6">
      <h1 class="text-h4">支付配置</h1>
      <v-spacer />
      <v-btn color="primary" @click="showCreateDialog = true">
        <v-icon start>mdi-plus</v-icon>
        添加支付渠道
      </v-btn>
    </div>

    <v-row>
      <v-col v-for="payment in payments" :key="payment.id" cols="12" md="6" lg="4">
        <v-card :class="{ 'opacity-50': !payment.enabled }">
          <v-card-title class="d-flex align-center">
            <v-icon :color="payment.enabled ? 'success' : 'grey'" class="mr-2">
              {{ payment.enabled ? 'mdi-check-circle' : 'mdi-close-circle' }}
            </v-icon>
            {{ payment.name }}
          </v-card-title>
          <v-card-subtitle>{{ payment.provider }}</v-card-subtitle>
          <v-card-text>
            <v-list density="compact" class="bg-transparent">
              <v-list-item>
                <v-list-item-title>商户ID</v-list-item-title>
                <v-list-item-subtitle>{{ payment.merchant_id }}</v-list-item-subtitle>
              </v-list-item>
              <v-list-item>
                <v-list-item-title>API地址</v-list-item-title>
                <v-list-item-subtitle>{{ payment.api_url }}</v-list-item-subtitle>
              </v-list-item>
            </v-list>
          </v-card-text>
          <v-card-actions>
            <v-switch
              :model-value="payment.enabled"
              color="success"
              hide-details
              density="compact"
              @update:model-value="togglePayment(payment)"
            />
            <v-spacer />
            <v-btn icon size="small" variant="text" @click="editPayment(payment)">
              <v-icon>mdi-pencil</v-icon>
            </v-btn>
            <v-btn icon size="small" variant="text" color="error" @click="deletePayment(payment)">
              <v-icon>mdi-delete</v-icon>
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="payments.length === 0" class="text-center py-12">
      <v-icon size="64" color="grey">mdi-credit-card-outline</v-icon>
      <div class="text-h6 mt-4 text-grey">暂无支付配置</div>
    </div>

    <!-- 创建/编辑对话框 -->
    <v-dialog v-model="showCreateDialog" max-width="600" persistent>
      <v-card>
        <v-card-title>{{ editingPayment ? '编辑支付配置' : '添加支付配置' }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="渠道名称"
              :rules="[v => !!v || '请输入渠道名称']"
            />

            <v-select
              v-model="form.provider"
              :items="providers"
              label="支付提供商"
              :rules="[v => !!v || '请选择提供商']"
            />

            <v-text-field
              v-model="form.merchant_id"
              label="商户ID"
              :rules="[v => !!v || '请输入商户ID']"
            />

            <v-text-field
              v-model="form.merchant_key"
              label="商户密钥"
              :type="showKey ? 'text' : 'password'"
              :append-inner-icon="showKey ? 'mdi-eye-off' : 'mdi-eye'"
              @click:append-inner="showKey = !showKey"
              :rules="[v => !!v || '请输入商户密钥']"
            />

            <v-text-field
              v-model="form.api_url"
              label="API地址"
              :rules="[v => !!v || '请输入API地址']"
            />

            <v-text-field
              v-model="form.notify_url"
              label="回调地址"
            />

            <v-switch
              v-model="form.enabled"
              label="启用"
              color="success"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" @click="savePayment" :loading="saving">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 删除确认 -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除支付配置 "{{ deletingPayment?.name }}" 吗？
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

const payments = ref([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const showKey = ref(false)

const showCreateDialog = ref(false)
const showDeleteDialog = ref(false)
const editingPayment = ref(null)
const deletingPayment = ref(null)
const formRef = ref(null)

const form = ref({
  name: '',
  provider: 'epay',
  merchant_id: '',
  merchant_key: '',
  api_url: '',
  notify_url: '',
  enabled: true
})

const providers = [
  { title: '彩虹易支付 (Epay)', value: 'epay' },
  { title: '自定义支付', value: 'custom' }
]

function editPayment(payment) {
  editingPayment.value = payment
  form.value = { ...payment }
  showCreateDialog.value = true
}

function deletePayment(payment) {
  deletingPayment.value = payment
  showDeleteDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingPayment.value = null
  form.value = {
    name: '',
    provider: 'epay',
    merchant_id: '',
    merchant_key: '',
    api_url: '',
    notify_url: '',
    enabled: true
  }
}

async function savePayment() {
  const { valid } = await formRef.value.validate()
  if (!valid) return

  saving.value = true
  try {
    if (editingPayment.value) {
      await adminAPI.payments.update(editingPayment.value.id, form.value)
    } else {
      await adminAPI.payments.create(form.value)
    }
    closeDialog()
    loadPayments()
  } catch (error) {
    console.error('Failed to save payment:', error)
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!deletingPayment.value) return

  deleting.value = true
  try {
    await adminAPI.payments.delete(deletingPayment.value.id)
    showDeleteDialog.value = false
    deletingPayment.value = null
    loadPayments()
  } catch (error) {
    console.error('Failed to delete payment:', error)
  } finally {
    deleting.value = false
  }
}

async function togglePayment(payment) {
  try {
    await adminAPI.payments.update(payment.id, { enabled: !payment.enabled })
    payment.enabled = !payment.enabled
  } catch (error) {
    console.error('Failed to toggle payment:', error)
  }
}

async function loadPayments() {
  loading.value = true
  try {
    const response = await adminAPI.payments.list()
    payments.value = response.data || []
  } catch (error) {
    console.error('Failed to load payments:', error)
  } finally {
    loading.value = false
  }
}

onMounted(loadPayments)
</script>
