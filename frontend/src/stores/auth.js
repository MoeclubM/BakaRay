import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authAPI, userAPI } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || null)
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  const loading = ref(false)
  const error = ref(null)

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function persist() {
    if (token.value) localStorage.setItem('token', token.value)
    else localStorage.removeItem('token')

    if (user.value) localStorage.setItem('user', JSON.stringify(user.value))
    else localStorage.removeItem('user')
  }

  function getErrorMessage(err, fallback) {
    return err?.message || err?.msg || err?.error || fallback
  }

  async function fetchProfile() {
    if (!token.value) return false
    try {
      const response = await userAPI.getProfile()
      if (response.code !== 0) throw new Error(response.message || '获取用户信息失败')
      user.value = response.data
      persist()
      return true
    } catch {
      logout()
      return false
    }
  }

  async function login(username, password) {
    loading.value = true
    error.value = null
    try {
      const response = await authAPI.login({ username, password })
      if (response.code !== 0 || !response.token) {
        throw new Error(response.message || '登录失败')
      }

      token.value = response.token
      persist()
      await fetchProfile()
      return true
    } catch (err) {
      error.value = getErrorMessage(err, '登录失败')
      return false
    } finally {
      loading.value = false
    }
  }

  async function register(username, password, inviteCode) {
    loading.value = true
    error.value = null
    try {
      const response = await authAPI.register({
        username,
        password,
        invite_code: inviteCode
      })
      if (response.code !== 0) throw new Error(response.message || '注册失败')
      return true
    } catch (err) {
      error.value = getErrorMessage(err, '注册失败')
      return false
    } finally {
      loading.value = false
    }
  }

  async function refreshToken() {
    try {
      if (!token.value) return false
      const response = await authAPI.refresh({ token: token.value })
      if (response.code !== 0 || !response.token) return false
      token.value = response.token
      persist()
      return true
    } catch {
      logout()
      return false
    }
  }

  function logout() {
    token.value = null
    user.value = null
    persist()
  }

  return {
    token,
    user,
    loading,
    error,
    isAuthenticated,
    isAdmin,
    login,
    register,
    refreshToken,
    fetchProfile,
    logout
  }
})

