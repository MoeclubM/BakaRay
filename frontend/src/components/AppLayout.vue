<template>
  <v-app>
    <!-- 顶部导航栏 -->
    <v-app-bar elevation="0" color="surface">
      <v-app-bar-nav-icon @click="drawer = !drawer" />

      <v-toolbar-title>
        <span class="font-weight-bold">{{ siteName }}</span>
      </v-toolbar-title>

      <v-spacer />

      <!-- 主题切换 -->
      <v-btn icon @click="toggleTheme">
        <v-icon>{{ isDark ? 'mdi-white-balance-sunny' : 'mdi-moon-waning-crescent' }}</v-icon>
      </v-btn>

      <!-- 用户菜单 -->
      <v-menu>
        <template v-slot:activator="{ props }">
          <v-btn icon v-bind="props">
            <v-avatar size="32" color="primary">
              <span class="text-body-2">{{ userInitials }}</span>
            </v-avatar>
          </v-btn>
        </template>
        <v-list>
          <v-list-item prepend-icon="mdi-account" :title="user?.username" :subtitle="user?.email" />
          <v-divider />
          <v-list-item prepend-icon="mdi-account-circle" title="个人中心" to="/profile" />
          <v-list-item v-if="isAdmin" prepend-icon="mdi-cog" title="管理后台" to="/admin" />
          <v-divider />
          <v-list-item prepend-icon="mdi-logout" title="退出登录" @click="logout" />
        </v-list>
      </v-menu>
    </v-app-bar>

    <!-- 侧边导航栏 -->
    <v-navigation-drawer v-model="drawer" temporary>
      <v-list nav>
        <v-list-item
          v-for="item in menuItems"
          :key="item.title"
          :to="item.to"
          :prepend-icon="item.icon"
          :title="item.title"
          :value="item.title"
          rounded="lg"
        />
      </v-list>

      <template v-if="isAdmin">
        <v-divider class="my-2" />
        <v-list-subheader>管理后台</v-list-subheader>
        <v-list nav>
          <v-list-item
            v-for="item in adminMenuItems"
            :key="item.title"
            :to="item.to"
            :prepend-icon="item.icon"
            :title="item.title"
            :value="item.title"
            rounded="lg"
          />
        </v-list>
      </template>
    </v-navigation-drawer>

    <!-- 主内容区 -->
    <v-main>
      <v-container fluid class="pa-4">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useTheme } from 'vuetify'
import { useAuthStore } from '@/stores/auth'
import api from '@/api'

const router = useRouter()
const theme = useTheme()
const authStore = useAuthStore()

const drawer = ref(false)
const siteName = ref('BakaRay')

const isDark = computed(() => theme.global.current.value.dark)
const user = computed(() => authStore.user)
const isAdmin = computed(() => authStore.isAdmin)

const userInitials = computed(() => {
  if (!user.value?.username) return '?'
  return user.value.username.substring(0, 2).toUpperCase()
})

const menuItems = [
  { title: '仪表盘', icon: 'mdi-view-dashboard', to: '/' },
  { title: '节点列表', icon: 'mdi-server-network', to: '/nodes' },
  { title: '转发规则', icon: 'mdi-routes', to: '/rules' },
  { title: '套餐购买', icon: 'mdi-package-variant', to: '/packages' },
  { title: '我的订单', icon: 'mdi-receipt', to: '/orders' }
]

const adminMenuItems = [
  { title: '概览', icon: 'mdi-chart-bar', to: '/admin' },
  { title: '节点管理', icon: 'mdi-server', to: '/admin/nodes' },
  { title: '用户管理', icon: 'mdi-account-group', to: '/admin/users' },
  { title: '套餐配置', icon: 'mdi-package-variant-closed', to: '/admin/packages' },
  { title: '订单管理', icon: 'mdi-cart', to: '/admin/orders' },
  { title: '节点组', icon: 'mdi-lan', to: '/admin/node-groups' },
  { title: '用户组', icon: 'mdi-account-multiple', to: '/admin/user-groups' },
  { title: '支付配置', icon: 'mdi-credit-card', to: '/admin/payments' },
  { title: '站点设置', icon: 'mdi-cog', to: '/admin/settings' }
]

function toggleTheme() {
  theme.global.name.value = isDark.value ? 'light' : 'dark'
}

async function logout() {
  authStore.logout()
  router.push('/login')
}

// 加载站点名称
async function loadSiteName() {
  try {
    const response = await api.get('/admin/site')
    if (response.data?.site_name) {
      siteName.value = response.data.site_name
    }
  } catch {
    // 忽略错误
  }
}

loadSiteName()
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
