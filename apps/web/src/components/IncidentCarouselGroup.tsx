import React, { useEffect, useMemo, useRef, useState } from 'react'
import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp } from 'lucide-react'
import type { Incident } from '../types'
import { formatDate, INCIDENT_IMPACT_LABELS, INCIDENT_STATUS_LABELS } from '../lib/utils'
import { IncidentTimeline } from './IncidentTimeline'

interface IncidentCarouselGroupProps {
  title: string
  incidents: Incident[]
  expandedIncidents: Set<string>
  onToggleExpand: (incidentId: string) => void
  emptyMessage?: string
  subtitle?: string
}

function getImpactToken(impact: Incident['impact']): string {
  switch (impact) {
    case 'critical':
      return '--status-major'
    case 'major':
      return '--status-partial'
    case 'minor':
      return '--status-degraded'
    default:
      return '--status-operational'
  }
}

function getStatusToken(status: Incident['status']): string {
  switch (status) {
    case 'investigating':
      return '--warning'
    case 'identified':
      return '--partial'
    case 'monitoring':
      return '--info'
    case 'resolved':
      return '--success'
    default:
      return '--text-subtle'
  }
}

export function IncidentCarouselGroup({
  title,
  incidents,
  expandedIncidents,
  onToggleExpand,
  emptyMessage = 'No incidents in this group.',
  subtitle,
}: IncidentCarouselGroupProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const pointerStartX = useRef<number | null>(null)

  useEffect(() => {
    setCurrentIndex((prev) => Math.min(prev, Math.max(incidents.length - 1, 0)))
  }, [incidents.length])

  const currentIncident = incidents[currentIndex]
  const currentPositionLabel = `${Math.min(currentIndex + 1, Math.max(incidents.length, 1))} / ${Math.max(incidents.length, 1)}`

  const severityStyle = useMemo(() => {
    if (!currentIncident) {
      return {
        backgroundColor: 'var(--surface-incident)',
        borderColor: 'var(--border-incident)',
      }
    }

    const impactToken = getImpactToken(currentIncident.impact)
    return {
      backgroundColor: 'color-mix(in srgb, var(--surface) 88%, var(--surface-incident))',
      borderColor: `color-mix(in srgb, var(${impactToken}) 22%, var(--border))`,
      boxShadow: `inset 4px 0 0 var(${impactToken})`,
    }
  }, [currentIncident])

  const move = (delta: number) => {
    if (incidents.length <= 1) return
    setCurrentIndex((prev) => {
      const next = prev + delta
      if (next < 0) return incidents.length - 1
      if (next >= incidents.length) return 0
      return next
    })
  }

  const handlePointerDown: React.PointerEventHandler<HTMLDivElement> = (event) => {
    pointerStartX.current = event.clientX
  }

  const handlePointerUp: React.PointerEventHandler<HTMLDivElement> = (event) => {
    if (pointerStartX.current === null || incidents.length <= 1) {
      pointerStartX.current = null
      return
    }

    const delta = event.clientX - pointerStartX.current
    pointerStartX.current = null

    if (Math.abs(delta) < 40) return
    move(delta < 0 ? 1 : -1)
  }

  return (
    <div
      className="rounded-2xl border p-4 sm:p-5"
      style={{
        borderColor: 'var(--border)',
        backgroundColor: 'var(--surface)',
      }}
    >
      <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base sm:text-lg font-semibold" style={{ color: 'var(--text)' }}>{title}</h3>
            <span
              className="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium"
              style={{
                backgroundColor: 'var(--surface-incident)',
                color: 'var(--text-muted)',
                border: '1px solid var(--border-incident)',
              }}
            >
              {incidents.length}
            </span>
          </div>
          {subtitle && (
            <p className="text-sm mt-1" style={{ color: 'var(--text-muted)' }}>{subtitle}</p>
          )}
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => move(-1)}
            disabled={incidents.length <= 1}
            className="inline-flex h-9 w-9 items-center justify-center rounded-full border disabled:opacity-40 disabled:cursor-not-allowed"
            style={{
              borderColor: 'var(--border)',
              color: 'var(--text)',
              backgroundColor: 'var(--surface)',
            }}
            aria-label={`Previous incident in ${title}`}
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <span className="min-w-[3rem] text-center text-xs font-medium" style={{ color: 'var(--text-subtle)' }}>
            {currentPositionLabel}
          </span>
          <button
            type="button"
            onClick={() => move(1)}
            disabled={incidents.length <= 1}
            className="inline-flex h-9 w-9 items-center justify-center rounded-full border disabled:opacity-40 disabled:cursor-not-allowed"
            style={{
              borderColor: 'var(--border)',
              color: 'var(--text)',
              backgroundColor: 'var(--surface)',
            }}
            aria-label={`Next incident in ${title}`}
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {incidents.length === 0 || !currentIncident ? (
        <div
          className="rounded-2xl border border-dashed px-4 py-6 text-sm"
          style={{
            borderColor: 'var(--border)',
            color: 'var(--text-muted)',
            backgroundColor: 'var(--surface-uptime)',
          }}
        >
          {emptyMessage}
        </div>
      ) : (
        <div
          className="rounded-2xl border p-4 sm:p-5 select-none"
          style={severityStyle}
          onPointerDown={handlePointerDown}
          onPointerUp={handlePointerUp}
        >
          <div className="flex flex-col gap-4">
            <div className="flex items-start justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 mb-3">
                  <span
                    className="inline-flex h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: `var(${getImpactToken(currentIncident.impact)})` }}
                    aria-hidden="true"
                  />
                  <span className="text-xs font-medium uppercase tracking-[0.18em]" style={{ color: 'var(--text-subtle)' }}>
                    {INCIDENT_IMPACT_LABELS[currentIncident.impact]} severity
                  </span>
                </div>

                <h4 className="text-lg font-semibold leading-tight" style={{ color: 'var(--text)' }}>
                  {currentIncident.title}
                </h4>

                <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.16em]" style={{ color: 'var(--text-subtle)' }}>Status</p>
                    <p className="text-sm font-medium mt-1" style={{ color: `var(${getStatusToken(currentIncident.status)})` }}>
                      {INCIDENT_STATUS_LABELS[currentIncident.status]}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.16em]" style={{ color: 'var(--text-subtle)' }}>Started</p>
                    <p className="text-sm font-medium mt-1" style={{ color: 'var(--text)' }}>
                      {formatDate(currentIncident.createdAt)}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.16em]" style={{ color: 'var(--text-subtle)' }}>Severity</p>
                    <p className="text-sm font-medium mt-1" style={{ color: 'var(--text)' }}>
                      {INCIDENT_IMPACT_LABELS[currentIncident.impact]}
                    </p>
                  </div>
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.16em]" style={{ color: 'var(--text-subtle)' }}>Resolved</p>
                    <p className="text-sm font-medium mt-1" style={{ color: 'var(--text)' }}>
                      {currentIncident.resolvedAt ? formatDate(currentIncident.resolvedAt) : 'Still active'}
                    </p>
                  </div>
                </div>
              </div>

              <button
                type="button"
                onClick={() => onToggleExpand(currentIncident.id)}
                className="inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full border"
                style={{
                  borderColor: 'var(--border)',
                  color: 'var(--text-subtle)',
                  backgroundColor: 'var(--surface)',
                }}
                aria-label={expandedIncidents.has(currentIncident.id) ? 'Collapse incident details' : 'Expand incident details'}
              >
                {expandedIncidents.has(currentIncident.id) ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
              </button>
            </div>

            <p className="text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
              {currentIncident.description}
            </p>

            {currentIncident.creatorUsername && (
              <p className="text-xs" style={{ color: 'var(--text-subtle)' }}>
                Reported by {currentIncident.creatorUsername}
              </p>
            )}

            {expandedIncidents.has(currentIncident.id) && <IncidentTimeline updates={currentIncident.updates || []} />}
          </div>
        </div>
      )}
    </div>
  )
}

export default IncidentCarouselGroup
