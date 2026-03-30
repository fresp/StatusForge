import { useCallback, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AlertCircle, AlertTriangle, CheckCircle, ChevronDown, ChevronRight, ChevronUp, Wrench, XCircle } from 'lucide-react'
import { useApi, useCategorySummary } from '../hooks/useApi'
import { STATUS_LABELS } from '../lib/utils'
import type { CategoryServiceStatus, ComponentStatus, Incident, MonitorMetrics, StatusPageSettings } from '../types'
import Footer from '../components/layout/Footer'
import { useWebSocket } from '../hooks/useWebSocket'
import { UptimeTimeline } from '../components/status/UptimeTimeline'
import { IncidentCarouselGroup } from '../components/IncidentCarouselGroup'

const EMPTY_INCIDENTS: Incident[] = []
const EMPTY_SERVICES: CategoryServiceStatus[] = []

function getStatusToken(status: ComponentStatus): string {
  switch (status) {
    case 'operational':
      return '--status-operational'
    case 'degraded_performance':
      return '--status-degraded'
    case 'partial_outage':
      return '--status-partial'
    case 'major_outage':
      return '--status-major'
    case 'maintenance':
      return '--status-maintenance'
    default:
      return '--status-operational'
  }
}

function StatusIcon({ status }: { status: ComponentStatus }) {
  const cls = 'w-5 h-5'
  const color = `var(${getStatusToken(status)})`

  switch (status) {
    case 'operational':
      return <CheckCircle className={cls} style={{ color }} />
    case 'degraded_performance':
      return <AlertTriangle className={cls} style={{ color }} />
    case 'partial_outage':
      return <AlertCircle className={cls} style={{ color }} />
    case 'major_outage':
      return <XCircle className={cls} style={{ color }} />
    case 'maintenance':
      return <Wrench className={cls} style={{ color }} />
    default:
      return <CheckCircle className={cls} style={{ color }} />
  }
}

function isIncidentActive(status: string): boolean {
  const normalized = status.toLowerCase()
  return normalized !== 'resolved' && normalized !== 'completed' && normalized !== 'closed' && normalized !== 'postmortem'
}

function impactRank(impact: string): number {
  switch (impact.toLowerCase()) {
    case 'critical':
      return 3
    case 'major':
      return 2
    case 'minor':
      return 1
    default:
      return 0
  }
}

function impactToStatus(impact: string): ComponentStatus {
  switch (impact.toLowerCase()) {
    case 'minor':
      return 'degraded_performance'
    case 'major':
      return 'partial_outage'
    case 'critical':
      return 'major_outage'
    default:
      return 'operational'
  }
}

function impactToLabel(impact: string): string {
  switch (impact.toLowerCase()) {
    case 'minor':
      return 'Degraded / Medium disruptions'
    case 'major':
      return 'Partial outage'
    case 'critical':
      return 'Major outage'
    default:
      return 'No known issues'
  }
}

function incidentAffectsService(incident: Incident, service: CategoryServiceStatus): boolean {
  const serviceName = service.name.trim().toLowerCase()

  if (incident.affectedComponentTargets && incident.affectedComponentTargets.length > 0) {
    return incident.affectedComponentTargets.some((target) => {
      const targetName = target.component.name.trim().toLowerCase()
      if (target.component.id === service.id || targetName === serviceName) {
        return true
      }

      if (target.subComponents && target.subComponents.length > 0) {
        return target.subComponents.some((subComponent) => {
          const subComponentName = subComponent.name.trim().toLowerCase()
          return subComponent.id === service.id || subComponentName === serviceName
        })
      }

      return false
    })
  }

  if (incident.affectedComponents.length > 0) {
    return incident.affectedComponents.some((component) => {
      const componentName = component.name.trim().toLowerCase()
      return component.id === service.id || componentName === serviceName
    })
  }

  return false
}

function PlatformStatus({ data, aggregateStatus }: { data: NonNullable<ReturnType<typeof useCategorySummary>['data']>; aggregateStatus: ComponentStatus }) {
  return (
    <header className="rounded-md border border-[var(--border)] bg-[var(--surface)] p-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{data.name}</h1>
          {data.description && <p className="text-sm mt-1 text-[var(--text-muted)]">{data.description}</p>}
        </div>
        <div className="flex items-center gap-2">
          <StatusIcon status={aggregateStatus} />
          <span className="text-sm font-semibold" style={{ color: `var(${getStatusToken(aggregateStatus)})` }}>
            {STATUS_LABELS[aggregateStatus]}
          </span>
        </div>
      </div>
    </header>
  )
}

