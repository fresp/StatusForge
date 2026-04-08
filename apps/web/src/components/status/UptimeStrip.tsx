import type { ComponentStatus, UptimeBar } from '../../types'

type Props = {
  history: UptimeBar[]
}

const TARGET_BAR_COUNT = 40

const STATUS_SEVERITY: Record<ComponentStatus, number> = {
  operational: 0,
  degraded_performance: 1,
  maintenance: 2,
  partial_outage: 3,
  major_outage: 4,
}

function compressHistory(history: UptimeBar[], targetBarCount = TARGET_BAR_COUNT): UptimeBar[] {
  const recentHistory = history.slice(-90)
  if (recentHistory.length <= targetBarCount) {
    return recentHistory
  }

  const chunkSize = Math.ceil(recentHistory.length / targetBarCount)
  const compressed: UptimeBar[] = []

  for (let index = 0; index < recentHistory.length; index += chunkSize) {
    const chunk = recentHistory.slice(index, index + chunkSize)
    if (chunk.length === 0) {
      continue
    }

    const worstStatus = chunk.reduce<ComponentStatus>((currentWorst, entry) => {
      return STATUS_SEVERITY[entry.status] > STATUS_SEVERITY[currentWorst] ? entry.status : currentWorst
    }, 'operational')

    const averageUptime = chunk.reduce((total, entry) => total + entry.uptimePercent, 0) / chunk.length
    const lastEntry = chunk[chunk.length - 1]

    compressed.push({
      date: lastEntry.date,
      status: worstStatus,
      uptimePercent: averageUptime,
    })
  }

  return compressed.slice(-targetBarCount)
}

function getBarColor(status: ComponentStatus): string {
  switch (status) {
    case 'major_outage':
      return 'var(--status-major)'
    case 'partial_outage':
      return 'var(--status-partial)'
    case 'maintenance':
      return 'var(--status-maintenance)'
    case 'degraded_performance':
      return 'var(--status-degraded)'
    default:
      return 'var(--status-operational)'
  }
}

export default function UptimeStrip({ history }: Props) {
  const bars = compressHistory(history)
  const average = history.length > 0
    ? history.reduce((total, entry) => total + entry.uptimePercent, 0) / history.length
    : null

  return (
    <div className="space-y-3">
      <div className="flex items-end gap-1 overflow-hidden">
        {bars.map((bar) => (
          <span
            key={`${bar.date}-${bar.status}`}
            className="h-8 flex-1 rounded-full"
            style={{ backgroundColor: getBarColor(bar.status) }}
            title={`${bar.date} • ${bar.uptimePercent.toFixed(2)}% uptime`}
            aria-label={`${bar.date} ${bar.uptimePercent.toFixed(2)} percent uptime`}
          />
        ))}
      </div>
      <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-[0.24em]" style={{ color: 'var(--text-subtle)' }}>
        <span>90 days ago</span>
        <span>{average !== null ? `${average.toFixed(2)}% uptime` : 'Monitoring'}</span>
      </div>
    </div>
  )
}
