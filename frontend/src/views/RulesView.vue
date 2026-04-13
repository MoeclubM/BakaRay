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
      <v-data-table :headers="headers" :items="rules" :loading="loading">
        <template #item.enabled="{ item }">
          <v-switch
            :model-value="item.enabled"
            color="success"
            hide-details
            density="compact"
            @update:model-value="toggleRule(item)"
          />
        </template>

        <template #item.protocol="{ item }">
          <div class="d-flex flex-wrap ga-1">
            <v-chip size="small" :color="getProtocolColor(item.protocol)">
              {{ getForwardProtocolTitle(item.protocol) }}
            </v-chip>
            <v-chip
              v-if="item.tunnel_enabled"
              size="small"
              color="deep-purple"
              variant="tonal"
            >
              隧道 {{ getForwardProtocolTitle(item.tunnel_protocol) }}
            </v-chip>
          </div>
        </template>

        <template #item.listen_port="{ item }">
          <div class="d-flex flex-column">
            <span>入口 {{ item.listen_port }}</span>
            <span v-if="item.tunnel_enabled" class="text-caption text-medium-emphasis">
              出口 {{ item.tunnel_port }}
            </span>
          </div>
        </template>

        <template #item.traffic="{ item }">
          <div class="text-caption">
            <span class="text-success">↓ {{ formatBytes(item.traffic_used) }}</span>
            <span class="ml-2">/ {{ formatBytes(item.traffic_limit) }}</span>
          </div>
          <v-progress-linear
            :model-value="
              item.traffic_limit > 0
                ? (item.traffic_used / item.traffic_limit) * 100
                : 0
            "
            :color="
              getTrafficColor(
                item.traffic_limit > 0
                  ? item.traffic_used / item.traffic_limit
                  : 0
              )
            "
            height="4"
            rounded
          />
        </template>

        <template #item.speed_limit="{ item }">
          {{ item.speed_limit ? item.speed_limit + " Kbps" : "不限速" }}
        </template>

        <template #item.created_at="{ item }">
          {{ formatDate(item.created_at) }}
        </template>

        <template #item.actions="{ item }">
          <v-btn icon size="small" variant="text" @click="editRule(item)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
          <v-btn
            icon
            size="small"
            variant="text"
            color="error"
            @click="deleteRule(item)"
          >
            <v-icon>mdi-delete</v-icon>
          </v-btn>
        </template>

        <template #no-data>
          <div class="text-center py-12">
            <v-icon size="64" color="grey">mdi-traffic-light</v-icon>
            <div class="text-h6 mt-4 text-grey">暂无规则</div>
          </div>
        </template>
      </v-data-table>
    </v-card>

    <v-dialog v-model="showCreateDialog" max-width="720" persistent>
      <v-card>
        <v-card-title>{{ editingRule ? "编辑规则" : "创建规则" }}</v-card-title>
        <v-card-text>
          <v-form ref="formRef">
            <v-text-field
              v-model="form.name"
              label="规则名称"
              :rules="[(v) => !!v || '请输入规则名称']"
              class="mb-2"
            />

            <v-row>
              <v-col cols="12" md="6">
                <v-select
                  v-model="form.protocol"
                  :items="directProtocolOptions"
                  item-title="title"
                  item-value="value"
                  label="直接转发协议"
                  :rules="[(v) => !!v || '请选择协议']"
                />
              </v-col>
              <v-col cols="12" md="6">
                <v-select
                  v-model="form.node_id"
                  :items="availableEntryNodes"
                  item-title="name"
                  item-value="id"
                  label="入口节点"
                  :rules="[(v) => !!v || '请选择入口节点']"
                />
              </v-col>
            </v-row>

            <v-row>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model.number="form.listen_port"
                  label="入口监听端口"
                  type="number"
                  :rules="[(v) => v > 0 || '请输入有效端口']"
                />
              </v-col>
              <v-col cols="12" md="6">
                <v-select v-model="form.mode" :items="modes" label="转发模式" />
              </v-col>
            </v-row>

            <v-alert type="info" variant="tonal" density="compact" class="mb-4">
              {{ selectedProtocolDescription }}
            </v-alert>

            <v-switch
              v-model="form.tunnel_enabled"
              label="通过出口节点建立隧道"
              color="primary"
              class="mb-2"
            />

            <v-expand-transition>
              <div v-if="form.tunnel_enabled">
                <v-row>
                  <v-col cols="12" md="4">
                    <v-select
                      v-model="form.exit_node_id"
                      :items="availableExitNodes"
                      item-title="name"
                      item-value="id"
                      label="出口节点"
                      :rules="[(v) => !!v || '请选择出口节点']"
                    />
                  </v-col>
                  <v-col cols="12" md="4">
                    <v-select
                      v-model="form.tunnel_protocol"
                      :items="tunnelProtocolOptions"
                      item-title="title"
                      item-value="value"
                      label="隧道协议"
                      :rules="[(v) => !!v || '请选择隧道协议']"
                    />
                  </v-col>
                  <v-col cols="12" md="4">
                    <v-text-field
                      v-model.number="form.tunnel_port"
                      label="出口隧道端口"
                      type="number"
                      :rules="[(v) => v > 0 || '请输入有效端口']"
                    />
                  </v-col>
                </v-row>

                <v-alert
                  type="info"
                  variant="tonal"
                  density="compact"
                  class="mb-4"
                >
                  {{ selectedTunnelDescription }}
                </v-alert>
              </div>
            </v-expand-transition>

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

            <v-row v-for="(target, index) in form.targets" :key="index" class="mb-2">
              <v-col cols="12" md="5">
                <v-text-field
                  v-model="target.host"
                  label="目标地址"
                  :rules="[(v) => !!v || '请输入目标地址']"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="3">
                <v-text-field
                  v-model.number="target.port"
                  label="目标端口"
                  type="number"
                  :rules="[(v) => v > 0 || '请输入有效端口']"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="2">
                <v-text-field
                  v-model.number="target.weight"
                  label="权重"
                  type="number"
                  min="1"
                  density="compact"
                />
              </v-col>
              <v-col cols="12" md="2" class="d-flex align-center">
                <v-switch
                  v-model="target.enabled"
                  label="启用"
                  density="compact"
                  hide-details
                  class="mr-2"
                />
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
              <v-col cols="12" md="6">
                <v-text-field
                  v-model.number="form.traffic_limit"
                  label="流量限制 (GB)"
                  type="number"
                />
              </v-col>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model.number="form.speed_limit"
                  label="限速 (Kbps)"
                  type="number"
                  hint="0 或留空表示不限速"
                />
              </v-col>
            </v-row>

            <v-switch v-model="form.enabled" label="启用规则" color="success" />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="closeDialog">取消</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveRule">保存</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card>
        <v-card-title>确认删除</v-card-title>
        <v-card-text>
          确定要删除规则 "{{ deletingRule?.name }}" 吗？此操作不可恢复。
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="showDeleteDialog = false">取消</v-btn>
          <v-btn color="error" :loading="deleting" @click="confirmDelete">删除</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { nodeAPI, ruleAPI } from "@/api";
