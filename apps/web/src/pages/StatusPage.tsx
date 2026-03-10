import React, { useState, useCallback, useMemo } from 'react'
import { CheckCircle, AlertTriangle, AlertCircle, XCircle, Wrench, ChevronDown, ChevronUp } from 'lucide-react'
import { useApi } from '../hooks/useApi'
import { useWebSocket } from '../hooks/useWebSocket'
import type { StatusSummary, ComponentWithSubs, Incident, Maintenance } from '../types'
import { STATUS_COLORS, STATUS_LABELS, STATUS_TEXT_COLORS, getOverallStatusLabel, formatDate, formatDateShort, INCIDENT_STATUS_LABELS, INCIDENT_IMPACT_LABELS } from '../lib/utils'
import { IncidentTimeline } from '../components/IncidentTimeline'

function StatusIcon({ status }: { status: string }) {
  const cls = 'w-5 h-5'
  switch (status) {
    case 'operational': return <CheckCircle className={`${cls} text-green-500`} />
    case 'degraded_performance': return <AlertTriangle className={`${cls} text-yellow-500`} />
    case 'partial_outage': return <AlertCircle className={`${cls} text-orange-500`} />
    case 'major_outage': return <XCircle className={`${cls} text-red-500`} />
    case 'maintenance': return <Wrench className={`${cls} text-blue-500`} />
    default: return <CheckCircle className={`${cls} text-green-500`} />
  }
}

