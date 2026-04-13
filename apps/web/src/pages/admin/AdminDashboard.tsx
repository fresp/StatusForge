import React from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, Layers, Activity, Users, Wrench, TrendingUp } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import type { Component, Incident, Monitor, Subscriber, Maintenance } from '../../types'
import { STATUS_LABELS, STATUS_COLORS, INCIDENT_STATUS_LABELS, formatDate } from '../../lib/utils'

function StatCard({
  title,
  value,
  icon: Icon,
  iconClassName,
  to,
}: {
  title: string
  value: number | string
  icon: React.ElementType
  iconClassName: string
  to: string
}) {
  return (
    <Link
      to={to}
      className="admin-surface admin-surface-hover p-6 flex flex-col gap-4 group"
    >
      <div className="flex items-start justify-between">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${iconClassName}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div className="w-8 h-8 rounded-full flex items-center justify-center text-slate-400 group-hover:text-blue-500 group-hover:bg-blue-50 transition-colors">
          <TrendingUp className="w-4 h-4 opacity-0 -translate-x-1 translate-y-1 group-hover:opacity-100 group-hover:translate-x-0 group-hover:translate-y-0 transition-all" />
        </div>
      </div>
      <div>
        <p className="text-3xl font-bold text-slate-800 tracking-tight">{value}</p>
        <p className="text-[13px] font-medium text-slate-500 mt-1">{title}</p>
      </div>
    </Link>
  )
}

export default function AdminDashboard() {
  const { data: components, total: totalComponents } = useApi<Component[]>('/components')
  const { data: incidents, total: totalIncidents } = useApi<Incident[]>('/incidents')
  const { data: monitors, total: totalMonitors } = useApi<Monitor[]>('/monitors')
  const { data: subscribers, total: totalSubscribers } = useApi<Subscriber[]>('/subscribers')
  const { data: maintenance } = useApi<Maintenance[]>('/maintenance')

  const activeIncidents = incidents?.filter(i => i.status !== 'resolved') || []
  const scheduledMaintenance = Array.isArray(maintenance)
    ? maintenance.filter(m => m.status === 'scheduled')
    : []

  return (
    <div className="max-w-6xl">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900 tracking-tight">Dashboard</h1>
        <p className="text-sm text-slate-500 mt-1">System overview and active status</p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 lg:grid-cols-3 gap-5 mb-8">
        <StatCard
          title="Components"
          value={totalComponents}
          icon={Layers}
          iconClassName="bg-blue-50 text-blue-600"
          to="/admin/components"
        />
        <StatCard
          title="Active Incidents"
          value={activeIncidents.length}
          icon={AlertTriangle}
          iconClassName={activeIncidents.length > 0 ? 'bg-rose-50 text-rose-600' : 'bg-emerald-50 text-emerald-600'}
          to="/admin/incidents"
        />
        <StatCard
          title="Monitors"
          value={totalMonitors}
          icon={Activity}
          iconClassName="bg-indigo-50 text-indigo-600"
          to="/admin/monitors"
        />
        <StatCard
          title="Subscribers"
          value={totalSubscribers}
          icon={Users}
          iconClassName="bg-violet-50 text-violet-600"
          to="/admin/subscribers"
        />
        <StatCard
          title="Scheduled Maintenance"
          value={scheduledMaintenance.length}
          icon={Wrench}
          iconClassName="bg-amber-50 text-amber-600"
          to="/admin/maintenance"
        />
        <StatCard
          title="Total Incidents"
          value={totalIncidents}
          icon={TrendingUp}
          iconClassName="bg-slate-100 text-slate-600"
          to="/admin/incidents"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Component Status */}
        <div className="admin-surface flex flex-col">
          <div className="px-6 py-5 border-b border-slate-100 flex items-center justify-between">
            <h2 className="font-semibold text-slate-800">Component Status</h2>
            <Link to="/admin/components" className="text-xs font-medium text-blue-600 hover:text-blue-700">View all</Link>
          </div>
          <div className="p-2 flex-1">
            {(components || []).length === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-slate-500">No components yet.</div>
            ) : (
              <div className="space-y-1">
                {(components || []).slice(0, 8).map(c => (
                  <Link key={c.id} to={`/admin/components`} className="flex items-center justify-between px-4 py-3 rounded-lg hover:bg-slate-50 transition-colors group">
                    <span className="text-sm font-medium text-slate-700 group-hover:text-blue-600 transition-colors">{c.name}</span>
                    <span className={`flex items-center gap-2 text-xs font-medium`}>
                      <span className={`w-2 h-2 rounded-full ${STATUS_COLORS[c.status]} shadow-sm`} />
                      <span className="text-slate-600">{STATUS_LABELS[c.status]}</span>
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Recent Incidents */}
        <div className="admin-surface flex flex-col">
          <div className="px-6 py-5 border-b border-slate-100 flex items-center justify-between">
            <h2 className="font-semibold text-slate-800">Recent Incidents</h2>
            <Link to="/admin/incidents" className="text-xs font-medium text-blue-600 hover:text-blue-700">View all</Link>
          </div>
          <div className="p-2 flex-1">
            {(incidents || []).length === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-slate-500">No incidents.</div>
            ) : (
              <div className="space-y-1">
                {(incidents || []).slice(0, 5).map(inc => (
                  <Link key={inc.id} to={`/admin/incidents`} className="block px-4 py-3 rounded-lg hover:bg-slate-50 transition-colors group">
                    <div className="flex items-start justify-between gap-3">
                      <p className="text-sm font-medium text-slate-800 group-hover:text-blue-600 transition-colors line-clamp-1">{inc.title}</p>
                      <span className={`badge ${inc.status === 'resolved' ? 'badge-success' : 'badge-error'} flex-shrink-0`}>
                        {INCIDENT_STATUS_LABELS[inc.status]}
                      </span>
                    </div>
                    <p className="text-xs text-slate-500 mt-1.5">{formatDate(inc.createdAt)}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
