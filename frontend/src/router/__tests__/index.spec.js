import { beforeEach, describe, expect, it, vi } from 'vitest'

const authState = vi.hoisted(() => ({
  isAuthenticated: false,
  isAdmin: false
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('@/components/AppLayout.vue', () => ({
  default: { template: '<div>AppLayout</div>' }
}))

vi.mock('@/components/AdminLayout.vue', () => ({
  default: { template: '<div>AdminLayout</div>' }
}))

vi.mock('@/views/LoginView.vue', () => ({
  default: { template: '<div>Login</div>' }
}))

vi.mock('@/views/DashboardView.vue', () => ({
  default: { template: '<div>Dashboard</div>' }
}))

vi.mock('@/views/admin/AdminLoginView.vue', () => ({
  default: { template: '<div>AdminLogin</div>' }
}))

vi.mock('@/views/admin/AdminNodes.vue', () => ({
  default: { template: '<div>AdminNodes</div>' }
}))

async function loadRouter() {
  vi.resetModules()
  const module = await import('../index.js')
  return module.default
}

describe('router', () => {
  beforeEach(() => {
    authState.isAuthenticated = false
    authState.isAdmin = false
  })

  it('defines the expected public and admin routes', async () => {
    const router = await loadRouter()
    const routeNames = router.getRoutes().map((route) => route.name).filter(Boolean)

    expect(routeNames).toContain('Login')
    expect(routeNames).toContain('Register')
    expect(routeNames).toContain('Dashboard')
    expect(routeNames).toContain('Rules')
    expect(routeNames).toContain('AdminLogin')
    expect(routeNames).toContain('AdminNodes')
    expect(routeNames).toContain('NotFound')
    expect(routeNames).not.toContain('Profile')
  })

  it('marks auth and admin meta correctly', async () => {
    const router = await loadRouter()

    const loginRoute = router.getRoutes().find((route) => route.name === 'Login')
    const adminLoginRoute = router.getRoutes().find((route) => route.name === 'AdminLogin')
    const rulesRoute = router.getRoutes().find((route) => route.name === 'Rules')
    const adminNodesRoute = router.getRoutes().find((route) => route.name === 'AdminNodes')

    expect(loginRoute.meta.guest).toBe(true)
    expect(adminLoginRoute.meta.guest).toBe(true)
    expect(rulesRoute).toBeDefined()
    expect(adminNodesRoute).toBeDefined()
  })

  it('keeps /rules available and sends /profile to not found', async () => {
    const router = await loadRouter()

    expect(router.resolve('/rules').name).toBe('Rules')
    expect(router.resolve('/profile').name).toBe('NotFound')
  })

  it('redirects unauthenticated frontend access to login', async () => {
    const router = await loadRouter()

    await router.push('/rules')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('Login')
    expect(router.currentRoute.value.query.redirect).toBe('/rules')
  })

  it('redirects unauthenticated admin access to admin login', async () => {
    const router = await loadRouter()

    await router.push('/admin/nodes')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('AdminLogin')
    expect(router.currentRoute.value.query.redirect).toBe('/admin/nodes')
  })

  it('redirects authenticated non-admin users away from admin pages', async () => {
    authState.isAuthenticated = true
    const router = await loadRouter()

    await router.push('/admin/nodes')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('AdminLogin')
  })

  it('redirects authenticated users away from guest pages', async () => {
    authState.isAuthenticated = true
    const router = await loadRouter()

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('Dashboard')
  })
})
