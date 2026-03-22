import React, { useState, useCallback, useEffect } from 'react'
import { CheckCircle, AlertTriangle, AlertCircle, XCircle, Wrench, ChevronDown, ChevronUp } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { useWebSocket } from '../hooks/useWebSocket'
import type { StatusSummary, ComponentWithSubs, Incident, Maintenance, StatusPageSettings } from '../types'
import { STATUS_LABELS, getOverallStatusLabel, formatDate, formatDateShort, INCIDENT_STATUS_LABELS, INCIDENT_IMPACT_LABELS } from '../lib/utils'
import { IncidentTimeline } from '../components/IncidentTimeline'
import { loadThemePresetStylesheet, getThemePresets, DEFAULT_THEME_PRESET } from '../lib/themePresetLoader'

const DEFAULT_SETTINGS: StatusPageSettings = {
  head: {
    title: 'Status Platform',
    description: 'Live system status and incident updates.',
    keywords: 'status, uptime, incidents, maintenance',
    faviconUrl: '/vite.svg',
    metaTags: {},
  },
  branding: {
    siteName: 'System Status',
    logoUrl: '',
    backgroundImageUrl: '',
    heroImageUrl: '',
  },
  theme: {
    preset: DEFAULT_THEME_PRESET,
  },
  footer: {
    text: '',
    showPoweredBy: true,
  },
  customCss: '',
  updatedAt: '',
  createdAt: '',
}

function normalizeSettings(settings?: StatusPageSettings | null): StatusPageSettings {
  if (!settings) {
    return DEFAULT_SETTINGS
  }

  const preset = settings.theme?.preset?.trim() || DEFAULT_THEME_PRESET
  const normalizedPreset = preset.endsWith('.css') ? preset : `${preset}.css`

  return {
    head: {
      title: settings.head?.title ?? DEFAULT_SETTINGS.head.title,
      description: settings.head?.description ?? DEFAULT_SETTINGS.head.description,
      keywords: settings.head?.keywords ?? DEFAULT_SETTINGS.head.keywords,
      faviconUrl: settings.head?.faviconUrl ?? DEFAULT_SETTINGS.head.faviconUrl,
      metaTags: settings.head?.metaTags || {},
    },
    branding: {
      siteName: settings.branding?.siteName ?? DEFAULT_SETTINGS.branding.siteName,
      logoUrl: settings.branding?.logoUrl ?? '',
      backgroundImageUrl: settings.branding?.backgroundImageUrl ?? '',
      heroImageUrl: settings.branding?.heroImageUrl ?? '',
    },
    theme: {
      preset: normalizedPreset,
      appliedPreset: settings.theme?.appliedPreset,
    },
    footer: {
      text: settings.footer?.text ?? '',
      showPoweredBy: settings.footer?.showPoweredBy ?? true,
    },
    customCss: settings.customCss ?? '',
    updatedAt: settings.updatedAt ?? '',
    createdAt: settings.createdAt ?? '',
  }
}

