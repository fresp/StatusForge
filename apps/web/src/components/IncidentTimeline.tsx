import React from 'react'
import type { IncidentUpdate } from '../types'
import { INCIDENT_STATUS_LABELS, formatDate } from '../lib/utils'

interface IncidentTimelineProps {
  updates: IncidentUpdate[]
}

export function IncidentTimeline({ updates }: IncidentTimelineProps) {
  return (
    <div className="pl-4 border-l-2 border-gray-200 space-y-2">
      {updates.length === 0 ? (
        <p className="text-sm text-gray-400">No updates yet.</p>
      ) : (
        updates.map((update) => (
          <div key={update.id} className="text-sm">
            <span className="font-medium text-gray-700">
              {INCIDENT_STATUS_LABELS[update.status]}
            </span>
            <span className="text-gray-500 ml-2">{update.message}</span>
            <span className="text-gray-400 ml-2 text-xs">{formatDate(update.createdAt)}</span>
          </div>
        ))
      )}
    </div>
  )
}

export default IncidentTimeline
