import React, { useState } from 'react'
import { Plus, Pencil } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import api from '../../lib/api'
import type { Maintenance, Component, MaintenanceStatus } from '../../types'
import { formatDate } from '../../lib/utils'
import Modal from '../../components/Modal'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import { AdminListCard, AdminTableEmptyRow, textOrEmDash } from '../../components/AdminTableShell'

const STATUSES: MaintenanceStatus[] = ['scheduled', 'in_progress', 'completed']

const STATUS_LABELS: Record<MaintenanceStatus, string> = {
  scheduled: 'Scheduled',
  in_progress: 'In Progress',
  completed: 'Completed',
}

const STATUS_COLORS: Record<MaintenanceStatus, string> = {
  scheduled: 'bg-blue-100 text-blue-700',
  in_progress: 'bg-yellow-100 text-yellow-700',
  completed: 'bg-green-100 text-green-700',
}

interface FormState {
  title: string
  description: string
  components: string[]
  startTime: string
  endTime: string
  status: MaintenanceStatus
}

const DEFAULT_FORM: FormState = {
  title: '',
  description: '',
  components: [],
  startTime: '',
  endTime: '',
  status: 'scheduled',
}

function toDatetimeLocal(iso: string) {
  if (!iso) return ''
  return iso.slice(0, 16)
}

export default function AdminMaintenance() {
  const { page, limit, apiParams, setPage, setLimit } = useAdminPagination()
  const { data: maintenance, total: totalMaintenance, totalPages, loading, refetch } = useApi<Maintenance[]>('/maintenance', [], apiParams)
  const { data: components } = useApi<Component[]>('/components', [], { page: 1, limit: 10 })
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Maintenance | null>(null)
  const [form, setForm] = useState<FormState>(DEFAULT_FORM)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  function openCreate() {
    setEditing(null)
    setForm(DEFAULT_FORM)
    setError('')
    setShowModal(true)
  }

  function openEdit(m: Maintenance) {
    setEditing(m)
    setForm({
      title: m.title,
      description: m.description,
      components: m.components,
      startTime: toDatetimeLocal(m.startTime),
      endTime: toDatetimeLocal(m.endTime),
      status: m.status,
    })
    setError('')
    setShowModal(true)
  }

  function closeModal() {
    setShowModal(false)
    setEditing(null)
  }

  function toggleComponent(id: string) {
    setForm(f => ({
      ...f,
      components: f.components.includes(id)
        ? f.components.filter(c => c !== id)
        : [...f.components, id],
    }))
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      const payload = {
        ...form,
        startTime: new Date(form.startTime).toISOString(),
        endTime: new Date(form.endTime).toISOString(),
      }
      if (editing) {
        await api.patch(`/maintenance/${editing.id}`, payload)
      } else {
        await api.post('/maintenance', payload)
      }
      await refetch()
      closeModal()
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Maintenance</h1>
          <p className="text-sm text-gray-500 mt-1">{totalMaintenance} total</p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" /> Schedule Maintenance
        </button>
      </div>

      <AdminListCard>
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Title</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Start</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">End</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Creator</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {(maintenance || []).map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 font-medium text-gray-900">{m.title}</td>
                <td className="px-6 py-4">
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_COLORS[m.status]}`}>
                    {STATUS_LABELS[m.status]}
                  </span>
                </td>
                <td className="px-6 py-4 text-gray-500">{formatDate(m.startTime)}</td>
                 <td className="px-6 py-4 text-gray-500">{formatDate(m.endTime)}</td>
                 <td className="px-6 py-4 text-sm text-gray-500">{textOrEmDash(m.creatorUsername)}</td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-end">
                    <button onClick={() => openEdit(m)} className="text-gray-400 hover:text-blue-600 transition-colors">
                      <Pencil className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
             {(maintenance || []).length === 0 && (
               <AdminTableEmptyRow colSpan={6}>
                 No maintenance windows scheduled.
               </AdminTableEmptyRow>
             )}
          </tbody>
        </table>

        <AdminPaginationControls
          page={page}
          totalPages={totalPages}
          total={totalMaintenance}
          limit={limit}
          loading={loading}
          onPageChange={setPage}
           onLimitChange={setLimit}
         />
       </AdminListCard>

      {showModal && (
        <Modal title={editing ? 'Edit Maintenance' : 'Schedule Maintenance'} onClose={closeModal} size="lg">
          {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
              <input
                required
                value={form.title}
                onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <textarea
                rows={2}
                value={form.description}
                onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Start Time</label>
                <input
                  type="datetime-local"
                  required
                  value={form.startTime}
                  onChange={e => setForm(f => ({ ...f, startTime: e.target.value }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">End Time</label>
                <input
                  type="datetime-local"
                  required
                  value={form.endTime}
                  onChange={e => setForm(f => ({ ...f, endTime: e.target.value }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
              <select
                value={form.status}
                onChange={e => setForm(f => ({ ...f, status: e.target.value as MaintenanceStatus }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {STATUSES.map(s => <option key={s} value={s}>{STATUS_LABELS[s]}</option>)}
              </select>
            </div>
            {(components || []).length > 0 && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Affected Components</label>
                <div className="space-y-1">
                  {(components || []).map(c => (
                    <label key={c.id} className="flex items-center gap-2 text-sm cursor-pointer">
                      <input
                        type="checkbox"
                        checked={form.components.includes(c.id)}
                        onChange={() => toggleComponent(c.id)}
                        className="rounded"
                      />
                      {c.name}
                    </label>
                  ))}
                </div>
              </div>
            )}
            <div className="flex gap-3 pt-2">
              <button type="button" onClick={closeModal} className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50">
                Cancel
              </button>
              <button type="submit" disabled={saving} className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Saving...' : editing ? 'Update' : 'Schedule'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
