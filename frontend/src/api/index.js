import client from './axios'

function normalizeListResponse(res) {
  const raw = res?.data
  if (raw && Array.isArray(raw.list)) {
    return {
      ...res,
      raw,
      data: raw.list,
      total: raw.total ?? 0,
      page: raw.page ?? 1
    }
  }
  if (Array.isArray(raw)) {
    return { ...res, raw, data: raw, total: raw.length }
  }
  return { ...res, raw, data: [] }
}

// 认证相关
export const authAPI = {
  login: (data) => client.post('/auth/login', data),
  register: (data) => client.post('/auth/register', data),
  refresh: (data) => client.post('/auth/refresh', data)
}

// 用户相关
export const userAPI = {
  getProfile: () => client.get('/user/profile').then((res) => ({ ...res, data: res.data })),
  updateProfile: (data) => client.put('/user/profile', data),
  getTrafficStats: (params) => client.get('/statistics/traffic', { params })
}

// 节点相关
export const nodeAPI = {
  list: (params) => client.get('/nodes', { params }).then(normalizeListResponse),
  get: (id) => client.get(`/nodes/${id}`)
}

// 规则相关
export const ruleAPI = {
  list: (params) => client.get('/rules', { params }).then(normalizeListResponse),
  get: (id) => client.get(`/rules/${id}`),
  create: (data) => client.post('/rules', data),
  update: (id, data) => client.put(`/rules/${id}`, data),
  delete: (id) => client.delete(`/rules/${id}`)
}

// 套餐相关
export const packageAPI = {
  list: () => client.get('/packages').then(normalizeListResponse)
}

// 订单相关
export const orderAPI = {
  list: (params) => client.get('/orders', { params }).then(normalizeListResponse),
  create: (data) => client.post('/orders', data)
}

// 充值/支付相关
export const paymentAPI = {
  list: () => client.get('/payments').then(normalizeListResponse)
}

export const depositAPI = {
  create: (data) => client.post('/deposit', data),
  callback: (params) => client.get('/deposit/callback', { params })
}

// 管理后台
export const adminAPI = {
  site: {
    get: () => client.get('/admin/site').then((res) => ({ ...res, data: res.data })),
    update: (data) => client.put('/admin/site', data)
  },
  nodes: {
    list: (params) => client.get('/admin/nodes', { params }).then(normalizeListResponse),
    create: (data) => client.post('/admin/nodes', data),
    update: (id, data) => client.put(`/admin/nodes/${id}`, data),
    delete: (id) => client.delete(`/admin/nodes/${id}`),
    reload: (id) => client.post(`/admin/nodes/${id}/reload`)
  },
  users: {
    list: (params) => client.get('/admin/users', { params }).then(normalizeListResponse),
    create: (data) => client.post('/admin/users', data),
    update: (id, data) => client.put(`/admin/users/${id}`, data),
    delete: (id) => client.delete(`/admin/users/${id}`),
    adjustBalance: (id, data) => client.post(`/admin/users/${id}/balance`, data)
  },
  packages: {
    list: (params) => client.get('/admin/packages', { params }).then(normalizeListResponse),
    create: (data) => client.post('/admin/packages', data),
    update: (id, data) => client.put(`/admin/packages/${id}`, data),
    delete: (id) => client.delete(`/admin/packages/${id}`)
  },
  orders: {
    list: (params) => client.get('/admin/orders', { params }).then(normalizeListResponse),
    updateStatus: (id, data) => client.put(`/admin/orders/${id}/status`, data)
  },
  payments: {
    list: () => client.get('/admin/payments').then(normalizeListResponse),
    create: (data) => client.post('/admin/payments', data),
    update: (id, data) => client.put(`/admin/payments/${id}`, data),
    delete: (id) => client.delete(`/admin/payments/${id}`)
  },
  nodeGroups: {
    list: () => client.get('/admin/node-groups').then(normalizeListResponse),
    create: (data) => client.post('/admin/node-groups', data),
    update: (id, data) => client.put(`/admin/node-groups/${id}`, data),
    delete: (id) => client.delete(`/admin/node-groups/${id}`)
  },
  userGroups: {
    list: () => client.get('/admin/user-groups').then(normalizeListResponse),
    create: (data) => client.post('/admin/user-groups', data),
    update: (id, data) => client.put(`/admin/user-groups/${id}`, data),
    delete: (id) => client.delete(`/admin/user-groups/${id}`)
  },
  rules: {
    count: () => client.get('/admin/rules/count').then((res) => ({ ...res, data: res.data?.total ?? 0 }))
  }
}

export default {
  authAPI,
  userAPI,
  nodeAPI,
  ruleAPI,
  packageAPI,
  orderAPI,
  paymentAPI,
  depositAPI,
  adminAPI
}

