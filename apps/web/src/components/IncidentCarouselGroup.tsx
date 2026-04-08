import React, { useEffect, useRef, useState } from 'react'
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
      return '--status-degraded'
    case 'identified':
      return '--status-partial'
    case 'monitoring':
      return '--primary'
    case 'resolved':
      return '--status-resolved-text'
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
  const componentNames = currentIncident
    ? (currentIncident.affectedComponentTargets && currentIncident.affectedComponentTargets.length > 0
      ? currentIncident.affectedComponentTargets
          .map((target) => {
            const subNames = (target.subComponents || []).map((subComponent) => subComponent.name)
            if (subNames.length === 0) {
              return target.component.name
            }
            return `${target.component.name} (${subNames.join(', ')})`
          })
          .join(', ')
      : (currentIncident.affectedComponents || []).map(component => component.name).join(', '))
    : ''

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
    <div className="py-1">
      <div className="flex flex-wrap items-start justify-between gap-3 pb-2">
        <div className="min-w-0">
          <div className="flex items-baseline gap-2">
            <h3 className="text-base font-medium" style={{ color: 'var(--text)' }}>{title}</h3>
            <span className="text-sm" style={{ color: 'var(--text-subtle)' }}>
              ({incidents.length})
            </span>
          </div>
          {subtitle && <p className="text-sm mt-1" style={{ color: 'var(--text-subtle)' }}>{subtitle}</p>}
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => move(-1)}
            disabled={incidents.length <= 1}
            className="inline-flex h-8 w-8 items-center justify-center rounded-full disabled:opacity-40 disabled:cursor-not-allowed"
            style={{
              color: 'var(--text-subtle)',
              backgroundColor: 'transparent',
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
            className="inline-flex h-8 w-8 items-center justify-center rounded-full disabled:opacity-40 disabled:cursor-not-allowed"
            style={{
              color: 'var(--text-subtle)',
              backgroundColor: 'transparent',
            }}
            aria-label={`Next incident in ${title}`}
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>

      {incidents.length === 0 || !currentIncident ? (
        <div className="py-3 text-sm" style={{ color: 'var(--text-muted)' }}>
          {emptyMessage}
        </div>
      ) : (
        <div
          className="py-3 border-b select-none"
          style={{ borderColor: 'color-mix(in srgb, var(--border) 60%, transparent)' }}
          onPointerDown={handlePointerDown}
          onPointerUp={handlePointerUp}
        >
          <div className="flex flex-col gap-y-2">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1 space-y-1">
                <div className="flex items-center gap-2">
                  <span
                    className="inline-flex h-2 w-2 rounded-full"
                    style={{ backgroundColor: `var(${getImpactToken(currentIncident.impact)})` }}
                    aria-hidden="true"
                  />
                  <h4 className="text-base font-medium leading-tight" style={{ color: 'var(--text)' }}>
                    {currentIncident.title}
                  </h4>
                </div>

                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm" style={{ color: 'var(--text-subtle)' }}>
                  <span>{formatDate(currentIncident.createdAt)}</span>
                  <span aria-hidden="true">•</span>
                  <span style={{ color: `var(${getImpactToken(currentIncident.impact)})` }}>
                    {INCIDENT_IMPACT_LABELS[currentIncident.impact]}
                  </span>
                  {componentNames && (
                    <>
                      <span aria-hidden="true">•</span>
                      <span>{componentNames}</span>
                    </>
                  )}
                  {currentIncident.resolvedAt && (
                    <>
                      <span aria-hidden="true">•</span>
                      <span>Resolved {formatDate(currentIncident.resolvedAt)}</span>
                    </>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-1 flex-shrink-0">
                <span className="text-xs font-medium" style={{ color: `var(${getStatusToken(currentIncident.status)})` }}>
                  {INCIDENT_STATUS_LABELS[currentIncident.status]}
                </span>
                <button
                  type="button"
                  onClick={() => onToggleExpand(currentIncident.id)}
                  className="inline-flex h-7 w-7 items-center justify-center rounded-md"
                  style={{ color: 'var(--text-subtle)', backgroundColor: 'transparent' }}
                  aria-label={expandedIncidents.has(currentIncident.id) ? 'Collapse incident details' : 'Expand incident details'}
                >
                  {expandedIncidents.has(currentIncident.id) ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                </button>
              </div>
            </div>

            <p className="text-sm leading-6" style={{ color: 'var(--text-muted)' }}>
              {currentIncident.description}
            </p>

            {currentIncident.creatorUsername && (
              <p className="text-xs" style={{ color: 'var(--text-subtle)' }}>
                Reported by {currentIncident.creatorUsername}
              </p>
            )}

            {expandedIncidents.has(currentIncident.id) && (
              <div className="pl-3">
                <IncidentTimeline updates={currentIncident.updates || []} />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default IncidentCarouselGroup
