import React, { useState } from 'react'
import { Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import api from '../../lib/api'
import type { Incident, IncidentUpdate, Component, IncidentStatus, IncidentImpact, SubComponent } from '../../types'
import { INCIDENT_STATUS_LABELS, INCIDENT_IMPACT_LABELS, formatDate } from '../../lib/utils'
import Modal from '../../components/Modal'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import { AdminListCard, AdminTableEmptyRow, textOrEmDash } from '../../components/AdminTableShell'

const STATUSES: IncidentStatus[] = ['investigating', 'identified', 'monitoring', 'resolved']
const IMPACTS: IncidentImpact[] = ['none', 'minor', 'major', 'critical']

interface IncidentForm {
  title: string
  description: string
  status: IncidentStatus
  impact: IncidentImpact
  affectedComponentTargets: Array<{
    componentId: string
    subComponentIds: string[]
  }>
}

const DEFAULT_FORM: IncidentForm = {
  title: '',
  description: '',
  status: 'investigating',
  impact: 'minor',
  affectedComponentTargets: [],
}

function IncidentRow({ incident, components, onRefetch }: {
  incident: Incident
  components: Component[]
  onRefetch: () => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [showUpdateModal, setShowUpdateModal] = useState(false)
  const [updateMsg, setUpdateMsg] = useState('')
  const [updateStatus, setUpdateStatus] = useState<IncidentStatus>('investigating')
  const [updates, setUpdates] = useState<IncidentUpdate[] | null>(null)
  const [saving, setSaving] = useState(false)

  async function loadUpdates() {
    if (!expanded) {
      const res = await api.get(`/incidents/${incident.id}/updates`)
      setUpdates(res.data)
    }
    setExpanded(e => !e)
  }

  async function submitUpdate(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    try {
      await api.post(`/incidents/${incident.id}/update`, { message: updateMsg, status: updateStatus })
      setShowUpdateModal(false)
      setUpdateMsg('')
      onRefetch()
      const res = await api.get(`/incidents/${incident.id}/updates`)
      setUpdates(res.data)
    } catch {
      alert('Failed to add update')
    } finally {
      setSaving(false)
    }
  }

  const statusColor = incident.status === 'resolved'
    ? 'bg-green-100 text-green-700'
    : incident.status === 'monitoring'
      ? 'bg-blue-100 text-blue-700'
      : 'bg-red-100 text-red-700'

  const impactColor: Record<IncidentImpact, string> = {
    none: 'bg-gray-100 text-gray-600',
    minor: 'bg-yellow-100 text-yellow-700',
    major: 'bg-orange-100 text-orange-700',
    critical: 'bg-red-100 text-red-700',
  }

  return (
    <>
      <tr className="hover:bg-gray-50">
        <td className="px-6 py-4 font-medium text-gray-900 max-w-xs truncate">{incident.title}</td>
        <td className="px-6 py-4">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${statusColor}`}>
            {INCIDENT_STATUS_LABELS[incident.status]}
          </span>
        </td>
        <td className="px-6 py-4">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${impactColor[incident.impact]}`}>
            {INCIDENT_IMPACT_LABELS[incident.impact]}
          </span>
        </td>
        <td className="px-6 py-4 text-sm text-gray-500">{formatDate(incident.createdAt)}</td>
         <td className="px-6 py-4 text-sm text-gray-500">{textOrEmDash(incident.creatorUsername)}</td>
        <td className="px-6 py-4">
          <div className="flex items-center justify-end gap-2">
            <button
              onClick={() => setShowUpdateModal(true)}
              className="text-xs text-blue-600 hover:underline"
            >
              Add Update
            </button>
            <button onClick={loadUpdates} className="text-gray-400 hover:text-gray-600">
              {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
            </button>
          </div>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={6} className="px-6 pb-4 bg-gray-50">
            <div className="pl-4 border-l-2 border-gray-200 space-y-2 mt-1">
              {(updates || []).length === 0 ? (
                <p className="text-sm text-gray-400">No updates yet.</p>
              ) : (
                (updates || []).map(u => (
                  <div key={u.id} className="text-sm">
                    <span className="font-medium text-gray-700">{INCIDENT_STATUS_LABELS[u.status]}</span>
                    <span className="text-gray-500 ml-2">{u.message}</span>
                    <span className="text-gray-400 ml-2 text-xs">{formatDate(u.createdAt)}</span>
                  </div>
                ))
              )}
            </div>
          </td>
        </tr>
      )}

      {showUpdateModal && (
        <Modal title="Add Incident Update" onClose={() => setShowUpdateModal(false)} size="lg">
          <form onSubmit={submitUpdate} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
              <select
                value={updateStatus}
                onChange={e => setUpdateStatus(e.target.value as IncidentStatus)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {STATUSES.map(s => <option key={s} value={s}>{INCIDENT_STATUS_LABELS[s]}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Message</label>
              <textarea
                required
                rows={3}
                value={updateMsg}
                onChange={e => setUpdateMsg(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Describe what's happening..."
              />
            </div>
            <div className="flex gap-3">
              <button type="button" onClick={() => setShowUpdateModal(false)} className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50">
                Cancel
              </button>
              <button type="submit" disabled={saving} className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Posting...' : 'Post Update'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  )
}

export default function AdminIncidents() {
  const { page, limit, apiParams, setPage, setLimit } = useAdminPagination()
  const { data: incidents, total: totalIncidents, totalPages, loading, refetch } = useApi<Incident[]>('/incidents', [], apiParams)
  const { data: components } = useApi<Component[]>('/components', [], { page: 1, limit: 10 })
  const { data: subComponents } = useApi<SubComponent[]>('/subcomponents', [], { page: 1, limit: 10 })
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState<IncidentForm>(DEFAULT_FORM)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState<'all' | 'active' | 'resolved'>('all')

  function openCreate() {
    setForm(DEFAULT_FORM)
    setError('')
    setShowModal(true)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.post('/incidents', form)
      await refetch()
      setShowModal(false)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create incident')
    } finally {
      setSaving(false)
    }
  }

  function toggleComponent(componentId: string) {
    setForm(f => ({
      ...f,
      affectedComponentTargets: f.affectedComponentTargets.some(t => t.componentId === componentId)
        ? f.affectedComponentTargets.filter(t => t.componentId !== componentId)
        : [...f.affectedComponentTargets, { componentId, subComponentIds: [] }],
    }))
  }

  function toggleSubComponent(componentId: string, subComponentId: string) {
    setForm((current) => {
      const index = current.affectedComponentTargets.findIndex(target => target.componentId === componentId)
      if (index === -1) {
        return {
          ...current,
          affectedComponentTargets: [
            ...current.affectedComponentTargets,
            { componentId, subComponentIds: [subComponentId] },
          ],
        }
      }

      const target = current.affectedComponentTargets[index]
      const hasSub = target.subComponentIds.includes(subComponentId)
      const nextSubComponentIds = hasSub
        ? target.subComponentIds.filter(id => id !== subComponentId)
        : [...target.subComponentIds, subComponentId]

      const nextTargets = [...current.affectedComponentTargets]
      nextTargets[index] = {
        ...target,
        subComponentIds: nextSubComponentIds,
      }

      return {
        ...current,
        affectedComponentTargets: nextTargets,
      }
    })
  }

  function getTarget(componentId: string) {
    return form.affectedComponentTargets.find(t => t.componentId === componentId)
  }

  const filtered = (incidents || []).filter(i => {
    if (filter === 'active') return i.status !== 'resolved'
    if (filter === 'resolved') return i.status === 'resolved'
    return true
  })

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Incidents</h1>
          <p className="text-sm text-gray-500 mt-1">{totalIncidents} total</p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" /> Create Incident
        </button>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-2 mb-4">
        {(['all', 'active', 'resolved'] as const).map(f => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-3 py-1.5 rounded-lg text-sm font-medium capitalize transition-colors ${filter === f ? 'bg-gray-900 text-white' : 'bg-white border border-gray-200 text-gray-600 hover:bg-gray-50'
              }`}
          >
            {f}
          </button>
        ))}
      </div>

       <AdminListCard>
         <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Title</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Impact</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Created</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Creator</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {filtered.map(inc => (
              <IncidentRow key={inc.id} incident={inc} components={components || []} onRefetch={refetch} />
            ))}
             {filtered.length === 0 && (
               <AdminTableEmptyRow colSpan={6}>
                 No incidents found.
               </AdminTableEmptyRow>
             )}
          </tbody>
        </table>

        <AdminPaginationControls
          page={page}
          totalPages={totalPages}
          total={totalIncidents}
          limit={limit}
          loading={loading}
          onPageChange={setPage}
           onLimitChange={setLimit}
         />
       </AdminListCard>

      {showModal && (
        <Modal title="Create Incident" onClose={() => setShowModal(false)} size="lg">
          {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
              <input
                required
                value={form.title}
                onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Brief incident title"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <textarea
                required
                rows={3}
                value={form.description}
                onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="What is happening?"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <select
                  value={form.status}
                  onChange={e => setForm(f => ({ ...f, status: e.target.value as IncidentStatus }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {STATUSES.map(s => <option key={s} value={s}>{INCIDENT_STATUS_LABELS[s]}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Impact</label>
                <select
                  value={form.impact}
                  onChange={e => setForm(f => ({ ...f, impact: e.target.value as IncidentImpact }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {IMPACTS.map(i => <option key={i} value={i}>{INCIDENT_IMPACT_LABELS[i]}</option>)}
                </select>
              </div>
            </div>
            {(components || []).length > 0 && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Affected Components</label>
                <div className="space-y-1">
                  {(components || []).map((component) => {
                    const target = getTarget(component.id)
                    const checked = Boolean(target)
                    const relatedSubComponents = (subComponents || []).filter(
                      (subComponent) => subComponent.componentId === component.id,
                    )

                    return (
                      <div key={component.id} className="rounded-lg border border-gray-200 px-3 py-2">
                        <label className="flex items-center gap-2 text-sm cursor-pointer font-medium text-gray-800">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => toggleComponent(component.id)}
                            className="rounded"
                          />
                          {component.name}
                        </label>

                        {checked && relatedSubComponents.length > 0 && (
                          <div className="mt-2 pl-6 space-y-1">
                            {relatedSubComponents.map((subComponent) => {
                              const isSubChecked = target?.subComponentIds.includes(subComponent.id) || false
                              return (
                                <label key={subComponent.id} className="flex items-center gap-2 text-xs text-gray-600 cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={isSubChecked}
                                    onChange={() => toggleSubComponent(component.id, subComponent.id)}
                                    className="rounded"
                                  />
                                  {subComponent.name}
                                </label>
                              )
                            })}
                            <p className="text-[11px] text-gray-500 pt-1">
                              Leave sub-components unchecked to affect the whole component.
                            </p>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
            <div className="flex gap-3 pt-2">
              <button type="button" onClick={() => setShowModal(false)} className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50">
                Cancel
              </button>
              <button type="submit" disabled={saving} className="flex-1 bg-red-600 hover:bg-red-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Creating...' : 'Create Incident'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