function upsertMetaTag(selector: string, content: string) {
  const existing = document.head.querySelector(`meta[${selector}]`)
  if (content) {
    if (existing) {
      existing.setAttribute('content', content)
      return
    }

    const meta = document.createElement('meta')
    const [attr, value] = selector.split('=')
    meta.setAttribute(attr, value.replace(/"/g, ''))
    meta.setAttribute('content', content)
    document.head.appendChild(meta)
    return
  }

  if (existing) {
    existing.remove()
  }
}

function setCustomMetaTags(metaTags: Record<string, string>) {
  const existing = document.head.querySelectorAll('meta[data-status-page-meta="true"]')
  existing.forEach(node => node.remove())

  Object.entries(metaTags).forEach(([key, value]) => {
    if (!key || !value) {
      return
    }

    const meta = document.createElement('meta')
    if (key.startsWith('og:') || key.startsWith('twitter:')) {
      meta.setAttribute('property', key)
    } else {
      meta.setAttribute('name', key)
    }
    meta.setAttribute('content', value)
    meta.setAttribute('data-status-page-meta', 'true')
    document.head.appendChild(meta)
  })
}

function upsertFavicon(url: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = url
}

function upsertCustomCss(css: string) {
  const id = 'status-page-custom-css'
  let styleEl = document.getElementById(id) as HTMLStyleElement | null
  if (!styleEl) {
    styleEl = document.createElement('style')
    styleEl.id = id
    document.head.appendChild(styleEl)
  }
  styleEl.textContent = css
}

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

type ThemeVariableStyle = React.CSSProperties & Record<`--${string}`, string>

function StatusIcon({ status }: { status: string }) {
  const cls = 'w-5 h-5'
  const color = `var(${getStatusToken(status)})`
  switch (status) {
    case 'operational': return <CheckCircle className={cls} style={{ color }} />
    case 'degraded_performance': return <AlertTriangle className={cls} style={{ color }} />
    case 'partial_outage': return <AlertCircle className={cls} style={{ color }} />
    case 'major_outage': return <XCircle className={cls} style={{ color }} />
    case 'maintenance': return <Wrench className={cls} style={{ color }} />
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
  const { data: summary, refetch: refetchSummary } = useApi<StatusSummary>('/status/summary')
  const { data: components, refetch: refetchComponents } = useApi<ComponentWithSubs[]>('/status/components')
  const { data: incidentData, refetch: refetchIncidents } = useApi<{ active: Incident[]; resolved: Incident[] }>('/status/incidents')
  const { data: settingsData, refetch: refetchSettings } = useApi<StatusPageSettings>('/status/settings')

  const { data: maintenanceData } = useApi<Maintenance[]>('/maintenance')

  const handleWsMessage = useCallback((event: { type: string }) => {
    if (['component_updated', 'component_created'].includes(event.type)) {
      refetchComponents()
      refetchSummary()
    }
    if (['incident_created', 'incident_updated', 'incident_resolved', 'incident_update_added'].includes(event.type)) {
      refetchIncidents()
      refetchSummary()
    }
    if (event.type === 'status_page_settings_updated') {
      refetchSettings()
    }
  }, [refetchComponents, refetchSummary, refetchIncidents, refetchSettings])

  useWebSocket(handleWsMessage)

  const overallStatus = summary?.overallStatus || 'operational'
  const settings = normalizeSettings(settingsData)
  const activeIncidents = incidentData?.active || []
  const resolvedIncidents = incidentData?.resolved || []
  const upcomingMaintenance = maintenanceData?.filter(m => m.status !== 'completed') || []
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

  useEffect(() => {
    document.title = settings.head.title
    upsertMetaTag('name="description"', settings.head.description)
    upsertMetaTag('name="keywords"', settings.head.keywords)
    setCustomMetaTags(settings.head.metaTags)
    upsertFavicon(settings.head.faviconUrl)
    upsertCustomCss(settings.customCss)
  }, [settings])

  useEffect(() => {
    const presets = getThemePresets().presets
    loadThemePresetStylesheet(settings.theme.preset, presets).catch(() => {})
  }, [settings.theme.preset])

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
        className="py-12 px-4"
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

        {/* Active Incidents Banner */}
        {activeIncidents.map(incident => {
          const isExpanded = expandedIncidents.has(incident.id)
          return (
            <div key={incident.id} className="border rounded-lg p-4" style={incidentSurfaceStyle}>
              <div className="flex items-start gap-3">
                <XCircle className="w-5 h-5 mt-0.5 flex-shrink-0" style={{ color: 'var(--status-major)' }} />
                <div className="flex-1">
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="font-semibold">{incident.title}</h3>
                      <p className="text-sm mt-1" style={{ color: mutedTextColor }}>{incident.description}</p>
                      <div className="flex gap-4 mt-2 text-xs" style={{ color: subtleTextColor }}>
                        <span>Status: {INCIDENT_STATUS_LABELS[incident.status]}</span>
                        <span>Impact: {INCIDENT_IMPACT_LABELS[incident.impact]}</span>
                        <span>Since: {formatDate(incident.createdAt)}</span>
                        {incident.creatorUsername && <span>Created by: {incident.creatorUsername}</span>}
                      </div>
                    </div>
                    <button
                      onClick={() => setExpandedIncidents(prev => {
                        const next = new Set(prev)
                        if (next.has(incident.id)) {
                          next.delete(incident.id)
                        } else {
                          next.add(incident.id)
                        }
                        return next
                      })}
                      className="flex-shrink-0 transition-colors"
                      style={{ color: 'var(--text-subtle)' }}
                    >
                      {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                    </button>
                  </div>
                  {isExpanded && <IncidentTimeline updates={incident.updates || []} />}
                </div>
              </div>
            </div>
          )
        })}

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
            className="rounded-xl shadow-sm border overflow-hidden"
            style={cardSurfaceStyle}
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

        {/* Incident History */}
        {resolvedIncidents.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold mb-4" style={sectionTitleStyle}>Incident History</h2>
            <div className="space-y-4">
              {resolvedIncidents.map(incident => {
                const isExpanded = expandedIncidents.has(incident.id)
                return (
                  <div key={incident.id} className="rounded-xl border p-5" style={cardSurfaceStyle}>
                    <div className="flex items-start justify-between">
                      <div>
                        <h3 className="font-medium" style={sectionTitleStyle}>{incident.title}</h3>
                        <p className="text-sm mt-1" style={{ color: mutedTextColor }}>{incident.description}</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-xs px-2 py-1 rounded-full font-medium" style={{ backgroundColor: 'var(--status-resolved-bg)', color: 'var(--status-resolved-text)' }}>
                          Resolved
                        </span>
                        <button
                          onClick={() => setExpandedIncidents(prev => {
                            const next = new Set(prev)
                            if (next.has(incident.id)) {
                              next.delete(incident.id)
                            } else {
                              next.add(incident.id)
                            }
                            return next
                          })}
                          className="flex-shrink-0 transition-colors"
                          style={{ color: 'var(--text-subtle)' }}
                        >
                          {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                        </button>
                      </div>
                    </div>
                    <div className="flex gap-4 mt-3 text-xs" style={{ color: subtleTextColor }}>
                      <span>Created: {formatDate(incident.createdAt)}</span>
                      {incident.creatorUsername && <span>Created by: {incident.creatorUsername}</span>}
                      {incident.resolvedAt && <span>Resolved: {formatDate(incident.resolvedAt)}</span>}
                    </div>
                    {isExpanded && <IncidentTimeline updates={incident.updates || []} />}
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* Footer */}
         <div className="text-center py-8 border-t" style={{ borderColor: 'var(--border)' }}>
          {settings.footer.text && (
            <p className="text-sm mb-1" style={{ color: mutedTextColor }}>{settings.footer.text}</p>
          )}
          {settings.footer.showPoweredBy && (
            <p className="text-sm" style={{ color: subtleTextColor }}>Powered by <a href='https://github.com/fresp/StatusForge'>StatusForge</a></p>
          )}
        </div>
      </div>
    </div>
  )
}
