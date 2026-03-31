// WPP Gateway - Alpine.js App

document.addEventListener('alpine:init', () => {

  // Toast Store
  Alpine.store('toast', {
    items: [],
    show(type, message, duration) {
      const id = Date.now()
      const finalDuration = duration || 4000
      this.items = [...this.items, { id, type, message, removing: false }]
      setTimeout(() => this.dismiss(id), finalDuration)
    },
    dismiss(id) {
      this.items = this.items.map(t =>
        t.id === id ? { ...t, removing: true } : t
      )
      setTimeout(() => {
        this.items = this.items.filter(t => t.id !== id)
      }, 300)
    },
  })

  // Theme Store
  Alpine.store('theme', {
    dark: false,
    init() {
      const saved = localStorage.getItem('wpp_theme')
      if (saved) {
        this.dark = saved === 'dark'
      } else {
        this.dark = window.matchMedia('(prefers-color-scheme: dark)').matches
      }
      this.apply()
    },
    toggle() {
      this.dark = !this.dark
      localStorage.setItem('wpp_theme', this.dark ? 'dark' : 'light')
      this.apply()
    },
    apply() {
      document.documentElement.classList.toggle('dark', this.dark)
    }
  })

  // Lang Store
  Alpine.store('lang', {
    current: getLang(),
    toggle() {
      this.current = this.current === 'pt' ? 'en' : 'pt'
      setLang(this.current)
    }
  })

  // Auth Store
  Alpine.store('auth', {
    authenticated: false,
    tenant: null,

    async init() {
      const key = Api.getApiKey()
      if (key) {
        try {
          const res = await Api.validateKey()
          this.tenant = res.data
          this.authenticated = true
        } catch {
          Api.clearApiKey()
          this.authenticated = false
        }
      }
    },

    async login(apiKey) {
      Api.setApiKey(apiKey)
      try {
        const res = await Api.validateKey()
        this.tenant = res.data
        this.authenticated = true
        Alpine.store('toast').show('success', t('connected_success'))
        Alpine.store('router').navigate('dashboard')
      } catch (err) {
        Api.clearApiKey()
        this.authenticated = false
        throw err
      }
    },

    logout() {
      Api.clearApiKey()
      this.authenticated = false
      this.tenant = null
      Alpine.store('router').navigate('login')
    },
  })

  // Router Store
  Alpine.store('router', {
    page: 'login',
    params: {},
    history: [],

    init() {
      this.handleHash()
      window.addEventListener('hashchange', () => this.handleHash())
    },

    handleHash() {
      const hash = window.location.hash.slice(1) || '/login'
      const routes = [
        { pattern: /^\/dashboard$/, page: 'dashboard' },
        { pattern: /^\/groups$/, page: 'groups' },
        { pattern: /^\/groups\/([^/]+)$/, page: 'group-detail', paramKey: 'groupId' },
        { pattern: /^\/groups\/([^/]+)\/instances\/([^/]+)$/, page: 'instance-detail', paramKeys: ['groupId', 'instanceId'] },
        { pattern: /^\/broadcasts$/, page: 'broadcasts' },
        { pattern: /^\/broadcasts\/([^/]+)\/([^/]+)$/, page: 'broadcast-detail', paramKeys: ['groupId', 'broadcastId'] },
        { pattern: /^\/logs$/, page: 'logs' },
        { pattern: /^\/login$/, page: 'login' },
      ]

      for (const route of routes) {
        const match = hash.match(route.pattern)
        if (match) {
          const params = {}
          if (route.paramKey) {
            params[route.paramKey] = match[1]
          } else if (route.paramKeys) {
            route.paramKeys.forEach((key, i) => {
              params[key] = match[i + 1]
            })
          }
          this.page = route.page
          this.params = params
          return
        }
      }

      this.page = Alpine.store('auth').authenticated ? 'dashboard' : 'login'
      this.params = {}
    },

    navigate(page, params) {
      const paramsCopy = params ? { ...params } : {}
      let hash = '/' + page
      if (page === 'group-detail' && paramsCopy.groupId) {
        hash = `/groups/${paramsCopy.groupId}`
      } else if (page === 'instance-detail' && paramsCopy.groupId && paramsCopy.instanceId) {
        hash = `/groups/${paramsCopy.groupId}/instances/${paramsCopy.instanceId}`
      } else if (page === 'broadcast-detail' && paramsCopy.groupId && paramsCopy.broadcastId) {
        hash = `/broadcasts/${paramsCopy.groupId}/${paramsCopy.broadcastId}`
      }
      window.location.hash = hash
    },

    back() {
      window.history.back()
    },
  })

  // UI Store
  Alpine.store('ui', {
    sidebarOpen: window.innerWidth >= 1024,
    sidebarCollapsed: false,
    modalOpen: false,
    modalContent: '',
    confirmCallback: null,

    toggleSidebar() {
      if (window.innerWidth < 1024) {
        this.sidebarOpen = !this.sidebarOpen
      } else {
        this.sidebarCollapsed = !this.sidebarCollapsed
      }
    },

    openModal() { this.modalOpen = true },
    closeModal() {
      this.modalOpen = false
      this.confirmCallback = null
    },

    confirm(message, callback) {
      this.modalContent = message
      this.confirmCallback = callback
      this.modalOpen = true
    },

    executeConfirm() {
      if (this.confirmCallback) {
        this.confirmCallback()
      }
      this.closeModal()
    },
  })
})
