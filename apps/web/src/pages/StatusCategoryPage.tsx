import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { AlertCircle, AlertTriangle, CheckCircle, ChevronDown, ChevronRight, Wrench, XCircle } from 'lucide-react'
import { useCategorySummary } from '../hooks/useApi'
import { STATUS_LABELS, formatDate } from '../lib/utils'
import type { CategoryServiceStatus, ComponentStatus, Incident } from '../types'

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

function IncidentSummary({ incident }: { incident: Incident }) {
  const badgeStatus = impactToStatus(incident.impact)
  return (
    <article className="rounded-lg border border-[var(--border)] bg-[var(--bg)] p-4">
      <div className="flex items-start justify-between gap-4">
        <h4 className="text-sm font-semibold leading-tight">{incident.title}</h4>
        <span
          className="text-[11px] font-medium rounded-full px-2 py-1 whitespace-nowrap"
          style={{
            backgroundColor: `color-mix(in oklab, var(${getStatusToken(badgeStatus)}) 14%, transparent)`,
            color: `var(${getStatusToken(badgeStatus)})`,
          }}
        >
          {impactToLabel(incident.impact)}
        </span>
      </div>
      <p className="text-sm text-[var(--text-muted)] mt-2">{incident.description}</p>
      <p className="text-xs text-[var(--text-subtle)] mt-2">Started: {formatDate(incident.createdAt)}</p>
    </article>
  )
}

