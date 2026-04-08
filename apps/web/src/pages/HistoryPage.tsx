import { useEffect, useMemo, useState, type CSSProperties } from 'react'
import { BarChart3, ChevronLeft, ChevronRight, History, TimerReset } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { IncidentCarouselGroup } from '../components/IncidentCarouselGroup'
import Footer from '../components/layout/Footer'
import StatusTopNav from '../components/status/StatusTopNav'
import type { Incident, StatusPageSettings, StatusSummary } from '../types'
import {
  formatDate,
  groupIncidentsByStatus,
} from '../lib/utils'
import { loadThemePresetStylesheet, getThemePresets, DEFAULT_THEME_PRESET } from '../lib/themePresetLoader'
import {
  DEFAULT_STATUS_PAGE_SETTINGS,
  applyStatusPageHeadSettings,
  normalizeStatusPageSettings,
  readCachedStatusPageSettings,
} from '../lib/statusPageSettings'

interface QuarterCursor {
  year: number
  quarter: number
}

interface MonthBucket {
  monthIndex: number
  monthLabel: string
  incidents: Incident[]
}

const MONTH_SHORT = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

function toLocalBoundaryIso(date: Date, endOfDay: boolean): string {
  const boundary = new Date(date)
  if (endOfDay) {
    boundary.setHours(23, 59, 59, 999)
  } else {
    boundary.setHours(0, 0, 0, 0)
  }
  return boundary.toISOString()
}

function getCurrentQuarterCursor(reference = new Date()): QuarterCursor {
  return {
    year: reference.getFullYear(),
    quarter: Math.floor(reference.getMonth() / 3),
  }
}

function toQuarterIndex(cursor: QuarterCursor): number {
  return cursor.year * 4 + cursor.quarter
}

function shiftQuarter(cursor: QuarterCursor, delta: number): QuarterCursor {
  const absoluteQuarter = toQuarterIndex(cursor) + delta
  const year = Math.floor(absoluteQuarter / 4)
  const quarter = ((absoluteQuarter % 4) + 4) % 4

  return { year, quarter }
}

function formatQuarterLabel(cursor: QuarterCursor): string {
  const startMonth = cursor.quarter * 3
  const endMonth = startMonth + 2

  return `${MONTH_SHORT[startMonth]} – ${MONTH_SHORT[endMonth]} ${cursor.year}`
}

function formatQuarterChip(cursor: QuarterCursor): string {
  return `Q${cursor.quarter + 1} ${cursor.year}`
}

function getQuarterDateRange(cursor: QuarterCursor, today: Date): { startDate: string; endDate: string } {
  const quarterStartMonth = cursor.quarter * 3
  const start = new Date(cursor.year, quarterStartMonth, 1)

  const isCurrentQuarter =
    cursor.year === today.getFullYear() && cursor.quarter === Math.floor(today.getMonth() / 3)

  const end = isCurrentQuarter
    ? new Date(today.getFullYear(), today.getMonth(), today.getDate())
    : new Date(cursor.year, quarterStartMonth + 3, 0)

  return {
    startDate: toLocalBoundaryIso(start, false),
    endDate: toLocalBoundaryIso(end, true),
  }
}

function groupIncidentsByQuarterMonths(incidents: Incident[], cursor: QuarterCursor): MonthBucket[] {
  const quarterStartMonth = cursor.quarter * 3

  return Array.from({ length: 3 }, (_, offset) => {
    const monthIndex = quarterStartMonth + (2 - offset)
    const monthLabel = new Date(cursor.year, monthIndex, 1).toLocaleString('en-US', { month: 'long' })

    const monthIncidents = incidents
      .filter((incident) => {
        const createdAt = new Date(incident.createdAt)
        return createdAt.getFullYear() === cursor.year && createdAt.getMonth() === monthIndex
      })
      .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())

    return {
      monthIndex,
      monthLabel,
      incidents: monthIncidents,
    }
  })
}

