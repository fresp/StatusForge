import React, { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { IncidentCarouselGroup } from '../components/IncidentCarouselGroup'
import type { Incident, StatusPageSettings } from '../types'
import {
  formatDate,
  groupIncidentsByStatus,
} from '../lib/utils'
import { loadThemePresetStylesheet, getThemePresets, DEFAULT_THEME_PRESET } from '../lib/themePresetLoader'

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

export default function HistoryPage() {
  const today = useMemo(() => new Date(), [])
  const currentQuarter = useMemo(() => getCurrentQuarterCursor(today), [today])
  const [selectedQuarter, setSelectedQuarter] = useState<QuarterCursor>(currentQuarter)

  const quarterRange = useMemo(() => getQuarterDateRange(selectedQuarter, today), [selectedQuarter, today])
  const incidentsUrl = useMemo(
    () =>
      `/status/incidents?start_date=${encodeURIComponent(quarterRange.startDate)}&end_date=${encodeURIComponent(quarterRange.endDate)}`,
    [quarterRange.endDate, quarterRange.startDate]
  )

  const { data: incidentData, loading: incidentsLoading, error: incidentsError } =
    useApi<{ active: Incident[]; resolved: Incident[] }>(incidentsUrl, [incidentsUrl])
  const { data: settingsData } = useApi<StatusPageSettings>('/status/settings')
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

  const allIncidents = useMemo(
    () => [...(incidentData?.active ?? []), ...(incidentData?.resolved ?? [])],
    [incidentData]
  )

  const monthBuckets = useMemo(
    () => groupIncidentsByQuarterMonths(allIncidents, selectedQuarter),
    [allIncidents, selectedQuarter]
  )

  const canGoNext = toQuarterIndex(selectedQuarter) < toQuarterIndex(currentQuarter)

  const themePreset = (settingsData?.theme?.preset?.trim() || DEFAULT_THEME_PRESET).endsWith('.css')
    ? settingsData?.theme?.preset?.trim() || DEFAULT_THEME_PRESET
    : `${settingsData?.theme?.preset?.trim() || DEFAULT_THEME_PRESET}.css`

  React.useEffect(() => {
    const pageTitle = settingsData?.head?.title?.trim() || 'Status Page'
    document.title = `${pageTitle} - Incident History`
  }, [settingsData?.head?.title])

  React.useEffect(() => {
    const presets = getThemePresets().presets
    loadThemePresetStylesheet(themePreset, presets).catch(() => {})
  }, [themePreset])

  return (
    <div
      className="min-h-screen"
      style={{
        backgroundColor: 'var(--bg)',
        color: 'var(--text)',
        fontFamily: 'var(--font-family)',
      }}
    >
      <div
        className="py-10 px-4 border-b"
        style={{
          borderColor: 'var(--border)',
          backgroundColor: 'var(--surface)',
        }}
      >
        <div className="max-w-5xl mx-auto flex items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold">Incident History</h1>
            <p className="text-sm mt-1" style={{ color: 'var(--text-muted)' }}>
              Browse incidents by quarter and month.
            </p>
          </div>
          <Link
            to="/"
            className="inline-flex items-center rounded-lg px-4 py-2 text-sm font-medium border"
            style={{
              borderColor: 'var(--border)',
              color: 'var(--text)',
              backgroundColor: 'var(--surface)',
            }}
          >
            Back to Status
          </Link>
        </div>
      </div>

      <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">
        <div
          className="rounded-xl border p-4 flex items-center justify-between"
          style={{
            borderColor: 'var(--border)',
            backgroundColor: 'var(--surface)',
          }}
        >
          <button
            onClick={() => setSelectedQuarter((prev) => shiftQuarter(prev, -1))}
            className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-sm font-medium border"
            style={{
              borderColor: 'var(--border)',
              color: 'var(--text)',
              backgroundColor: 'var(--surface)',
            }}
            aria-label="Show previous quarter"
          >
            <ChevronLeft className="w-4 h-4" />
            Prev
          </button>

          <div className="text-sm font-semibold" style={{ color: 'var(--text)' }}>
            {'< '} {formatQuarterLabel(selectedQuarter)} {' >'}
          </div>

          <button
            onClick={() => setSelectedQuarter((prev) => shiftQuarter(prev, 1))}
            disabled={!canGoNext}
            className="inline-flex items-center gap-1 rounded-lg px-3 py-2 text-sm font-medium border disabled:opacity-50 disabled:cursor-not-allowed"
            style={{
              borderColor: 'var(--border)',
              color: 'var(--text)',
              backgroundColor: 'var(--surface)',
            }}
            aria-label="Show next quarter"
          >
            Next
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>

        {incidentsLoading ? (
          <div
            className="rounded-xl border p-6"
            style={{
              borderColor: 'var(--border)',
              backgroundColor: 'var(--surface)',
            }}
          >
            <p className="text-sm" style={{ color: 'var(--text-muted)' }}>Loading incidents...</p>
          </div>
        ) : incidentsError ? (
          <div
            className="rounded-xl border p-6"
            style={{
              borderColor: 'var(--border)',
              backgroundColor: 'var(--surface)',
            }}
          >
            <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
              Unable to load incidents for this quarter.
            </p>
          </div>
        ) : (
          <section className="space-y-4">
            {monthBuckets.map((monthGroup) => (
              <div
                key={`${selectedQuarter.year}-${monthGroup.monthIndex}`}
                className="rounded-xl border p-5"
                style={{
                  borderColor: 'var(--border)',
                  backgroundColor: 'var(--surface)',
                }}
              >
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-lg font-semibold">{monthGroup.monthLabel}</h3>
                  <span className="text-xs" style={{ color: 'var(--text-subtle)' }}>
                    {monthGroup.incidents.length} incident{monthGroup.incidents.length === 1 ? '' : 's'}
                  </span>
                </div>

                {monthGroup.incidents.length === 0 ? (
                  <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
                    No incidents reported in this month.
                  </p>
                ) : (
                  <div className="space-y-3">
                    {groupIncidentsByStatus(monthGroup.incidents).map((statusGroup) => (
                      <IncidentCarouselGroup
                        key={`${selectedQuarter.year}-${monthGroup.monthIndex}-${statusGroup.key}`}
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
                    ))}
                  </div>
                )}
              </div>
            ))}
          </section>
        )}
      </div>
    </div>
  )
}
