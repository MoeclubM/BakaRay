import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

// Mock localStorage
const localStorageMock = {
  store: {},
  getItem: vi.fn((key) => localStorageMock.store[key] || null),
  setItem: vi.fn((key, value) => { localStorageMock.store[key] = value }),
  removeItem: vi.fn((key) => { delete localStorageMock.store[key] }),
  clear: vi.fn(() => { localStorageMock.store = {} })
}

Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true
})

// Mock API modules
const { mockAuthAPI, mockUserAPI } = vi.hoisted(() => ({
  mockAuthAPI: {
    login: vi.fn(),
    register: vi.fn(),
    refresh: vi.fn()
  },
  mockUserAPI: {
    getProfile: vi.fn()
  }
}))

vi.mock('@/api', () => ({
  authAPI: mockAuthAPI,
  userAPI: mockUserAPI
}))

describe('auth store', () => {
  let pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    localStorageMock.store = {}
    vi.clearAllMocks()
  })

  afterEach(() => {
    setActivePinia(null)
  })

  describe('initial state', () => {
    it('should initialize with null token and user', () => {
      const store = useAuthStore()
      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })

    it('should read token from localStorage on init', () => {
      localStorageMock.store.token = 'test-token'
      localStorageMock.store.user = JSON.stringify({ id: 1, username: 'test' })
      const store = useAuthStore()
      expect(store.token).toBe('test-token')
      expect(store.user).toEqual({ id: 1, username: 'test' })
    })

    it('should compute isAuthenticated correctly', () => {
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(false)
      store.token = 'test-token'
      expect(store.isAuthenticated).toBe(true)
    })

    it('should compute isAdmin correctly', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
      store.user = { role: 'user' }
      expect(store.isAdmin).toBe(false)
      store.user = { role: 'admin' }
      expect(store.isAdmin).toBe(true)
    })
  })

  describe('login', () => {
    it('should return false when credentials are invalid', async () => {
      mockAuthAPI.login.mockResolvedValue({ code: -1, message: 'Invalid credentials' })
      const store = useAuthStore()
      const result = await store.login('test', 'wrongpassword')
      expect(result).toBe(false)
      expect(store.error).toBe('Invalid credentials')
    })

    it('should login successfully and set token', async () => {
      mockAuthAPI.login.mockResolvedValue({ code: 0, token: 'test-token' })
      mockUserAPI.getProfile.mockResolvedValue({ code: 0, data: { id: 1, username: 'testuser' } })
      const store = useAuthStore()
      const result = await store.login('testuser', 'password')
      expect(result).toBe(true)
      expect(store.token).toBe('test-token')
      expect(store.user).toEqual({ id: 1, username: 'testuser' })
    })

    it('should set loading state during login', async () => {
      mockAuthAPI.login.mockImplementation(() => new Promise(resolve => setTimeout(() => resolve({ code: 0, token: 'test' }), 10)))
      mockUserAPI.getProfile.mockResolvedValue({ code: 0, data: {} })
      const store = useAuthStore()
      const loginPromise = store.login('test', 'password')
      expect(store.loading).toBe(true)
      await loginPromise
      expect(store.loading).toBe(false)
    })
  })

  describe('register', () => {
    it('should register successfully', async () => {
      mockAuthAPI.register.mockResolvedValue({ code: 0 })
      const store = useAuthStore()
      const result = await store.register('newuser', 'password123', 'INVITE123')
      expect(result).toBe(true)
      expect(store.error).toBeNull()
    })

    it('should handle registration failure', async () => {
      mockAuthAPI.register.mockResolvedValue({ code: -1, message: 'Username exists' })
      const store = useAuthStore()
      const result = await store.register('existinguser', 'password123', 'INVITE123')
      expect(result).toBe(false)
      expect(store.error).toBe('Username exists')
    })
  })

  describe('logout', () => {
    it('should clear all state on logout', () => {
      localStorageMock.store.token = 'test-token'
      localStorageMock.store.user = JSON.stringify({ id: 1 })
      const store = useAuthStore()
      store.logout()
      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('token')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('user')
    })
  })

  describe('fetchProfile', () => {
    it('should return false when no token', async () => {
      const store = useAuthStore()
      const result = await store.fetchProfile()
      expect(result).toBe(false)
    })

    it('should fetch and set user profile', async () => {
      mockUserAPI.getProfile.mockResolvedValue({
        code: 0,
        data: { id: 1, username: 'testuser', user_group_name: '正式用户组' }
      })
      const store = useAuthStore()
      store.token = 'test-token'
      const result = await store.fetchProfile()
      expect(result).toBe(true)
      expect(store.user).toEqual({ id: 1, username: 'testuser', user_group_name: '正式用户组' })
    })

    it('should logout on profile fetch failure', async () => {
      mockUserAPI.getProfile.mockRejectedValue(new Error('Unauthorized'))
      const store = useAuthStore()
      store.token = 'test-token'
      store.user = { id: 1 }
      const result = await store.fetchProfile()
      expect(result).toBe(false)
      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
    })
  })

  describe('refreshToken', () => {
    it('should return false when no token', async () => {
      const store = useAuthStore()
      const result = await store.refreshToken()
      expect(result).toBe(false)
    })

    it('should refresh token successfully', async () => {
      mockAuthAPI.refresh.mockResolvedValue({ code: 0, token: 'new-token' })
      const store = useAuthStore()
      store.token = 'old-token'
      const result = await store.refreshToken()
      expect(result).toBe(true)
      expect(store.token).toBe('new-token')
    })

    it('should keep current session on refresh rejection response', async () => {
      mockAuthAPI.refresh.mockResolvedValue({ code: -1 })
      const store = useAuthStore()
      store.token = 'old-token'
      store.user = { id: 1 }
      const result = await store.refreshToken()
      expect(result).toBe(false)
      expect(store.token).toBe('old-token')
      expect(store.user).toEqual({ id: 1 })
    })
  })

  describe('localStorage persistence', () => {
    it('should persist token and user after successful login', async () => {
      mockAuthAPI.login.mockResolvedValue({ code: 0, token: 'persisted-token' })
      mockUserAPI.getProfile.mockResolvedValue({ code: 0, data: { id: 1, username: 'test' } })

      const store = useAuthStore()
      const result = await store.login('test', 'password')

      expect(result).toBe(true)
      expect(localStorageMock.setItem).toHaveBeenCalledWith('token', 'persisted-token')
      expect(localStorageMock.setItem).toHaveBeenCalledWith('user', JSON.stringify({ id: 1, username: 'test' }))
    })

    it('should remove persisted data on logout', () => {
      const store = useAuthStore()
      store.token = 'persisted-token'
      store.user = { id: 1, username: 'test' }

      store.logout()

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('token')
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('user')
    })
  })
})