import { useSnackbar } from "@/composables/useSnackbar";
import {
  directProtocolOptions,
  tunnelProtocolOptions,
  getForwardProtocolDescription,
  getForwardProtocolTitle,
} from "@/constants/forwardProtocols";

const { showSnackbar } = useSnackbar();

const rules = ref([]);
const nodes = ref([]);
const loading = ref(false);
const saving = ref(false);
const deleting = ref(false);

const showCreateDialog = ref(false);
const showDeleteDialog = ref(false);
const editingRule = ref(null);
const deletingRule = ref(null);
const formRef = ref(null);

function defaultForm() {
  return {
    name: "",
    node_id: null,
    protocol: "tcp",
    listen_port: 0,
    targets: [{ host: "", port: 0, weight: 1, enabled: true }],
    traffic_limit: 0,
    speed_limit: 0,
    mode: "direct",
    enabled: true,
    tunnel_enabled: false,
    exit_node_id: null,
    tunnel_protocol: "ws",
    tunnel_port: 0,
  };
}

const form = ref(defaultForm());

const headers = [
  { title: "状态", key: "enabled", width: 80 },
  { title: "名称", key: "name" },
  { title: "协议", key: "protocol" },
  { title: "端口", key: "listen_port" },
  { title: "流量", key: "traffic" },
  { title: "限速", key: "speed_limit" },
  { title: "创建时间", key: "created_at" },
  { title: "操作", key: "actions", width: 100 },
];

