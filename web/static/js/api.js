// WPP Gateway - API Client

const Api = {
  baseUrl: '',

  getApiKey() {
    return localStorage.getItem('wpp_api_key') || ''
  },

  setApiKey(key) {
    localStorage.setItem('wpp_api_key', key)
  },

  clearApiKey() {
    localStorage.removeItem('wpp_api_key')
  },

  async request(method, path, body) {
    const headers = { 'Content-Type': 'application/json' }
    const apiKey = this.getApiKey()
    if (apiKey) {
      headers['X-API-Key'] = apiKey
    }

    const opts = { method, headers }
    if (body && method !== 'GET') {
      opts.body = JSON.stringify(body)
    }

    const response = await fetch(`${this.baseUrl}${path}`, opts)

    if (response.status === 401) {
      this.clearApiKey()
      Alpine.store('auth').logout()
      Alpine.store('toast').show('error', t('session_expired'))
      throw new Error('Unauthorized')
    }

    if (response.status === 204) {
      return null
    }

    const data = await response.json()

    if (!data.success) {
      const msg = data.error?.message || t('unknown_error')
      throw new Error(msg)
    }

    return data
  },

  get(path) { return this.request('GET', path) },
  post(path, body) { return this.request('POST', path, body) },
  patch(path, body) { return this.request('PATCH', path, body) },
  put(path, body) { return this.request('PUT', path, body) },
  del(path) { return this.request('DELETE', path) },

  // Auth
  async validateKey() {
    return this.get('/v1/account')
  },

  async register(name) {
    const response = await fetch('/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    return response.json()
  },

  // Account
  getAccount() { return this.get('/v1/account') },
  getUsage() { return this.get('/v1/account/usage') },
  updateAccount(data) { return this.patch('/v1/account', data) },
  getAuditLog(params) {
    const qs = new URLSearchParams(params).toString()
    return this.get(`/v1/account/audit-log?${qs}`)
  },

  // Logs
  getLogs(params) {
    const qs = new URLSearchParams(params).toString()
    return this.get(`/v1/logs?${qs}`)
  },

  // Groups
  listGroups() { return this.get('/v1/groups') },
  getGroup(id) { return this.get(`/v1/groups/${id}`) },
  createGroup(data) { return this.post('/v1/groups', data) },
  updateGroup(id, data) { return this.patch(`/v1/groups/${id}`, data) },
  deleteGroup(id) { return this.del(`/v1/groups/${id}`) },
  getGroupStatus(id) { return this.get(`/v1/groups/${id}/status`) },
  pauseGroup(id) { return this.post(`/v1/groups/${id}/pause`) },
  resumeGroup(id) { return this.post(`/v1/groups/${id}/resume`) },

  // Instances
  listInstances(groupId) { return this.get(`/v1/groups/${groupId}/instances`) },
  getInstance(id) { return this.get(`/v1/instances/${id}`) },
  createInstance(groupId, data) { return this.post(`/v1/groups/${groupId}/instances`, data) },
  updateInstance(id, data) { return this.patch(`/v1/instances/${id}`, data) },
  deleteInstance(id) { return this.del(`/v1/instances/${id}`) },
  getQRCode(id) { return this.get(`/v1/instances/${id}/qrcode`) },
  pairPhone(id, phone) { return this.post(`/v1/instances/${id}/pair`, { phone }) },
  connectInstance(id) { return this.post(`/v1/instances/${id}/connect`) },
  disconnectInstance(id) { return this.post(`/v1/instances/${id}/disconnect`) },
  restartInstance(id) { return this.post(`/v1/instances/${id}/restart`) },
  getInstanceStatus(id) { return this.get(`/v1/instances/${id}/status`) },

  // Messages
  sendMessage(groupId, data) { return this.post(`/v1/groups/${groupId}/messages/send`, data) },
  listMessages(groupId, params) {
    const qs = new URLSearchParams(params).toString()
    return this.get(`/v1/groups/${groupId}/messages?${qs}`)
  },
  getMessageStatus(groupId, msgId) {
    return this.get(`/v1/groups/${groupId}/messages/${msgId}/status`)
  },

  // Broadcasts
  createBroadcast(groupId, data) { return this.post(`/v1/groups/${groupId}/messages/broadcast`, data) },
  listBroadcasts(groupId) { return this.get(`/v1/groups/${groupId}/messages/broadcast`) },
  getBroadcast(groupId, id) { return this.get(`/v1/groups/${groupId}/messages/broadcast/${id}`) },
  pauseBroadcast(groupId, id) { return this.post(`/v1/groups/${groupId}/messages/broadcast/${id}/pause`) },
  resumeBroadcast(groupId, id) { return this.post(`/v1/groups/${groupId}/messages/broadcast/${id}/resume`) },
  cancelBroadcast(groupId, id) { return this.post(`/v1/groups/${groupId}/messages/broadcast/${id}/cancel`) },

  // Metrics
  getGroupMetrics(groupId) { return this.get(`/v1/groups/${groupId}/metrics`) },
  getDailyMetrics(groupId, params) {
    const qs = new URLSearchParams(params).toString()
    return this.get(`/v1/groups/${groupId}/metrics/daily?${qs}`)
  },
  getInstanceMetrics(groupId) { return this.get(`/v1/groups/${groupId}/metrics/instances`) },

  // Blacklist
  listBlacklist(groupId) { return this.get(`/v1/groups/${groupId}/blacklist`) },
  addToBlacklist(groupId, numbers, reason) {
    return this.post(`/v1/groups/${groupId}/blacklist`, { numbers, reason })
  },
  removeFromBlacklist(groupId, number) {
    return this.del(`/v1/groups/${groupId}/blacklist/${number}`)
  },

  // Webhooks
  getWebhook(groupId) { return this.get(`/v1/groups/${groupId}/webhook`) },
  configureWebhook(groupId, data) { return this.put(`/v1/groups/${groupId}/webhook`, data) },
  testWebhook(groupId) { return this.post(`/v1/groups/${groupId}/webhook/test`) },
}
