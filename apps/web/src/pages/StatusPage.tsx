import React, { useState, useCallback, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { CheckCircle, AlertTriangle, AlertCircle, XCircle, Wrench } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { useWebSocket } from '../hooks/useWebSocket'
import type { StatusSummary, ComponentWithSubs, Incident, Maintenance, StatusPageSettings } from '../types'
import { STATUS_LABELS, getOverallStatusLabel, formatDate, formatDateShort, groupIncidentsByRecentDays, groupIncidentsByStatus } from '../lib/utils'
import { IncidentCarouselGroup } from '../components/IncidentCarouselGroup'
import {
  DEFAULT_STATUS_PAGE_SETTINGS,
  applyStatusPageHeadSettings,
  applyStatusPageThemePreset,
  cacheStatusPageSettings,
  getBootstrappedStatusPageSettings,
  normalizeStatusPageSettings,
  parseStatusPageSettingsPayload,
  readCachedStatusPageSettings,
} from '../lib/statusPageSettings'

function getStatusToken(status: string): string {
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

function toCategoryPrefix(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

type ThemeVariableStyle = React.CSSProperties & Record<`--${string}`, string>

function StatusIcon({ status }: { status: string }) {
  const cls = 'w-5 h-5'
  const color = `var(${getStatusToken(status)})`
  switch (status) {
    case 'operational': return <CheckCircle className={cls} />
    case 'degraded_performance': return <AlertTriangle className={cls} />
    case 'partial_outage': return <AlertCircle className={cls} />
    case 'major_outage': return <XCircle className={cls} />
    case 'maintenance': return <Wrench className={cls} />
    default: return <CheckCircle className={cls} style={{ color: 'var(--status-operational)' }} />
  }
}

function UptimeBar({ bars }: { bars: { date: string; uptimePercent: number; status: string }[] }) {
  return (
    <div className="flex gap-px items-end h-8 mt-2">
      {bars.map((bar, i) => (
        <div
          key={i}
          className="flex-1 rounded-sm opacity-80 hover:opacity-100 transition-opacity cursor-pointer"
          style={{
            backgroundColor: `var(${getStatusToken(bar.status)})`,
            height: `${Math.max(20, bar.uptimePercent / 100 * 32)}px`,
          }}
          title={`${bar.date}: ${bar.uptimePercent.toFixed(2)}% uptime`}
        />
      ))}
    </div>
  )
}

export default function StatusPage() {
  const navigate = useNavigate()
  const { data: summary, refetch: refetchSummary } = useApi<StatusSummary>('/status/summary')
  const { data: components, refetch: refetchComponents } = useApi<ComponentWithSubs[]>('/status/components')
  const { data: incidentData, refetch: refetchIncidents } = useApi<{ active: Incident[]; resolved: Incident[] }>('/status/incidents')
  const { data: settingsData } = useApi<StatusPageSettings>('/status/settings')

  const { data: maintenanceData } = useApi<Maintenance[]>('/status/maintenance')

  const [settings, setSettings] = useState<StatusPageSettings>(() => (
    getBootstrappedStatusPageSettings()
    ?? readCachedStatusPageSettings()
    ?? DEFAULT_STATUS_PAGE_SETTINGS
  ))

  const handleWsMessage = useCallback((event: { type: string; data: unknown }) => {
    if (['component_updated', 'component_created'].includes(event.type)) {
      refetchComponents()
      refetchSummary()
    }
    if (['incident_created', 'incident_updated', 'incident_resolved', 'incident_update_added'].includes(event.type)) {
      refetchIncidents()
      refetchSummary()
    }
    if (event.type === 'status_page_settings_updated') {
      const nextSettings = parseStatusPageSettingsPayload(event.data)
      if (!nextSettings) {
        return
      }

      setSettings(nextSettings)
      cacheStatusPageSettings(nextSettings)
    }
  }, [refetchComponents, refetchSummary, refetchIncidents])

  useWebSocket(handleWsMessage)

  const overallStatus = summary?.overallStatus || 'operational'
  const activeIncidents = incidentData?.active || []
  const resolvedIncidents = incidentData?.resolved || []
  const upcomingMaintenance = maintenanceData?.filter(m => m.status !== 'completed') || []
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!settingsData) {
      return
    }

    const nextSettings = normalizeStatusPageSettings(settingsData)
    setSettings(nextSettings)
    cacheStatusPageSettings(nextSettings)
  }, [settingsData])

  const recentIncidents = [...activeIncidents, ...resolvedIncidents].filter((incident: Incident) => {
    const sevenDaysAgo = new Date()
    sevenDaysAgo.setHours(0, 0, 0, 0)
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6)
    return new Date(incident.createdAt) >= sevenDaysAgo
  })
  const recentIncidentGroups = groupIncidentsByRecentDays(recentIncidents, 7)

  useEffect(() => {
    applyStatusPageHeadSettings(settings)
  }, [settings])

  useEffect(() => {
    applyStatusPageThemePreset(settings)
  }, [settings])

  const headerStatusToken = getStatusToken(overallStatus)

  const pageStyle: ThemeVariableStyle = {
    backgroundColor: 'var(--bg)',
    color: 'var(--text)',
    fontFamily: 'var(--font-family)',
    backgroundImage: settings.branding.backgroundImageUrl
      ? `linear-gradient(var(--bg-image-overlay), var(--bg-image-overlay)), url(${settings.branding.backgroundImageUrl})`
      : undefined,
    backgroundSize: settings.branding.backgroundImageUrl ? 'cover' : undefined,
    backgroundAttachment: settings.branding.backgroundImageUrl ? 'fixed' : undefined,
    backgroundPosition: settings.branding.backgroundImageUrl ? 'center' : undefined,
  }

  const headerStyle: React.CSSProperties = {
    backgroundColor: `var(${headerStatusToken})`,
    color: 'var(--on-primary)',
    boxShadow: 'inset 0 -3px 0 var(--color-accent)',
    borderRadius: '0px 0px 8px 8px'
  }

  const contentClassName = 'max-w-4xl mx-auto px-4 py-8 space-y-8'

  const cardSurfaceStyle: React.CSSProperties = {
    backgroundColor: 'var(--surface)',
    color: 'var(--text)',
    borderColor: 'var(--border)',
  }

  const sectionTitleStyle: React.CSSProperties = {
    color: 'var(--text)',
  }

  const mutedTextColor = 'var(--text-muted)'
  const subtleTextColor = 'var(--text-subtle)'
  const incidentSurfaceStyle: React.CSSProperties = {
    backgroundColor: 'var(--surface-incident)',
    borderColor: 'var(--border-incident)',
    color: 'var(--text)',
  }
  const maintenanceSurfaceStyle: React.CSSProperties = {
    backgroundColor: 'var(--surface-maintenance)',
    borderColor: 'var(--border-maintenance)',
    color: 'var(--text)',
  }
  const uptimeSurfaceStyle: React.CSSProperties = {
    backgroundColor: 'var(--surface-uptime)',
    borderColor: 'var(--border)',
  }
  const heroImageStyle: React.CSSProperties = {
    borderColor: 'var(--hero-image-border)',
  }
  const componentHeaderStyle: React.CSSProperties = {
    borderColor: 'var(--color-accent)',
  }
  const subComponentDividerStyle: React.CSSProperties = {
    borderColor: 'var(--subcomponent-divider)',
  }


  return (
    <div className="min-h-screen" style={pageStyle}>
      {/* Header */}
      <div
        className="max-w-4xl mx-auto px-4 py-8 space-y-8"
        style={headerStyle}
      >
        <div className="max-w-4xl mx-auto">
          <div className="flex items-center gap-3 mb-2">
            {settings.branding.logoUrl && (
              <img
                src={settings.branding.logoUrl}
                alt={`${settings.branding.siteName} logo`}
                className="w-10 h-10 object-contain rounded"
              />
            )}
            <h1 className="text-3xl font-bold">{settings.branding.siteName}</h1>
          </div>
          {settings.branding.heroImageUrl && (
            <img
              src={settings.branding.heroImageUrl}
              alt="Status page hero"
              className="w-full max-h-48 object-cover rounded-xl border mb-4"
              style={heroImageStyle}
            />
          )}
          <div className="flex items-center gap-3 text-xl">
            <StatusIcon status={overallStatus} />
            <span>{getOverallStatusLabel(overallStatus as any)}</span>
          </div>
          {activeIncidents.length > 0 && (
            <p className="mt-2 text-sm" style={{ color: 'var(--on-primary-subtle)' }}>{activeIncidents.length} active incident{activeIncidents.length > 1 ? 's' : ''}</p>
          )}
        </div>
      </div>

      <div className={contentClassName}>

        {activeIncidents.length > 0 && (
          <section className="space-y-4">

            <div className="space-y-4">
              {groupIncidentsByStatus(activeIncidents).map((group) => (
                <IncidentCarouselGroup
                  key={`active-${group.key}`}
                  title={group.label}
                  subtitle="Swipe or use arrows to browse active incidents without leaving the page context."
                  incidents={group.incidents}
                  expandedIncidents={expandedIncidents}
                  onToggleExpand={(incidentId) => setExpandedIncidents((prev) => {
                    const next = new Set(prev)
                    if (next.has(incidentId)) {
                      next.delete(incidentId)
                    } else {
                      next.add(incidentId)
                    }
                    return next
                  })}
                />
              ))}
            </div>
          </section>
        )}

        {/* Upcoming Maintenance */}
        {upcomingMaintenance.map(m => (
          <div key={m.id} className="border rounded-lg p-4" style={maintenanceSurfaceStyle}>
            <div className="flex items-start gap-3">
              <Wrench className="w-5 h-5 mt-0.5 flex-shrink-0" style={{ color: 'var(--status-maintenance)' }} />
              <div>
                <h3 className="font-semibold">{m.title}</h3>
                <p className="text-sm mt-1" style={{ color: mutedTextColor }}>{m.description}</p>
                <div className="flex gap-4 mt-2 text-xs" style={{ color: subtleTextColor }}>
                  <span>Status: {m.status.replace('_', ' ')}</span>
                  <span>{formatDate(m.startTime)} → {formatDate(m.endTime)}</span>
                  {m.creatorUsername && <span>Created by: {m.creatorUsername}</span>}
                </div>
              </div>
            </div>
          </div>
        ))}

        {/* Components */}
        {(components || []).map(comp => (
          <div
            key={comp.id}
            className="rounded-xl shadow-sm border overflow-hidden cursor-pointer transition-colors hover:bg-[var(--surface-uptime)]"
            style={cardSurfaceStyle}
            role="button"
            tabIndex={0}
            onClick={() => navigate(`/status/${toCategoryPrefix(comp.name)}`)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                navigate(`/status/${toCategoryPrefix(comp.name)}`)
              }
            }}
          >
            <div className="flex items-center justify-between px-6 py-4 border-b" style={componentHeaderStyle}>
              <div>
                <h2 className="text-lg font-semibold" style={sectionTitleStyle}>{comp.name}</h2>
                {comp.description && <p className="text-sm mt-0.5" style={{ color: subtleTextColor }}>{comp.description}</p>}
              </div>
              <div className="flex items-center gap-2">
                <StatusIcon status={comp.status} />
                <span className="text-sm font-medium" style={{ color: `var(${getStatusToken(comp.status)})` }}>
                  {STATUS_LABELS[comp.status]}
                </span>
              </div>
            </div>

            {/* SubComponents */}
            {comp.subComponents && comp.subComponents.length > 0 && (
              <div className="divide-y" style={subComponentDividerStyle}>
                {comp.subComponents.map(sub => (
                  <div key={sub.id} className="flex items-center justify-between px-6 py-3">
                    <span className="text-sm pl-4" style={{ color: mutedTextColor }}>{sub.name}</span>
                    <div className="flex items-center gap-2">
                      <StatusIcon status={sub.status} />
                      <span className="text-xs font-medium" style={{ color: `var(${getStatusToken(sub.status)})` }}>
                        {STATUS_LABELS[sub.status]}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* 90-day uptime */}
            {comp.uptimeHistory && comp.uptimeHistory.length > 0 && (
              <div className="px-6 py-4 border-t" style={uptimeSurfaceStyle}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs" style={{ color: subtleTextColor }}>90-day uptime</span>
                  <span className="text-xs" style={{ color: subtleTextColor }}>
                    {comp.uptimeHistory.length > 0
                      ? `${(comp.uptimeHistory.reduce((s, b) => s + b.uptimePercent, 0) / comp.uptimeHistory.length).toFixed(2)}% avg`
                      : ''}
                  </span>
                </div>
                <UptimeBar bars={comp.uptimeHistory} />
                <div className="flex justify-between mt-1">
                  <span className="text-xs" style={{ color: subtleTextColor }}>{formatDateShort(comp.uptimeHistory[0]?.date)}</span>
                  <span className="text-xs" style={{ color: subtleTextColor }}>Today</span>
                </div>
              </div>
            )}
          </div>
        ))}

        {/* Recent Incident History (last 7 days) */}
        {recentIncidentGroups.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold mb-4" style={sectionTitleStyle}>Recent Incident History</h2>
            <div className="space-y-6">
              {recentIncidentGroups.map((group) => {
                const dayLabel = new Date(`${group.dateKey}T00:00:00`).toLocaleDateString('en-US', {
                  weekday: 'long',
                  month: 'long',
                  day: 'numeric',
                })

                return (
                  <div key={group.dateKey}>
                    <h3 className="text-lg font-semibold mb-3" style={{ color: mutedTextColor }}>{dayLabel}</h3>
                    {group.incidents.length === 0 ? (
                      <div className="rounded-xl border p-5" style={cardSurfaceStyle}>
                        <p className="text-sm" style={{ color: mutedTextColor }}>No incidents reported.</p>
                      </div>
                    ) : (
                      <div className="space-y-4">
                        {groupIncidentsByStatus(group.incidents).map((statusGroup) => (
                          <IncidentCarouselGroup
                            key={`${group.dateKey}-${statusGroup.key}`}
                            title={statusGroup.label}
                            subtitle={dayLabel}
                            incidents={statusGroup.incidents}
                            expandedIncidents={expandedIncidents}
                            onToggleExpand={(incidentId) => setExpandedIncidents((prev) => {
                              const next = new Set(prev)
                              if (next.has(incidentId)) {
                                next.delete(incidentId)
                              } else {
                                next.add(incidentId)
                              }
                              return next
                            })}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>

          </div>
        )}

        <div className="py-2 border-t" style={{ borderColor: 'var(--border)' }}>
          <div className="flex items-center justify-between">

            {/* LEFT BLOCK */}
            <div className="flex flex-col gap-2">
              <div className="text-sm" style={{ color: mutedTextColor }}>
                {settings.footer.text}
              </div>

              {settings.footer.showPoweredBy && (
                <div
                  className="text-xs opacity-40"
                  style={{ color: subtleTextColor }}
                >
                  Powered by{" "}
                  <a href="https://github.com/fresp/StatusForge">
                    StatusForge
                  </a>
                </div>
              )}
            </div>

            {/* RIGHT CTA */}
            <div>
              {recentIncidentGroups.length > 0 && (
                <Link
                  to="/history"
                  className="inline-flex items-center rounded-lg px-4 py-2 text-sm font-medium border transition-colors"
                  style={{
                    borderColor: 'var(--border)',
                    color: 'var(--text)',
                    backgroundColor: 'var(--surface)',
                  }}
                >
                  View History
                </Link>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
