import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Sparkles, Wrench } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { useWebSocket } from '../hooks/useWebSocket'
import type { ComponentStatus, ComponentWithSubs, Incident, Maintenance, StatusSummary, StatusPageSettings } from '../types'
import { getOverallStatusLabel, groupIncidentsByStatus, formatDate } from '../lib/utils'
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
import Footer from '../components/layout/Footer'
import StatusTopNav from '../components/status/StatusTopNav'
import StatusHero from '../components/status/StatusHero'
import StatusServiceGrid from '../components/status/StatusServiceGrid'

function toCategoryPrefix(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

type ThemeVariableStyle = React.CSSProperties & Record<`--${string}`, string>

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
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

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

  useEffect(() => {
    if (!settingsData) {
      return
    }

    const nextSettings = normalizeStatusPageSettings(settingsData)
    setSettings(nextSettings)
    cacheStatusPageSettings(nextSettings)
  }, [settingsData])

  useEffect(() => {
    applyStatusPageHeadSettings(settings)
  }, [settings])

  useEffect(() => {
    applyStatusPageThemePreset(settings)
  }, [settings])

  const overallStatus: ComponentStatus = summary?.overallStatus ?? 'operational'
  const activeIncidents = incidentData?.active ?? []
  const resolvedIncidents = incidentData?.resolved ?? []
  const upcomingMaintenance = maintenanceData?.filter((maintenance) => maintenance.status !== 'completed') ?? []

  const recentIncidents = useMemo(() => {
    return [...activeIncidents, ...resolvedIncidents].filter((incident) => {
      const sevenDaysAgo = new Date()
      sevenDaysAgo.setHours(0, 0, 0, 0)
      sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 6)
      return new Date(incident.createdAt) >= sevenDaysAgo
    })
  }, [activeIncidents, resolvedIncidents])

  const displayedIncidentGroups = useMemo(() => {
    const source = activeIncidents.length > 0 ? activeIncidents : recentIncidents
    return groupIncidentsByStatus(source)
  }, [activeIncidents, recentIncidents])

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

  const sectionCardStyle: React.CSSProperties = {
    backgroundColor: 'var(--status-card-bg, rgba(255,255,255,0.88))',
    borderColor: 'var(--status-card-border, rgba(181,197,226,0.32))',
    boxShadow: 'var(--status-card-shadow, 0 24px 60px rgba(20,37,63,0.08))',
  }

  const incidentSectionSubtitle = activeIncidents.length > 0
    ? 'Swipe through the active incident feed without losing page context.'
    : 'Recent incident activity from the last seven days.'

  return (
    <div className="min-h-screen" style={pageStyle}>
      <StatusTopNav
        siteName={settings.branding.siteName}
        logoUrl={settings.branding.logoUrl}
        statusLabel={overallStatus === 'operational' ? 'stable' : 'attention'}
        activeView="dashboard"
      />

      <main className="pb-16">
        <StatusHero
          status={overallStatus}
          title={getOverallStatusLabel(overallStatus)}
          description={settings.head.description || 'We are continuously monitoring all services and publishing live operational status updates.'}
          siteName={settings.branding.siteName}
          activeIncidents={summary?.activeIncidents ?? activeIncidents.length}
          scheduledMaintenance={summary?.scheduledMaintenance ?? upcomingMaintenance.length}
        />

        <section className="px-4 pt-8 md:px-6 md:pt-10">
          <div className="mx-auto flex max-w-6xl items-end justify-between gap-6">
            <div>
              <p className="text-[11px] font-extrabold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
                Service overview
              </p>
              <h2 className="mt-3 text-3xl font-black tracking-[-0.04em] md:text-4xl" style={{ color: 'var(--text)' }}>
                System health at a glance
              </h2>
            </div>
            <div className="hidden items-center gap-2 rounded-full px-4 py-2 md:inline-flex" style={{ backgroundColor: 'var(--surface)' }}>
              <Sparkles className="h-4 w-4" style={{ color: 'var(--primary)' }} />
              <span className="text-xs font-bold uppercase tracking-[0.24em]" style={{ color: 'var(--text-muted)' }}>
                Live monitor stream
              </span>
            </div>
          </div>
        </section>

        <StatusServiceGrid
          components={components ?? []}
          onSelectComponent={(component) => navigate(`/status/${toCategoryPrefix(component.name)}`)}
        />

        {upcomingMaintenance.length > 0 && (
          <section className="px-4 pt-8 md:px-6 md:pt-10">
            <div className="mx-auto max-w-6xl">
              <div className="mb-5">
                <p className="text-[11px] font-extrabold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
                  Planned work
                </p>
                <h2 className="mt-3 text-2xl font-black tracking-[-0.03em] md:text-3xl" style={{ color: 'var(--text)' }}>
                  Scheduled maintenance
                </h2>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                {upcomingMaintenance.map((maintenance) => (
                  <div key={maintenance.id} className="rounded-[1.6rem] border p-6" style={sectionCardStyle}>
                    <div className="flex items-start gap-4">
                      <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl" style={{ backgroundColor: 'var(--surface-maintenance)' }}>
                        <Wrench className="h-5 w-5" style={{ color: 'var(--status-maintenance)' }} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <h3 className="text-lg font-black tracking-[-0.02em]" style={{ color: 'var(--text)' }}>
                          {maintenance.title}
                        </h3>
                        <p className="mt-2 text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
                          {maintenance.description}
                        </p>
                        <div className="mt-4 flex flex-wrap gap-x-4 gap-y-2 text-xs font-semibold uppercase tracking-[0.18em]" style={{ color: 'var(--text-subtle)' }}>
                          <span>Status {maintenance.status.replace('_', ' ')}</span>
                          <span>{formatDate(maintenance.startTime)} → {formatDate(maintenance.endTime)}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </section>
        )}

        {displayedIncidentGroups.length > 0 && (
          <section className="px-4 pt-8 md:px-6 md:pt-10">
            <div className="mx-auto max-w-6xl rounded-[1.8rem] border p-6 md:p-8" style={sectionCardStyle}>
              <div className="mb-5 md:mb-6">
                <p className="text-[11px] font-extrabold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
                  Incident room
                </p>
                <h2 className="mt-3 text-2xl font-black tracking-[-0.03em] md:text-3xl" style={{ color: 'var(--text)' }}>
                  {activeIncidents.length > 0 ? 'Active incident feed' : 'Recent incident timeline'}
                </h2>
                <p className="mt-2 text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
                  {incidentSectionSubtitle}
                </p>
              </div>

              {displayedIncidentGroups.map((group) => (
                <IncidentCarouselGroup
                  key={`incident-${group.key}`}
                  title={group.label}
                  subtitle={group.key === 'active' ? incidentSectionSubtitle : 'Resolved updates remain available for review.'}
                  incidents={group.incidents}
                  expandedIncidents={expandedIncidents}
                  onToggleExpand={(incidentId) => setExpandedIncidents((previous) => {
                    const next = new Set(previous)
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
      </main>

      <Footer centerText={settings.footer.text} showPoweredBy={settings.footer.showPoweredBy} />
    </div>
  )
}