function UptimeBar({ bars }: { bars: { date: string; uptimePercent: number; status: string }[] }) {
  return (
    <div className="flex gap-px items-end h-8 mt-2">
      {bars.map((bar, i) => (
        <div
          key={i}
          className={`flex-1 rounded-sm ${STATUS_COLORS[bar.status as keyof typeof STATUS_COLORS] || 'bg-green-500'} opacity-80 hover:opacity-100 transition-opacity cursor-pointer`}
          style={{ height: `${Math.max(20, bar.uptimePercent / 100 * 32)}px` }}
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
  }, [refetchComponents, refetchSummary, refetchIncidents])

  useWebSocket(handleWsMessage)

  const overallStatus = summary?.overallStatus || 'operational'
  const activeIncidents = incidentData?.active || []
  const resolvedIncidents = incidentData?.resolved || []
  const upcomingMaintenance = maintenanceData?.filter(m => m.status !== 'completed') || []
  const [expandedIncidents, setExpandedIncidents] = useState<Set<string>>(new Set())

  const headerBg: Record<string, string> = {
    operational: 'bg-green-600',
    degraded_performance: 'bg-yellow-500',
    partial_outage: 'bg-orange-500',
    major_outage: 'bg-red-600',
    maintenance: 'bg-blue-600',
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className={`${headerBg[overallStatus] || 'bg-green-600'} text-white py-12 px-4`}>
        <div className="max-w-4xl mx-auto">
          <h1 className="text-3xl font-bold mb-2">System Status</h1>
          <div className="flex items-center gap-3 text-xl">
            <CheckCircle className="w-7 h-7" />
            <span>{getOverallStatusLabel(overallStatus as any)}</span>
          </div>
          {activeIncidents.length > 0 && (
            <p className="mt-2 text-white/80 text-sm">{activeIncidents.length} active incident{activeIncidents.length > 1 ? 's' : ''}</p>
          )}
        </div>
      </div>

      <div className="max-w-4xl mx-auto px-4 py-8 space-y-8">

        {/* Active Incidents Banner */}
        {activeIncidents.map(incident => {
          const isExpanded = expandedIncidents.has(incident.id)
          return (
            <div key={incident.id} className="bg-red-50 border border-red-200 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <XCircle className="w-5 h-5 text-red-500 mt-0.5 flex-shrink-0" />
                <div className="flex-1">
                  <div className="flex items-start justify-between">
                    <div>
                      <h3 className="font-semibold text-red-900">{incident.title}</h3>
                      <p className="text-red-700 text-sm mt-1">{incident.description}</p>
                      <div className="flex gap-4 mt-2 text-xs text-red-600">
                        <span>Status: {INCIDENT_STATUS_LABELS[incident.status]}</span>
                        <span>Impact: {INCIDENT_IMPACT_LABELS[incident.impact]}</span>
                        <span>Since: {formatDate(incident.createdAt)}</span>
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
                      className="text-gray-400 hover:text-gray-600 flex-shrink-0"
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
          <div key={m.id} className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <Wrench className="w-5 h-5 text-blue-500 mt-0.5 flex-shrink-0" />
              <div>
                <h3 className="font-semibold text-blue-900">{m.title}</h3>
                <p className="text-blue-700 text-sm mt-1">{m.description}</p>
                <div className="flex gap-4 mt-2 text-xs text-blue-600">
                  <span>Status: {m.status.replace('_', ' ')}</span>
                  <span>{formatDate(m.startTime)} → {formatDate(m.endTime)}</span>
                </div>
              </div>
            </div>
          </div>
        ))}

        {/* Components */}
        {(components || []).map(comp => (
          <div key={comp.id} className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">{comp.name}</h2>
                {comp.description && <p className="text-sm text-gray-500 mt-0.5">{comp.description}</p>}
              </div>
              <div className="flex items-center gap-2">
                <StatusIcon status={comp.status} />
                <span className={`text-sm font-medium ${STATUS_TEXT_COLORS[comp.status]}`}>
                  {STATUS_LABELS[comp.status]}
                </span>
              </div>
            </div>

            {/* SubComponents */}
            {comp.subComponents && comp.subComponents.length > 0 && (
              <div className="divide-y divide-gray-50">
                {comp.subComponents.map(sub => (
                  <div key={sub.id} className="flex items-center justify-between px-6 py-3">
                    <span className="text-sm text-gray-700 pl-4">{sub.name}</span>
                    <div className="flex items-center gap-2">
                      <StatusIcon status={sub.status} />
                      <span className={`text-xs font-medium ${STATUS_TEXT_COLORS[sub.status]}`}>
                        {STATUS_LABELS[sub.status]}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* 90-day uptime */}
            {comp.uptimeHistory && comp.uptimeHistory.length > 0 && (
              <div className="px-6 py-4 bg-gray-50 border-t border-gray-100">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-gray-500">90-day uptime</span>
                  <span className="text-xs text-gray-500">
                    {comp.uptimeHistory.length > 0
                      ? `${(comp.uptimeHistory.reduce((s, b) => s + b.uptimePercent, 0) / comp.uptimeHistory.length).toFixed(2)}% avg`
                      : ''}
                  </span>
                </div>
                <UptimeBar bars={comp.uptimeHistory} />
                <div className="flex justify-between mt-1">
                  <span className="text-xs text-gray-400">{formatDateShort(comp.uptimeHistory[0]?.date)}</span>
                  <span className="text-xs text-gray-400">Today</span>
                </div>
              </div>
            )}
          </div>
        ))}

        {/* Incident History */}
        {resolvedIncidents.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Incident History</h2>
            <div className="space-y-4">
              {resolvedIncidents.map(incident => {
                const isExpanded = expandedIncidents.has(incident.id)
                return (
                  <div key={incident.id} className="bg-white rounded-xl border border-gray-200 p-5">
                    <div className="flex items-start justify-between">
                      <div>
                        <h3 className="font-medium text-gray-900">{incident.title}</h3>
                        <p className="text-sm text-gray-600 mt-1">{incident.description}</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="text-xs bg-green-100 text-green-700 px-2 py-1 rounded-full font-medium">
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
                          className="text-gray-400 hover:text-gray-600 flex-shrink-0"
                        >
                          {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                        </button>
                      </div>
                    </div>
                    <div className="flex gap-4 mt-3 text-xs text-gray-400">
                      <span>Created: {formatDate(incident.createdAt)}</span>
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
        <div className="text-center py-8 border-t border-gray-200">
          <p className="text-sm text-gray-400">
            Powered by Status Platform •{' '}
            <a href="/admin" className="text-blue-500 hover:underline">Admin</a>
          </p>
        </div>
      </div>
    </div>
  )
}
