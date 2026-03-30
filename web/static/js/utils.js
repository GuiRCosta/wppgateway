// WPP Gateway - Utility Functions

const StatusColors = {
  disconnected: { bg: 'bg-neutral-300', text: 'text-neutral-700', dot: 'bg-neutral-500' },
  connecting: { bg: 'bg-yellow-100', text: 'text-yellow-800', dot: 'bg-yellow-500' },
  available: { bg: 'bg-emerald-100', text: 'text-emerald-800', dot: 'bg-emerald-500' },
  resting: { bg: 'bg-blue-100', text: 'text-blue-800', dot: 'bg-blue-500' },
  warming: { bg: 'bg-orange-100', text: 'text-orange-800', dot: 'bg-orange-500' },
  suspect: { bg: 'bg-red-100', text: 'text-red-800', dot: 'bg-red-500' },
  banned: { bg: 'bg-red-200', text: 'text-red-900', dot: 'bg-red-700' },
}

const StatusLabels = {
  disconnected: 'Desconectado',
  connecting: 'Conectando',
  available: 'Disponivel',
  resting: 'Descansando',
  warming: 'Aquecendo',
  suspect: 'Suspeito',
  banned: 'Banido',
}

const StrategyLabels = {
  failover: 'Failover',
  rotation: 'Rotacao',
  hybrid: 'Hibrido',
}

const StrategyColors = {
  failover: { bg: 'bg-blue-100', text: 'text-blue-800' },
  rotation: { bg: 'bg-purple-100', text: 'text-purple-800' },
  hybrid: { bg: 'bg-orange-100', text: 'text-orange-800' },
}

const BroadcastStatusLabels = {
  pending: 'Pendente',
  processing: 'Processando',
  paused: 'Pausado',
  completed: 'Concluido',
  cancelled: 'Cancelado',
  failed: 'Falhou',
}

const BroadcastStatusColors = {
  pending: { bg: 'bg-neutral-200', text: 'text-neutral-700' },
  processing: { bg: 'bg-blue-100', text: 'text-blue-800' },
  paused: { bg: 'bg-yellow-100', text: 'text-yellow-800' },
  completed: { bg: 'bg-emerald-100', text: 'text-emerald-800' },
  cancelled: { bg: 'bg-neutral-300', text: 'text-neutral-700' },
  failed: { bg: 'bg-red-100', text: 'text-red-800' },
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
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function formatDateTime(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('pt-BR', {
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

  if (diffSec < 60) return 'agora'
  if (diffMin < 60) return `${diffMin}min atras`
  if (diffHour < 24) return `${diffHour}h atras`
  if (diffDay < 7) return `${diffDay}d atras`
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
    Alpine.store('toast').show('success', 'Copiado para a area de transferencia')
  }).catch(() => {
    Alpine.store('toast').show('error', 'Falha ao copiar')
  })
}

function getStatusColor(status) {
  return StatusColors[status] || StatusColors.disconnected
}

function getStatusLabel(status) {
  return StatusLabels[status] || status
}

function getStrategyLabel(strategy) {
  return StrategyLabels[strategy] || strategy
}

function getStrategyColor(strategy) {
  return StrategyColors[strategy] || StrategyColors.failover
}