const modes = [
  { title: "直连", value: "direct" },
  { title: "轮询", value: "rr" },
  { title: "负载均衡", value: "lb" },
];

const availableEntryNodes = computed(() =>
  (nodes.value || []).filter((node) => {
    const protocols = Array.isArray(node.protocols) ? node.protocols : [];
    return protocols.length === 0 || protocols.includes(form.value.protocol);
  })
);

const availableExitNodes = computed(() =>
  (nodes.value || []).filter((node) => {
    if (node.id === form.value.node_id) {
      return false;
    }
    const protocols = Array.isArray(node.protocols) ? node.protocols : [];
    return (
      protocols.length === 0 || protocols.includes(form.value.tunnel_protocol)
    );
  })
);

const selectedProtocolDescription = computed(() =>
  getForwardProtocolDescription(form.value.protocol)
);

const selectedTunnelDescription = computed(() =>
  form.value.tunnel_enabled
    ? getForwardProtocolDescription(form.value.tunnel_protocol)
    : ""
);

function formatBytes(bytes) {
  if (!bytes) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatDate(date) {
  return dayjs(date).format("YYYY-MM-DD HH:mm");
}

function getProtocolColor(protocol) {
  const colors = {
    tcp: "blue",
    udp: "green",
    tls: "orange",
    mtls: "orange-darken-2",
    ws: "teal",
    mws: "teal-darken-1",
    wss: "cyan",
    mwss: "cyan-darken-1",
    grpc: "indigo",
    h2: "purple",
    h2c: "deep-purple",
    kcp: "lime-darken-1",
    quic: "pink-darken-1",
  };
  return colors[protocol] || "grey";
}

function getTrafficColor(ratio) {
  if (ratio > 0.9) return "error";
  if (ratio > 0.7) return "warning";
  return "success";
}

function editRule(rule) {
  loadRuleDetail(rule.id);
}

function deleteRule(rule) {
  deletingRule.value = rule;
  showDeleteDialog.value = true;
}

function closeDialog() {
  showCreateDialog.value = false;
  editingRule.value = null;
  form.value = defaultForm();
}

function addTarget() {
  form.value.targets.push({ host: "", port: 0, weight: 1, enabled: true });
}

function removeTarget(index) {
  if (form.value.targets.length <= 1) return;
  form.value.targets.splice(index, 1);
}

async function loadRuleDetail(id) {
  loading.value = true;
  try {
    const body = await ruleAPI.get(id);
    const detail = body?.code === 0 ? body.data : null;
    if (!detail?.rule) throw new Error("Invalid rule detail");

    editingRule.value = detail.rule;
    form.value = {
      ...defaultForm(),
      name: detail.rule.name,
      node_id: detail.rule.node_id,
      protocol: detail.rule.protocol || "tcp",
      listen_port: detail.rule.listen_port,
      traffic_limit: detail.rule.traffic_limit
        ? detail.rule.traffic_limit / (1024 * 1024 * 1024)
        : 0,
      speed_limit: detail.rule.speed_limit || 0,
      mode: detail.rule.mode || "direct",
      enabled: !!detail.rule.enabled,
      tunnel_enabled: !!detail.rule.tunnel_enabled,
      exit_node_id: detail.rule.exit_node_id || null,
      tunnel_protocol: detail.rule.tunnel_protocol || "ws",
      tunnel_port: detail.rule.tunnel_port || 0,
      targets:
        detail.targets && detail.targets.length > 0
          ? detail.targets.map((target) => ({
              host: target.host,
              port: target.port,
              weight: target.weight ?? 1,
              enabled: target.enabled !== false,
            }))
          : [{ host: "", port: 0, weight: 1, enabled: true }],
    };

    showCreateDialog.value = true;
  } catch (error) {
    console.error("Failed to load rule detail:", error);
    showSnackbar(error.message || "加载规则失败", "error");
  } finally {
    loading.value = false;
  }
}

async function saveRule() {
  const { valid } = await formRef.value.validate();
  if (!valid) return;

  saving.value = true;
  try {
    const data = {
      name: form.value.name,
      node_id: form.value.node_id,
      protocol: form.value.protocol,
      listen_port: form.value.listen_port,
      traffic_limit: Math.max(
        0,
        Math.round((Number(form.value.traffic_limit) || 0) * 1024 * 1024 * 1024)
      ),
      speed_limit: Math.max(0, Number(form.value.speed_limit) || 0),
      mode: form.value.mode,
      enabled: form.value.enabled,
      tunnel_enabled: !!form.value.tunnel_enabled,
      exit_node_id: form.value.tunnel_enabled ? form.value.exit_node_id : 0,
      tunnel_protocol: form.value.tunnel_enabled ? form.value.tunnel_protocol : "",
      tunnel_port: form.value.tunnel_enabled ? Number(form.value.tunnel_port) || 0 : 0,
      targets: (form.value.targets || [])
        .map((target) => ({
          host: (target.host || "").trim(),
          port: Number(target.port) || 0,
          weight: Math.max(1, Number(target.weight) || 1),
          enabled: target.enabled !== false,
        }))
        .filter((target) => target.host && target.port > 0),
    };

    if (!data.node_id) delete data.node_id;
    if (!data.targets.length) {
      showSnackbar("请至少添加一个有效目标", "error");
      return;
    }
    if (data.speed_limit > 0) {
      showSnackbar("当前规则暂不支持限速", "error");
      return;
    }
    if (data.tunnel_enabled) {
      if (!data.exit_node_id || !data.tunnel_protocol || !data.tunnel_port) {
        showSnackbar("请完整填写隧道出口配置", "error");
        return;
      }
      if (data.exit_node_id === data.node_id) {
        showSnackbar("入口节点与出口节点不能相同", "error");
        return;
      }
    }

    if (editingRule.value) {
      await ruleAPI.update(editingRule.value.id, data);
      showSnackbar("规则已更新", "success");
    } else {
      await ruleAPI.create(data);
      showSnackbar("规则已创建", "success");
    }
    closeDialog();
    loadRules();
  } catch (error) {
    console.error("Failed to save rule:", error);
    showSnackbar(
      error.response?.data?.message || error.message || "保存失败",
      "error"
    );
  } finally {
    saving.value = false;
  }
}

async function confirmDelete() {
  if (!deletingRule.value) return;

  deleting.value = true;
  try {
    await ruleAPI.delete(deletingRule.value.id);
    showDeleteDialog.value = false;
    deletingRule.value = null;
    loadRules();
  } catch (error) {
    console.error("Failed to delete rule:", error);
  } finally {
    deleting.value = false;
  }
}

async function toggleRule(rule) {
  try {
    await ruleAPI.update(rule.id, { enabled: !rule.enabled });
    rule.enabled = !rule.enabled;
  } catch (error) {
    console.error("Failed to toggle rule:", error);
  }
}

async function loadRules() {
  loading.value = true;
  try {
    const response = await ruleAPI.list();
    rules.value = Array.isArray(response.data) ? response.data : [];
  } catch (error) {
    console.error("Failed to load rules:", error);
    rules.value = [];
    showSnackbar(
      error.response?.data?.message || error.message || "加载规则失败",
      "error"
    );
  } finally {
    loading.value = false;
  }
}

async function loadNodes() {
  try {
    const response = await nodeAPI.list();
    nodes.value = Array.isArray(response.data) ? response.data : [];
  } catch (error) {
    nodes.value = [];
    showSnackbar(
      error.response?.data?.message || error.message || "加载节点失败",
      "error"
    );
  }
}

onMounted(async () => {
  await Promise.all([loadRules(), loadNodes()]);
});
</script>
