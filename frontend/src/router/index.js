import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { guest: true }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { guest: true }
  },
  {
    path: '/',
    component: () => import('@/components/AppLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/DashboardView.vue')
      },
      {
        path: 'nodes',
        name: 'Nodes',
        component: () => import('@/views/NodesView.vue')
      },
      {
        path: 'rules',
        name: 'Rules',
        component: () => import('@/views/RulesView.vue')
      },
      {
        path: 'packages',
        name: 'Packages',
        component: () => import('@/views/PackagesView.vue')
      },
      {
        path: 'orders',
        name: 'Orders',
        component: () => import('@/views/OrdersView.vue')
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/ProfileView.vue')
      },
      // Admin routes
      {
        path: 'admin',
        name: 'Admin',
        component: () => import('@/views/admin/AdminDashboard.vue'),
        children: [
          {
            path: '',
            name: 'AdminOverview',
            component: () => import('@/views/admin/AdminOverview.vue')
          },
          {
            path: 'nodes',
            name: 'AdminNodes',
            component: () => import('@/views/admin/AdminNodes.vue')
          },
          {
            path: 'users',
            name: 'AdminUsers',
            component: () => import('@/views/admin/AdminUsers.vue')
          },
          {
            path: 'packages',
            name: 'AdminPackages',
            component: () => import('@/views/admin/AdminPackages.vue')
          },
          {
            path: 'orders',
            name: 'AdminOrders',
            component: () => import('@/views/admin/AdminOrders.vue')
          },
          {
            path: 'node-groups',
            name: 'AdminNodeGroups',
            component: () => import('@/views/admin/AdminNodeGroups.vue')
          },
          {
            path: 'user-groups',
            name: 'AdminUserGroups',
            component: () => import('@/views/admin/AdminUserGroups.vue')
          },
          {
            path: 'payments',
            name: 'AdminPayments',
            component: () => import('@/views/admin/AdminPayments.vue')
          },
          {
            path: 'settings',
            name: 'AdminSettings',
            component: () => import('@/views/admin/AdminSettings.vue')
          }
        ]
      }
    ]
  },
  {
    path: '/deposit/callback',
    name: 'DepositCallback',
    component: () => import('@/views/DepositCallback.vue')
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.meta.guest && authStore.isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router
