import React from 'react'
import type { IncidentUpdate } from '../types'
import { INCIDENT_STATUS_LABELS, formatDate } from '../lib/utils'

interface IncidentTimelineProps {
  updates: IncidentUpdate[]
}

// Function to get status color based on incident status
const getStatusColors = (status: string) => {
  const colors: Record<string, { dot: string; text: string; bg: string }> = {
    investigating: { dot: 'bg-yellow-500', text: 'text-yellow-700', bg: 'bg-yellow-50 border-yellow-200' },
    identified: { dot: 'bg-orange-500', text: 'text-orange-700', bg: 'bg-orange-50 border-orange-200' },
    monitoring: { dot: 'bg-blue-500', text: 'text-blue-700', bg: 'bg-blue-50 border-blue-200' },
    resolved: { dot: 'bg-green-500', text: 'text-green-700', bg: 'bg-green-50 border-green-200' },
    default: { dot: 'bg-gray-400', text: 'text-gray-700', bg: 'bg-gray-50 border-gray-200' },
  };
  return colors[status] || colors.default;
};

// Individual timeline item component
const TimelineItem = ({ update }: { update: IncidentUpdate }) => {
  const statusColors = getStatusColors(update.status);

  return (
    <div className="relative pb-3" key={update.id}>
      {/* Vertical line */}
      <div className="absolute left-4 top-0 h-full w-0.5 bg-gray-200 -ml-px"></div>

      {/* Timeline dot */}
      <div className="relative flex items-start mb-2">
        <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${statusColors.bg}`}>
          <div className={`w-3 h-3 rounded-full ${statusColors.dot}`}></div>
        </div>

        {/* Content */}
        <div className="ml-4 flex-1 min-w-0">
          <div className="flex flex-wrap items-baseline gap-x-2">
            <span className="font-medium text-sm text-gray-800">
              {INCIDENT_STATUS_LABELS[update.status]}
            </span>
            <span className="text-xs text-gray-400">
              {formatDate(update.createdAt)}
            </span>
          </div>
          <p className="text-sm text-gray-600">
            {update.message}
          </p>
        </div>
      </div>
    </div>
  );
};

export function IncidentTimeline({ updates }: IncidentTimelineProps) {
  return (
    <div className="pl-4 mt-2 border-l border-gray-200 space-y-2">
      {updates.length === 0 ? (
        <p className="text-sm text-gray-400">No updates yet.</p>
      ) : (
        updates.map((update) => (
          <TimelineItem key={update.id} update={update} />
        ))
      )}
    </div>
  );
};

export default IncidentTimeline