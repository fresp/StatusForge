import React, { useState } from 'react'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import api from '../../lib/api'
import type { Component, ComponentStatus } from '../../types'
import { STATUS_LABELS, STATUS_COLORS } from '../../lib/utils'
import Modal from '../../components/Modal'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import { AdminListCard, AdminTableEmptyRow, textOrEmDash } from '../../components/AdminTableShell'

const STATUSES: ComponentStatus[] = ['operational', 'degraded_performance', 'partial_outage', 'major_outage', 'maintenance']

interface FormState {
  name: string
  description: string
  status: ComponentStatus
}

const DEFAULT_FORM: FormState = { name: '', description: '', status: 'operational' }

export default function AdminComponents() {
  const { page, limit, apiParams, setPage, setLimit } = useAdminPagination()
  const { data: components, total: totalComponents, totalPages, loading, refetch } = useApi<Component[]>('/components', [], apiParams)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<Component | null>(null)
  const [form, setForm] = useState<FormState>(DEFAULT_FORM)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  function openCreate() {
    setEditing(null)
    setForm(DEFAULT_FORM)
    setError('')
    setShowModal(true)
  }

  function openEdit(c: Component) {
    setEditing(c)
    setForm({ name: c.name, description: c.description, status: c.status })
    setError('')
    setShowModal(true)
  }

  function closeModal() {
    setShowModal(false)
    setEditing(null)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      if (editing) {
        await api.patch(`/components/${editing.id}`, form)
      } else {
        await api.post('/components', form)
      }
      await refetch()
      closeModal()
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(c: Component) {
    if (!confirm(`Delete component "${c.name}"?`)) return
    try {
      await api.delete(`/components/${c.id}`)
      await refetch()
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete')
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Components</h1>
          <p className="text-sm text-gray-500 mt-1">{totalComponents} total</p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" /> Add Component
        </button>
      </div>

      <AdminListCard>
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Name</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Description</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {(components || []).map(c => (
              <tr key={c.id} className="hover:bg-gray-50">
                 <td className="px-6 py-4 font-medium text-gray-900">{c.name}</td>
                 <td className="px-6 py-4 text-gray-500">{textOrEmDash(c.description)}</td>
                <td className="px-6 py-4">
                  <span className="flex items-center gap-1.5">
                    <span className={`w-2 h-2 rounded-full ${STATUS_COLORS[c.status]}`} />
                    {STATUS_LABELS[c.status]}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-end gap-2">
                    <button onClick={() => openEdit(c)} className="text-gray-400 hover:text-blue-600 transition-colors">
                      <Pencil className="w-4 h-4" />
                    </button>
                    <button onClick={() => handleDelete(c)} className="text-gray-400 hover:text-red-600 transition-colors">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
             {(components || []).length === 0 && (
               <AdminTableEmptyRow colSpan={4}>
                 No components yet. Create one to get started.
               </AdminTableEmptyRow>
             )}
          </tbody>
        </table>

        <AdminPaginationControls
          page={page}
          totalPages={totalPages}
          total={totalComponents}
          limit={limit}
          loading={loading}
          onPageChange={setPage}
           onLimitChange={setLimit}
         />
       </AdminListCard>

      {showModal && (
        <Modal title={editing ? 'Edit Component' : 'New Component'} onClose={closeModal}>
          {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                required
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <input
                value={form.description}
                onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
              <select
                value={form.status}
                onChange={e => setForm(f => ({ ...f, status: e.target.value as ComponentStatus }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {STATUSES.map(s => (
                  <option key={s} value={s}>{STATUS_LABELS[s]}</option>
                ))}
              </select>
            </div>
            <div className="flex gap-3 pt-2">
              <button type="button" onClick={closeModal} className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50">
                Cancel
              </button>
              <button type="submit" disabled={saving} className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Saving...' : editing ? 'Update' : 'Create'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