function getMeanTimeToResolveMinutes(incidents: Incident[]): number | null {
  const resolvedWithDuration = incidents.filter((incident) => incident.resolvedAt)
  if (resolvedWithDuration.length === 0) {
    return null
  }

  const totalMinutes = resolvedWithDuration.reduce((sum, incident) => {
    const startedAt = new Date(incident.createdAt).getTime()
    const resolvedAt = new Date(incident.resolvedAt as string).getTime()
    return sum + Math.max(0, resolvedAt - startedAt) / (1000 * 60)
  }, 0)

  return Math.round(totalMinutes / resolvedWithDuration.length)
}

function formatMinutesLabel(totalMinutes: number | null): string {
  if (totalMinutes === null) {
    return '—'
  }

  if (totalMinutes < 60) {
    return `${totalMinutes}m`
  }

  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`
}

function getMonthEmptyMessage(monthLabel: string): string {
  return `No incidents reported in ${monthLabel}.`
}

const sectionCardStyle: CSSProperties = {
  backgroundColor: 'var(--status-card-bg, rgba(255,255,255,0.88))',
  borderColor: 'var(--status-card-border, rgba(181,197,226,0.32))',
  boxShadow: 'var(--status-card-shadow, 0 24px 60px rgba(20,37,63,0.08))',
}

export default function HistoryPage() {
  const today = useMemo(() => new Date(), [])
  const currentQuarter = useMemo(() => getCurrentQuarterCursor(today), [today])
  const [selectedQuarter, setSelectedQuarter] = useState<QuarterCursor>(currentQuarter)
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

  const quarterRange = useMemo(() => getQuarterDateRange(selectedQuarter, today), [selectedQuarter, today])
  const incidentsUrl = useMemo(
    () =>
      `/status/incidents?start_date=${encodeURIComponent(quarterRange.startDate)}&end_date=${encodeURIComponent(quarterRange.endDate)}`,
    [quarterRange.endDate, quarterRange.startDate]
  )

  const { data: incidentData, loading: incidentsLoading, error: incidentsError } =
    useApi<{ active: Incident[]; resolved: Incident[] }>(incidentsUrl, [incidentsUrl])
  const { data: summaryData } = useApi<StatusSummary>('/status/summary')
  const { data: settingsData } = useApi<StatusPageSettings>('/status/settings')

  const settings = useMemo(
    () => normalizeStatusPageSettings(settingsData ?? readCachedStatusPageSettings() ?? DEFAULT_STATUS_PAGE_SETTINGS),
    [settingsData]
  )

  const allIncidents = useMemo(
    () => [...(incidentData?.active ?? []), ...(incidentData?.resolved ?? [])],
    [incidentData]
  )

  const monthBuckets = useMemo(
    () => groupIncidentsByQuarterMonths(allIncidents, selectedQuarter),
    [allIncidents, selectedQuarter]
  )

  const canGoNext = toQuarterIndex(selectedQuarter) < toQuarterIndex(currentQuarter)
  const overallStatus = summaryData?.overallStatus ?? 'operational'
  const totalIncidents = allIncidents.length
  const totalResolved = allIncidents.filter((incident) => Boolean(incident.resolvedAt)).length
  const meanTimeToResolve = useMemo(() => getMeanTimeToResolveMinutes(allIncidents), [allIncidents])

  const themePreset = (settings.theme.preset?.trim() || DEFAULT_THEME_PRESET).endsWith('.css')
    ? settings.theme.preset.trim() || DEFAULT_THEME_PRESET
    : `${settings.theme.preset.trim() || DEFAULT_THEME_PRESET}.css`

  const activeQuarterMonthIndexes = useMemo(
    () => monthBuckets.filter((monthGroup) => monthGroup.incidents.length > 0).map((monthGroup) => monthGroup.monthIndex),
    [monthBuckets]
  )

  const primaryMonthIndex = activeQuarterMonthIndexes[0] ?? monthBuckets[0]?.monthIndex ?? selectedQuarter.quarter * 3

  const monthChips = useMemo(
    () => monthBuckets
      .slice()
      .sort((a, b) => a.monthIndex - b.monthIndex)
      .map((monthGroup) => ({
        label: monthGroup.monthLabel,
        active: monthGroup.monthIndex === primaryMonthIndex,
        hasIncidents: monthGroup.incidents.length > 0,
      })),
    [monthBuckets, primaryMonthIndex]
  )

  useEffect(() => {
    applyStatusPageHeadSettings({
      ...settings,
      head: {
        ...settings.head,
        title: `${settings.head.title} - Incident History`,
      },
    })
  }, [settings])

  useEffect(() => {
    const presets = getThemePresets().presets
    loadThemePresetStylesheet(themePreset, presets).catch(() => { })
  }, [themePreset])

  return (
    <div
      className="min-h-screen"
      style={{
        backgroundColor: 'var(--bg)',
        color: 'var(--text)',
        fontFamily: 'var(--font-family)',
        backgroundImage: settings.branding.backgroundImageUrl
          ? `linear-gradient(var(--bg-image-overlay), var(--bg-image-overlay)), url(${settings.branding.backgroundImageUrl})`
          : undefined,
        backgroundSize: settings.branding.backgroundImageUrl ? 'cover' : undefined,
        backgroundAttachment: settings.branding.backgroundImageUrl ? 'fixed' : undefined,
        backgroundPosition: settings.branding.backgroundImageUrl ? 'center' : undefined,
      }}
    >
      <StatusTopNav
        siteName={settings.branding.siteName}
        logoUrl={settings.branding.logoUrl}
        statusLabel={overallStatus === 'operational' ? 'stable' : 'attention'}
        activeView="history"
      />

      <main className="pb-16">
        <section className="px-4 pt-5 md:px-6 md:pt-8">
          <div
            className="relative mx-auto max-w-6xl overflow-hidden rounded-[2rem] px-6 py-8 shadow-[0_34px_80px_rgba(15,143,103,0.16)] md:px-10 md:py-12"
            style={{
              background: 'linear-gradient(135deg, color-mix(in srgb, var(--surface) 34%, white) 0%, var(--status-card-bg, rgba(255,255,255,0.88)) 48%, color-mix(in srgb, var(--primary) 8%, var(--surface)) 100%)',
              border: '1px solid var(--status-card-border, rgba(181,197,226,0.32))',
            }}
          >
            <div
              className="absolute right-0 top-0 h-52 w-52 -translate-y-1/3 translate-x-1/4 rounded-full blur-3xl md:h-72 md:w-72"
              style={{ backgroundColor: 'color-mix(in srgb, var(--primary) 16%, transparent)' }}
            />
            <div className="relative z-10 grid gap-8 lg:grid-cols-[minmax(0,1fr)_18rem] lg:items-end">
              <div className="max-w-3xl">
                <div className="mb-5 inline-flex items-center gap-2 rounded-full border px-4 py-2 backdrop-blur-md" style={{ borderColor: 'color-mix(in srgb, var(--border) 40%, transparent)', backgroundColor: 'color-mix(in srgb, var(--surface) 72%, transparent)' }}>
                  <History className="h-4 w-4" style={{ color: 'var(--primary)' }} />
                  <span className="text-[11px] font-extrabold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
                    System reliability archive
                  </span>
                </div>
                <h1 className="max-w-3xl text-4xl font-black leading-[0.95] tracking-[-0.04em] md:text-6xl" style={{ color: 'var(--text)' }}>
                  Incident history with
                  <span className="block" style={{ color: 'var(--primary)' }}>editorial clarity.</span>
                </h1>
                <p className="mt-5 max-w-2xl text-sm font-medium leading-6 md:text-lg md:leading-8" style={{ color: 'var(--text-muted)' }}>
                  A transparent archive of operational events, grouped by quarter and month so teams can trace reliability trends without losing incident detail.
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
                <div className="rounded-[1.5rem] p-5" style={{ backgroundColor: 'var(--primary)', color: 'var(--on-primary, #fff)' }}>
                  <p className="text-[11px] font-extrabold uppercase tracking-[0.24em] opacity-80">Quarter view</p>
                  <p className="mt-3 text-3xl font-black tracking-[-0.04em]">{formatQuarterChip(selectedQuarter)}</p>
                  <p className="mt-2 text-sm opacity-90">{totalIncidents} archived incidents</p>
                </div>
                <div className="rounded-[1.5rem] p-5" style={{ ...sectionCardStyle, backgroundColor: 'color-mix(in srgb, var(--status-card-bg, rgba(255,255,255,0.88)) 78%, var(--primary) 6%)' }}>
                  <TimerReset className="h-5 w-5" style={{ color: 'var(--primary)' }} />
                  <p className="mt-4 text-[11px] font-extrabold uppercase tracking-[0.24em]" style={{ color: 'var(--text-subtle)' }}>Mean resolve time</p>
                  <p className="mt-2 text-3xl font-black tracking-[-0.04em]" style={{ color: 'var(--text)' }}>{formatMinutesLabel(meanTimeToResolve)}</p>
                </div>
                <div className="rounded-[1.5rem] p-5" style={{ ...sectionCardStyle, backgroundColor: 'color-mix(in srgb, var(--status-card-bg, rgba(255,255,255,0.88)) 78%, var(--primary) 6%)' }}>
                  <BarChart3 className="h-5 w-5" style={{ color: 'var(--primary)' }} />
                  <p className="mt-4 text-[11px] font-extrabold uppercase tracking-[0.24em]" style={{ color: 'var(--text-subtle)' }}>Resolved in range</p>
                  <p className="mt-2 text-3xl font-black tracking-[-0.04em]" style={{ color: 'var(--text)' }}>{totalResolved}</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="px-4 pt-8 md:px-6 md:pt-10">
          <div className="mx-auto max-w-6xl rounded-[1.8rem] border p-5 md:p-6" style={sectionCardStyle}>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <p className="text-[11px] font-extrabold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
                  Time navigation
                </p>
                <h2 className="mt-3 text-2xl font-black tracking-[-0.03em] md:text-3xl" style={{ color: 'var(--text)' }}>
                  {formatQuarterLabel(selectedQuarter)}
                </h2>
                <p className="mt-2 text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
                  Review the quarter archive month by month and inspect each incident timeline in full context.
                </p>
              </div>

              <div className="flex items-center gap-3 self-start lg:self-auto">
                <button
                  onClick={() => setSelectedQuarter((prev) => shiftQuarter(prev, -1))}
                  className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-bold uppercase tracking-[0.18em]"
                  style={{
                    backgroundColor: 'var(--surface)',
                    color: 'var(--text-muted)',
                  }}
                  aria-label="Show previous quarter"
                >
                  <ChevronLeft className="h-4 w-4" />
                  Prev
                </button>
                <button
                  onClick={() => setSelectedQuarter((prev) => shiftQuarter(prev, 1))}
                  disabled={!canGoNext}
                  className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-bold uppercase tracking-[0.18em] disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{
                    backgroundColor: 'var(--surface)',
                    color: 'var(--text-muted)',
                  }}
                  aria-label="Show next quarter"
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            </div>

            <div className="mt-6 flex flex-wrap gap-3">
              {monthChips.map((chip) => (
                <span
                  key={chip.label}
                  className="inline-flex items-center rounded-full px-4 py-2 text-xs font-bold uppercase tracking-[0.16em]"
                  style={{
                    backgroundColor: chip.active ? 'var(--status-pill-bg, rgba(16,185,129,0.1))' : 'color-mix(in srgb, var(--surface) 92%, transparent)',
                    color: chip.active ? 'var(--status-pill-text, var(--primary))' : (chip.hasIncidents ? 'var(--text-muted)' : 'var(--text-subtle)'),
                    border: chip.active ? '1px solid transparent' : '1px solid color-mix(in srgb, var(--border) 40%, transparent)',
                  }}
                >
                  {chip.label}
                </span>
              ))}
            </div>
          </div>
        </section>

        <section className="px-4 pt-8 md:px-6 md:pt-10">
          <div className="mx-auto max-w-6xl space-y-6 md:space-y-8">
            {incidentsLoading ? (
              <div className="rounded-[1.8rem] border p-6 md:p-8" style={sectionCardStyle}>
                <p className="text-sm" style={{ color: 'var(--text-muted)' }}>Loading incidents...</p>
              </div>
            ) : incidentsError ? (
              <div className="rounded-[1.8rem] border p-6 md:p-8" style={sectionCardStyle}>
                <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
                  Unable to load incidents for this quarter.
                </p>
              </div>
            ) : (
              monthBuckets.map((monthGroup) => (
                <div
                  key={`${selectedQuarter.year}-${monthGroup.monthIndex}`}
                  className="rounded-[1.8rem] border p-5 md:p-7"
                  style={sectionCardStyle}
                >
                  <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                    <div>
                      <div className="inline-flex items-center gap-2 rounded-full px-3 py-1.5" style={{ backgroundColor: 'color-mix(in srgb, var(--surface) 86%, transparent)' }}>
                        <span className="h-2 w-2 rounded-full" style={{ backgroundColor: monthGroup.incidents.length > 0 ? 'var(--primary)' : 'var(--text-subtle)' }} />
                        <span className="text-[11px] font-extrabold uppercase tracking-[0.24em]" style={{ color: 'var(--text-subtle)' }}>
                          Month archive
                        </span>
                      </div>
                      <h2 className="mt-4 text-2xl font-black tracking-[-0.03em] md:text-3xl" style={{ color: 'var(--text)' }}>
                        {monthGroup.monthLabel}
                      </h2>
                    </div>
                    <div className="text-left md:text-right">
                      <p className="text-[11px] font-extrabold uppercase tracking-[0.24em]" style={{ color: 'var(--text-subtle)' }}>
                        Incidents in month
                      </p>
                      <p className="mt-2 text-3xl font-black tracking-[-0.04em]" style={{ color: 'var(--text)' }}>
                        {monthGroup.incidents.length}
                      </p>
                    </div>
                  </div>

                  {monthGroup.incidents.length === 0 ? (
                    <div className="mt-6 rounded-[1.5rem] px-6 py-10 text-center" style={{ backgroundColor: 'color-mix(in srgb, var(--surface) 90%, transparent)' }}>
                      <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full" style={{ backgroundColor: 'color-mix(in srgb, var(--primary) 16%, transparent)' }}>
                        <History className="h-7 w-7" style={{ color: 'var(--primary)' }} />
                      </div>
                      <h3 className="mt-5 text-xl font-black tracking-[-0.02em]" style={{ color: 'var(--text)' }}>
                        Quiet month
                      </h3>
                      <p className="mx-auto mt-3 max-w-md text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
                        {getMonthEmptyMessage(monthGroup.monthLabel)} The archive remained clear for this period.
                      </p>
                    </div>
                  ) : (
                    <div className="mt-6 space-y-5">
                      {groupIncidentsByStatus(monthGroup.incidents).map((statusGroup) => (
                        <div
                          key={`${selectedQuarter.year}-${monthGroup.monthIndex}-${statusGroup.key}`}
                          className="rounded-[1.4rem] border px-4 py-3 md:px-5"
                          style={{
                            borderColor: 'color-mix(in srgb, var(--status-card-border, rgba(181,197,226,0.32)) 72%, transparent)',
                            backgroundColor: 'color-mix(in srgb, var(--surface) 92%, transparent)',
                          }}
                        >
                          <IncidentCarouselGroup
                            title={statusGroup.label}
                            subtitle={`${monthGroup.monthLabel} · ${formatDate(monthGroup.incidents[0].createdAt).split(',')[2]?.trim() || selectedQuarter.year}`}
                            incidents={statusGroup.incidents}
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
                      ))}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </section>
      </main>

      <Footer
        centerText={settings.footer.text}
        showPoweredBy={false}
        showHistoryLink={false}
      />
    </div>
  )
}
