// WPP Gateway - Utility Functions

const StatusColors = {
  disconnected: { bg: 'bg-neutral-300 dark:bg-neutral-700', text: 'text-neutral-700 dark:text-neutral-300', dot: 'bg-neutral-500' },
  connecting: { bg: 'bg-yellow-100 dark:bg-yellow-900/40', text: 'text-yellow-800 dark:text-yellow-300', dot: 'bg-yellow-500' },
  available: { bg: 'bg-emerald-100 dark:bg-emerald-900/40', text: 'text-emerald-800 dark:text-emerald-300', dot: 'bg-emerald-500' },
  resting: { bg: 'bg-blue-100 dark:bg-blue-900/40', text: 'text-blue-800 dark:text-blue-300', dot: 'bg-blue-500' },
  warming: { bg: 'bg-orange-100 dark:bg-orange-900/40', text: 'text-orange-800 dark:text-orange-300', dot: 'bg-orange-500' },
  suspect: { bg: 'bg-red-100 dark:bg-red-900/40', text: 'text-red-800 dark:text-red-300', dot: 'bg-red-500' },
  banned: { bg: 'bg-red-200 dark:bg-red-900/60', text: 'text-red-900 dark:text-red-200', dot: 'bg-red-700' },
}

const StatusLabelKeys = {
  disconnected: 'disconnected',
  connecting: 'connecting_status',
  available: 'available',
  resting: 'resting',
  warming: 'warming',
  suspect: 'suspect',
  banned: 'banned',
}

const StrategyColors = {
  failover: { bg: 'bg-blue-100 dark:bg-blue-900/40', text: 'text-blue-800 dark:text-blue-300' },
  rotation: { bg: 'bg-purple-100 dark:bg-purple-900/40', text: 'text-purple-800 dark:text-purple-300' },
  hybrid: { bg: 'bg-orange-100 dark:bg-orange-900/40', text: 'text-orange-800 dark:text-orange-300' },
}

const BroadcastStatusLabelKeys = {
  pending: 'pending',
  processing: 'processing',
  paused: 'paused',
  completed: 'completed',
  cancelled: 'cancelled',
  failed: 'failed_status',
}

const BroadcastStatusColors = {
  pending: { bg: 'bg-neutral-200 dark:bg-neutral-700', text: 'text-neutral-700 dark:text-neutral-300' },
  processing: { bg: 'bg-blue-100 dark:bg-blue-900/40', text: 'text-blue-800 dark:text-blue-300' },
  paused: { bg: 'bg-yellow-100 dark:bg-yellow-900/40', text: 'text-yellow-800 dark:text-yellow-300' },
  completed: { bg: 'bg-emerald-100 dark:bg-emerald-900/40', text: 'text-emerald-800 dark:text-emerald-300' },
  cancelled: { bg: 'bg-neutral-300 dark:bg-neutral-700', text: 'text-neutral-700 dark:text-neutral-300' },
  failed: { bg: 'bg-red-100 dark:bg-red-900/40', text: 'text-red-800 dark:text-red-300' },
}

function formatPhone(phone) {
  if (!phone) return '-'
  const cleaned = phone.replace(/\D/g, '')
  if (cleaned.length === 13) {
    return `+${cleaned.slice(0, 2)} (${cleaned.slice(2, 4)}) ${cleaned.slice(4, 9)}-${cleaned.slice(9)}`
  }
  if (cleaned.length === 12) {
    return `+${cleaned.slice(0, 2)} (${cleaned.slice(2, 4)}) ${cleaned.slice(4, 8)}-${cleaned.slice(8)}`
  }
  return phone
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const locale = getLang() === 'en' ? 'en-US' : 'pt-BR'
  return date.toLocaleDateString(locale, { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function formatDateTime(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const locale = getLang() === 'en' ? 'en-US' : 'pt-BR'
  return date.toLocaleDateString(locale, {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function formatRelativeTime(dateStr) {
  if (!dateStr) return '-'
  const now = new Date()
  const date = new Date(dateStr)
  const diffMs = now - date
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)

  if (diffSec < 60) return t('now')
  if (diffMin < 60) return `${diffMin} ${t('min_ago')}`
  if (diffHour < 24) return `${diffHour} ${t('h_ago')}`
  if (diffDay < 7) return `${diffDay} ${t('d_ago')}`
  return formatDate(dateStr)
}

function formatNumber(num) {
  if (num === null || num === undefined) return '0'
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toString()
}

function formatPercent(value) {
  if (value === null || value === undefined) return '0%'
  return `${(value * 100).toFixed(1)}%`
}

function debounce(fn, delay) {
  let timer
  return function (...args) {
    clearTimeout(timer)
    timer = setTimeout(() => fn.apply(this, args), delay)
  }
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => {
    Alpine.store('toast').show('success', t('copied'))
  }).catch(() => {
    Alpine.store('toast').show('error', t('copy_failed'))
  })
}

function getStatusColor(status) {
  return StatusColors[status] || StatusColors.disconnected
}

function getStatusLabel(status) {
  const key = StatusLabelKeys[status]
  return key ? t(key) : status
}

function getStrategyLabel(strategy) {
  return t(strategy) || strategy
}

function getStrategyColor(strategy) {
  return StrategyColors[strategy] || StrategyColors.failover
}

function getBroadcastStatusLabel(status) {
  const key = BroadcastStatusLabelKeys[status]
  return key ? t(key) : status
}
