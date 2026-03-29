import type { ComponentStatus } from '../../types'

type UptimeDay = {
  date: string
  status: string
  uptimePercent: number
}

type Props = {
  history: UptimeDay[]
  showLabels?: boolean
  showAverage?: boolean
  average?: number
}

export function UptimeTimeline({
  history,
  showLabels = true,
  showAverage = false,
  average,
}: Props) {
  const last90 = history.slice(-90)

  return (
    <div className="w-full space-y-2">
      <div className="flex w-full gap-[2px] items-end">
        {last90.map((day, i) => {
          let color = 'var(--status-operational)'

          if (day.status === 'major_outage') {
            color = 'var(--status-major)'
          } else if (day.status === 'partial_outage') {
            color = 'var(--status-partial)'
          } else if (day.status === 'degraded_performance') {
            color = 'var(--status-degraded)'
          }

          return (
            <div
              key={i}
              className={`h-8 flex-1 min-w-[2px] rounded-[2px] ${i === last90.length - 1 ? 'ring-1 ring-white/20' : ''}`}
              style={{ backgroundColor: color }}
              title={`${day.date} - ${day.status as ComponentStatus}`}
            />
          )
        })}
      </div>

      {(showLabels || (showAverage && average !== undefined)) && (
        <div className="flex items-end justify-between text-xs">
          <span className="text-[var(--text-subtle)]">{showLabels ? (last90[0]?.date ?? '') : ''}</span>
          <div className="flex flex-col items-end leading-tight">
            {showLabels && <span className="text-[var(--text-subtle)]">Today</span>}
            {showAverage && average !== undefined && (
              <span className="text-[var(--text-muted)]">{average.toFixed(2)}% avg</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