function ServiceCard({ service, incidents }: { service: CategoryServiceStatus; incidents: Incident[] }) {
  const activeIncidents = incidents.filter((incident) => isIncidentActive(incident.status))
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())
  const [metricsExpanded, setMetricsExpanded] = useState(false)
  const highestImpact = activeIncidents.reduce<string>((current, incident) => {
    return impactRank(incident.impact) > impactRank(current) ? incident.impact : current
  }, '')
  const hasMonitoringData = service.uptimeHistory.length > 0
  const displayStatus = highestImpact ? impactToStatus(highestImpact) : service.status
  const displayLabel = STATUS_LABELS[displayStatus] ?? 'Unknown status'
  const {
    data: metrics,
    loading: metricsLoading,
  } = useApi<MonitorMetrics>(`/v1/monitors/${service.id}/metrics`, [service.id])

  const formatLatency = (value: number | null | undefined): string => {
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return '-'
    }

    return `${Math.round(value)} ms`
  }

  const formatAvailability = (value: number | null | undefined): string => {
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return '—'
    }

    return `${value.toFixed(1)}%`
  }

  const availabilityDotClass = (value: number | null | undefined): string => {
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return 'bg-[var(--text-subtle)]'
    }

    if (value >= 99) {
      return 'bg-emerald-500'
    }

    if (value >= 95) {
      return 'bg-amber-500'
    }

    return 'bg-red-500'
  }

  const historyRows = metrics?.history ?? []
  const hasMetricsData = historyRows.length > 0

  const p90 = metrics?.latency?.p90
  const p99 = metrics?.latency?.p99

  return (
    <article className='py-2 space-y-4'>
      <div className="flex items-center justify-between gap-4">
        <div>
          <h3 className="text-base font-semibold">{service.name}</h3>
          {hasMonitoringData && (
            <p className="text-xs text-[var(--text-muted)] mt-1">
              90-day uptime: {service.uptime90d.toFixed(2)}%
            </p>
          )}
        </div>
        <div className="text-right">
          <span
            className="text-xs font-medium rounded-full px-2.5 py-1"
            style={{
              backgroundColor: `color-mix(in oklab, var(${getStatusToken(displayStatus)}) 14%, transparent)`,
              color: `var(${getStatusToken(displayStatus)})`,
            }}
          >
            {displayLabel}
          </span>
        </div>
      </div>

      {hasMonitoringData && (
        <div className="py-2">
          <UptimeTimeline
            history={service.uptimeHistory}
            showAverage
            average={service.uptime90d}
          />
        </div>
      )}

      {hasMetricsData && (
        <section
          className="rounded-md border bg-[var(--surface)] p-5"
          style={{
            borderColor: 'var(--border)',
          }}
        >
          {metricsLoading ? (
            <div className="space-y-3 animate-pulse">
              <div className="h-4 w-20 rounded bg-[var(--surface-uptime)]" />
              <div className="h-8 w-80 max-w-full rounded bg-[var(--surface-uptime)]" />
            </div>
          ) : (
            <>
              <h4 className="text-sm font-semibold">Latency</h4>
              <p className="mt-2 text-2xl font-semibold leading-tight">
                {formatLatency(p90)}
                <span className="ml-2 text-sm font-medium text-[var(--text-muted)] align-middle">
                  (P90), {formatLatency(p99)} (P99) last 31 days
                </span>
              </p>

              <div className="mt-4 border-t border-[var(--border)] pt-3">
                <button
                  type="button"
                  className="inline-flex items-center gap-2 text-sm font-medium text-[var(--text)]"
                  onClick={() => setMetricsExpanded((current) => !current)}
                  aria-expanded={metricsExpanded}
                >
                  {metricsExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  Historical latency and availability
                </button>

                {metricsExpanded && (
                  <div className="mt-3 overflow-x-auto">
                    <table className="min-w-full text-sm">
                      <thead>
                        <tr className="border-b border-[var(--border)] text-left text-[var(--text-muted)]">
                          <th className="py-2 pr-4 font-medium">Month</th>
                          <th className="py-2 pr-4 font-medium">Latency (P90)</th>
                          <th className="py-2 pr-4 font-medium">Latency (P99)</th>
                          <th className="py-2 pr-2 font-medium">Availability</th>
                        </tr>
                      </thead>
                      <tbody>
                        {historyRows.length > 0 ? (
                          historyRows.map((entry) => {
                            const availability = entry.availability
                            const hasAvailability = typeof availability === 'number' && !Number.isNaN(availability)

                            return (
                              <tr key={entry.month} className="border-b border-[var(--border)] last:border-b-0">
                                <td className="py-2 pr-4 text-[var(--text)]">{entry.month}</td>
                                <td className="py-2 pr-4 text-[var(--text)]">{formatLatency(entry.latency?.p90)}</td>
                                <td className="py-2 pr-4 text-[var(--text)]">{formatLatency(entry.latency?.p99)}</td>
                                <td className="py-2 pr-2 text-[var(--text)]">
                                  {hasAvailability ? (
                                    <span className="inline-flex items-center gap-2">
                                      <span className={`inline-block h-2.5 w-2.5 rounded-full ${availabilityDotClass(availability)}`} />
                                      {formatAvailability(availability)}
                                    </span>
                                  ) : (
                                    '—'
                                  )}
                                </td>
                              </tr>
                            )
                          })
                        ) : (
                          <tr>
                            <td className="py-3 text-[var(--text-muted)]" colSpan={4}>
                              No historical metrics available.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </>
          )}
        </section>
      )}

      {activeIncidents.length > 0 ? (
        <div
          className="rounded-md border p-5"
          style={{
            borderColor: 'var(--border)',
            backgroundColor: 'var(--surface)',
          }}
        >
          <IncidentCarouselGroup
            title="Active incidents"
            incidents={activeIncidents}
            expandedIncidents={expandedIncidents}
            onToggleExpand={(incidentId) => {
              setExpandedIncidents((prev) => {
                const next = new Set(prev)
                if (next.has(incidentId)) {
                  next.delete(incidentId)
                } else {
                  next.add(incidentId)
                }
                return next
              })
            }}
          />
        </div>
      ) : (
        <div
          className="rounded-md border p-5"
          style={{
            borderColor: 'var(--border)',
            backgroundColor: 'var(--surface)',
          }}
        >
          <div className="flex items-start gap-3">
            {/* Icon */}
            <div className="mt-0.5 text-green-500">
              ●
            </div>

            <div>
              {/* Title */}
              <p className="text-sm font-medium">
                No known issues
              </p>

              {/* Timestamp */}
              <p className="text-xs text-[var(--text-muted)] mt-1">
                {new Date().toLocaleString()}
              </p>

              {/* Description */}
              <p className="text-sm text-[var(--text-muted)] mt-2">
                The service is up and running with no known issues.
              </p>
            </div>
          </div>
        </div>
      )}

    </article>
  )
}

export default function StatusCategoryPage() {
  const { categoryPrefix } = useParams<{ categoryPrefix: string }>()
  const { data, loading, error, refetch } = useCategorySummary(categoryPrefix)
  const { data: settingsData } = useApi<StatusPageSettings>('/status/settings')

  const incidents = data?.incidents ?? EMPTY_INCIDENTS
  const services = data?.services ?? EMPTY_SERVICES
  const aggregateStatus: ComponentStatus = data?.aggregateStatus ?? 'operational'
  const subComponentDividerStyle: React.CSSProperties = {
    borderColor: 'var(--subcomponent-divider)',
    ['--tw-divide-color' as any]: 'var(--subcomponent-divider)',
  }

  const incidentsByService = useMemo(() => {
    const serviceIncidentMap = new Map<string, Incident[]>()
    for (const service of services) {
      serviceIncidentMap.set(
        service.id,
        incidents.filter((incident) => incidentAffectsService(incident, service)),
      )
    }
    return serviceIncidentMap
  }, [incidents, services])

  const handleWsMessage = useCallback((event: { type: string; data: unknown }) => {
    if (['component_updated', 'component_created', 'incident_created', 'incident_updated', 'incident_resolved', 'incident_update_added'].includes(event.type)) {
      refetch()
    }
  }, [refetch])

  useWebSocket(handleWsMessage)

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--bg)] text-[var(--text)] flex flex-col">
        <main className="flex-1">
          <div className="max-w-5xl mx-auto px-4 py-10">Loading category status…</div>
        </main>
        <Footer centerText={settingsData?.footer?.text} showPoweredBy={settingsData?.footer?.showPoweredBy} />
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="min-h-screen bg-[var(--bg)] text-[var(--text)] flex flex-col">
        <main className="flex-1">
          <div className="max-w-5xl mx-auto px-4 py-10 space-y-3">
            <nav className="text-sm text-[var(--text-muted)] flex items-center gap-2">
              <Link to="/" className="hover:underline">Status</Link>
              <ChevronRight className="w-4 h-4" />
              <span>{categoryPrefix ?? 'Unknown category'}</span>
            </nav>
            <div className="rounded-md border border-[var(--border)] bg-[var(--surface)] p-6">
              <h1 className="text-xl font-semibold mb-2">Category unavailable</h1>
              <p className="text-sm text-[var(--text-muted)]">{error ?? 'Unable to load this category right now.'}</p>
            </div>
          </div>
        </main>
        <Footer centerText={settingsData?.footer?.text} showPoweredBy={settingsData?.footer?.showPoweredBy} />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[var(--bg)] text-[var(--text)] flex flex-col">
      <main className="flex-1">
        <div className="max-w-5xl mx-auto px-4 py-10 space-y-8">
          <nav className="text-sm text-[var(--text-muted)] flex items-center gap-2">
            <Link to="/" className="hover:underline">Status</Link>
            <ChevronRight className="w-4 h-4" />
            <span>{data.name}</span>
          </nav>

          <PlatformStatus data={data} aggregateStatus={aggregateStatus} />

          {services.length > 0 ? (
            <div className="divide-y divide-[color:var(--subcomponent-divider)] rounded-md border border-[var(--border)] bg-[var(--surface)]  p-5" aria-label="Services">
              {services.map((service) => (
                <ServiceCard
                  key={service.id}
                  service={service}
                  incidents={incidentsByService.get(service.id) ?? EMPTY_INCIDENTS}
                />
              ))}
            </div>
          ) : (
            <div className="rounded-md border border-[var(--border)] bg-[var(--surface)] p-6 text-sm text-[var(--text-muted)]">
              No services are configured for this category yet.
            </div>
          )}
        </div>
      </main>
      <Footer centerText={settingsData?.footer?.text} showPoweredBy={settingsData?.footer?.showPoweredBy} />
    </div>
  )
}
