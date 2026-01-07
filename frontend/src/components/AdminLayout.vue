<template>
  <v-app>
    <v-navigation-drawer v-model="drawer" app>
      <v-list-item class="px-4 py-3">
        <v-list-item-title class="text-h6">BakaRay</v-list-item-title>
        <v-list-item-subtitle>管理后台</v-list-item-subtitle>
      </v-list-item>

      <v-divider />

      <v-list nav density="compact">
        <v-list-item to="/admin" prepend-icon="mdi-view-dashboard" title="概览" exact />
        <v-list-item to="/admin/nodes" prepend-icon="mdi-server-network" title="节点管理" />
        <v-list-item to="/admin/users" prepend-icon="mdi-account-group" title="用户管理" />
        <v-list-item to="/admin/packages" prepend-icon="mdi-package-variant" title="套餐配置" />
        <v-list-item to="/admin/orders" prepend-icon="mdi-receipt" title="订单管理" />
        <v-list-item to="/admin/node-groups" prepend-icon="mdi-lan" title="节点组" />
        <v-list-item to="/admin/user-groups" prepend-icon="mdi-account-multiple" title="用户组" />
        <v-list-item to="/admin/payments" prepend-icon="mdi-credit-card" title="支付配置" />
        <v-list-item to="/admin/settings" prepend-icon="mdi-cog" title="站点设置" />
      </v-list>
    </v-navigation-drawer>

    <v-app-bar app flat color="error">
      <v-app-bar-nav-icon @click="drawer = !drawer" />
      <v-toolbar-title class="text-white">{{ currentTitle }}</v-toolbar-title>
      <v-spacer />
      <v-btn icon @click="toggleTheme">
        <v-icon class="text-white">{{ isDark ? 'mdi-weather-sunny' : 'mdi-weather-night' }}</v-icon>
      </v-btn>
      <v-menu>
        <template v-slot:activator="{ props }">
          <v-btn icon v-bind="props">
            <v-avatar size="32" color="primary">
              <span class="text-body-2">{{ userInitial }}</span>
            </v-avatar>
          </v-btn>
        </template>
        <v-list>
          <v-list-item>
            <v-list-item-title>{{ authStore.user?.username }}</v-list-item-title>
            <v-list-item-subtitle class="text-error">管理员</v-list-item-subtitle>
          </v-list-item>
          <v-divider />
          <v-list-item @click="logout">
            <template v-slot:prepend>
              <v-icon>mdi-logout</v-icon>
            </template>
            <v-list-item-title>退出登录</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-app-bar>

    <v-main>
      <v-container fluid>
        <router-view />
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from 'vuetify'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const theme = useTheme()
const authStore = useAuthStore()

const drawer = ref(true)
const isDark = computed(() => theme.global.current.value.dark)
const currentTitle = computed(() => route.meta.title || route.name || '管理后台')
const userInitial = computed(() => authStore.user?.username?.charAt(0).toUpperCase() || 'U')

const toggleTheme = () => {
  theme.global.name.value = isDark.value ? 'light' : 'dark'
}

const logout = async () => {
  authStore.logout()
  router.push('/login')
}
</script>