function ServiceRow({
  service,
  incidents,
  expanded,
  onToggle,
}: {
  service: CategoryServiceStatus
  incidents: Incident[]
  expanded: boolean
  onToggle: () => void
}) {
  const highestImpact = incidents.reduce<string>((current, incident) => {
    return impactRank(incident.impact) > impactRank(current) ? incident.impact : current
  }, '')

  const displayStatus = highestImpact ? impactToStatus(highestImpact) : service.status
  const displayLabel = highestImpact ? impactToLabel(highestImpact) : 'No known issues'

  const incidentsByStatus = useMemo(() => {
    const grouped = new Map<string, Incident[]>();
    ['critical', 'major', 'minor', 'none'].forEach(status => grouped.set(status, []));
    
    incidents.forEach(incident => {
      const statusKey = incident.impact.toLowerCase();
      const current = grouped.get(statusKey) || [];
      current.push(incident);
      grouped.set(statusKey, current);
    });
    
    const unknownImpactIncidents = incidents.filter(i => 
      !['critical', 'major', 'minor'].includes(i.impact.toLowerCase())
    );
    if (unknownImpactIncidents.length > 0) {
      const current = grouped.get('none') || [];
      current.push(...unknownImpactIncidents);
      grouped.set('none', current);
    }
    
    return grouped;
  }, [incidents]);

  return (
    <article className="rounded-xl border border-[var(--border)] bg-[var(--surface)] overflow-hidden">
      <button
        type="button"
        onClick={onToggle}
        className="w-full px-5 py-4 flex items-center justify-between gap-4 text-left hover:bg-[var(--surface-elevated)] transition-colors"
        aria-expanded={expanded}
      >
        <div>
          <h3 className="text-base font-semibold">{service.name}</h3>
          <p className="text-xs text-[var(--text-muted)] mt-1">90-day uptime: {service.uptime90d.toFixed(2)}%</p>
        </div>
        <div className="flex items-center gap-3">
          <span
            className="text-xs font-medium rounded-full px-2.5 py-1"
            style={{
              backgroundColor: `color-mix(in oklab, var(${getStatusToken(displayStatus)}) 14%, transparent)`,
              color: `var(${getStatusToken(displayStatus)})`,
            }}
          >
            {displayLabel}
          </span>
          <ChevronDown
            className={`w-4 h-4 text-[var(--text-muted)] transition-transform ${expanded ? 'rotate-180' : ''}`}
          />
        </div>
      </button>

      {expanded && (
        <div className="border-t border-[var(--border)] p-0 space-y-4">
          {incidents.length === 0 ? (
            <div className="px-5 py-4">
              <p className="text-sm text-[var(--text-muted)]">No known issues.</p>
            </div>
          ) : (
            <>
              {Array.from(incidentsByStatus.entries()).map(([status, statusIncidents]) => {
                if (statusIncidents.length === 0) return null;
                
                let statusLabel = '';
                switch(status) {
                  case 'critical':
                    statusLabel = 'Critical Incidents';
                    break;
                  case 'major':
                    statusLabel = 'Major Incidents';
                    break;
                  case 'minor':
                    statusLabel = 'Minor Incidents';
                    break;
                  default:
                    statusLabel = 'Other Incidents';
                }
                
                return (
                  <div key={status} className="px-5 py-4">
                    <h4 className="text-sm font-semibold mb-3 flex items-center gap-2">
                      <span 
                        className="inline-block w-2 h-2 rounded-full"
                        style={{ 
                          backgroundColor: `var(${getStatusToken(impactToStatus(status))})` 
                        }}
                      />
                      {statusLabel} ({statusIncidents.length})
                    </h4>
                    <div className="space-y-3">
                      {statusIncidents.map((incident) => (
                        <div key={incident.id} className="pl-4 border-l-2" style={{ borderColor: `var(${getStatusToken(impactToStatus(incident.impact))})` }}>
                          <IncidentSummary incident={incident} />
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}
            </>
          )}
        </div>
      )}
    </article>
  )
}

export default function StatusCategoryPage() {
  const { categoryPrefix } = useParams<{ categoryPrefix: string }>()
  const { data, loading, error } = useCategorySummary(categoryPrefix)

  const incidents = data?.incidents ?? EMPTY_INCIDENTS
  const services = data?.services ?? EMPTY_SERVICES
  const aggregateStatus: ComponentStatus = data?.aggregateStatus ?? 'operational'

  const activeIncidents = useMemo(
    () => incidents.filter((incident) => isIncidentActive(incident.status)),
    [incidents],
  )

  const incidentsByService = useMemo(() => {
    const serviceIncidentMap = new Map<string, Incident[]>()
    for (const service of services) {
      serviceIncidentMap.set(
        service.id,
        activeIncidents.filter((incident) => incidentAffectsService(incident, service)),
      )
    }
    return serviceIncidentMap
  }, [activeIncidents, services])

  const defaultExpandedIds = useMemo(
    () => services.filter((service) => (incidentsByService.get(service.id)?.length ?? 0) > 0).map((service) => service.id),
    [services, incidentsByService],
  )

  const [expandedServiceIds, setExpandedServiceIds] = useState<Set<string>>(new Set())

  useEffect(() => {
    setExpandedServiceIds((current) => {
      if (current.size === defaultExpandedIds.length && defaultExpandedIds.every((id) => current.has(id))) {
        return current
      }
      return new Set(defaultExpandedIds)
    })
  }, [defaultExpandedIds])

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--bg)] text-[var(--text)]">
        <div className="max-w-5xl mx-auto px-4 py-10">Loading category status…</div>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="min-h-screen bg-[var(--bg)] text-[var(--text)]">
        <div className="max-w-5xl mx-auto px-4 py-10 space-y-3">
          <nav className="text-sm text-[var(--text-muted)] flex items-center gap-2">
            <Link to="/" className="hover:underline">Status</Link>
            <ChevronRight className="w-4 h-4" />
            <span>{categoryPrefix ?? 'Unknown category'}</span>
          </nav>
          <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6">
            <h1 className="text-xl font-semibold mb-2">Category unavailable</h1>
            <p className="text-sm text-[var(--text-muted)]">{error ?? 'Unable to load this category right now.'}</p>
          </div>
        </div>
      </div>
    )
  }

  const toggleService = (serviceId: string) => {
    setExpandedServiceIds((current) => {
      const next = new Set(current)
      if (next.has(serviceId)) {
        next.delete(serviceId)
      } else {
        next.add(serviceId)
      }
      return next
    })
  }

  return (
    <div className="min-h-screen bg-[var(--bg)] text-[var(--text)]">
      <div className="max-w-5xl mx-auto px-4 py-10 space-y-8">
        <nav className="text-sm text-[var(--text-muted)] flex items-center gap-2">
          <Link to="/" className="hover:underline">Status</Link>
          <ChevronRight className="w-4 h-4" />
          <span>{data.name}</span>
        </nav>

        <header className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6">
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
          <p className="mt-4 text-xs text-[var(--text-subtle)]">Uptime (90d): {data.uptime90d.toFixed(2)}%</p>
        </header>

        <section className="space-y-4">
          {services.length === 0 ? (
            <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-6 text-sm text-[var(--text-muted)]">
              No sub-components are mapped to this category yet.
            </div>
          ) : (
            <div className="space-y-4">
              {services.map((service) => (
                <ServiceRow
                  key={service.id}
                  service={service}
                  incidents={incidentsByService.get(service.id) ?? []}
                  expanded={expandedServiceIds.has(service.id)}
                  onToggle={() => toggleService(service.id)}
                />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
