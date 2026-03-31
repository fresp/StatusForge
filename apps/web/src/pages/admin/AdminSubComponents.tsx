import React, { useState } from 'react'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import api from '../../lib/api'
import type { SubComponent, Component, ComponentStatus } from '../../types'
import { STATUS_LABELS, STATUS_COLORS } from '../../lib/utils'
import Modal from '../../components/Modal'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import { AdminListCard, AdminTableEmptyRow } from '../../components/AdminTableShell'

const STATUSES: ComponentStatus[] = ['operational', 'degraded_performance', 'partial_outage', 'major_outage', 'maintenance']

interface FormState {
  componentId: string
  name: string
  description: string
  status: ComponentStatus
}

const DEFAULT_FORM: FormState = { componentId: '', name: '', description: '', status: 'operational' }

export default function AdminSubComponents() {
  const { page, limit, apiParams, setPage, setLimit } = useAdminPagination()
  const { data: components, total: totalComponents, refetch: refetchComponents } = useApi<Component[]>('/components', [], { page: 1, limit: 10 })
  const {
    data: subComponents,
    total: totalSubComponents,
    totalPages,
    loading,
    refetch: refetchSubComponents,
  } = useApi<SubComponent[]>('/subcomponents', [], apiParams)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<SubComponent | null>(null)

  const [form, setForm] = useState<FormState>(DEFAULT_FORM)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const hasComponents = Boolean(components?.length)

  const isUnchanged =
    Boolean(editing) &&
    editing?.componentId === form.componentId &&
    editing?.name === form.name &&
    (editing?.description || '') === form.description &&
    editing?.status === form.status

  // Refetch both components and subcomponents
  const refetch = async () => {
    await Promise.all([refetchComponents(), refetchSubComponents()])
  }

  function openCreate() {
    setEditing(null)
    setForm({ ...DEFAULT_FORM, componentId: components?.[0]?.id || '' })
    setError('')
    setShowModal(true)
  }

  function openEdit(s: SubComponent) {
    setEditing(s)
    setForm({ componentId: s.componentId, name: s.name, description: s.description, status: s.status })
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
      if (!form.componentId) {
        setError('Parent component is required')
        return
      }

      if (editing) {
        await api.patch(`/subcomponents/${editing.id}`, form)
      } else {
        await api.post('/subcomponents', form)
      }
      await refetch()
      closeModal()
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(s: SubComponent) {
    if (!confirm(`Delete sub-component "${s.name}"?`)) return

    try {
      await api.delete(`/subcomponents/${s.id}`)
      await refetch()
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete')
    }
  }

  function getComponentName(id: string) {
    return components?.find(c => c.id === id)?.name || id
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Sub-Components</h1>
          <p className="text-sm text-gray-500 mt-1">{totalSubComponents} total · {totalComponents} components</p>
        </div>
        <button
          onClick={openCreate}
          disabled={!hasComponents}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" /> Add Sub-Component
        </button>
      </div>

      {!components?.length && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-3 text-sm text-yellow-700 mb-4">
          Create at least one component before adding sub-components.
        </div>
      )}

      <AdminListCard>
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Name</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Parent Component</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {(subComponents || []).map(s => (
              <tr key={s.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 font-medium text-gray-900">{s.name}</td>
                <td className="px-6 py-4 text-gray-500">{getComponentName(s.componentId)}</td>
                <td className="px-6 py-4">
                  <span className="flex items-center gap-1.5">
                    <span className={`w-2 h-2 rounded-full ${STATUS_COLORS[s.status]}`} />
                    {STATUS_LABELS[s.status]}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-end gap-2">
                    <button onClick={() => openEdit(s)} className="text-gray-400 hover:text-blue-600 transition-colors" aria-label={`Edit ${s.name}`}>
                      <Pencil className="w-4 h-4" />
                    </button>
                    <button onClick={() => handleDelete(s)} className="text-gray-400 hover:text-red-600 transition-colors" aria-label={`Delete ${s.name}`}>
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {(subComponents || []).length === 0 && (
              <AdminTableEmptyRow colSpan={4}>
                No sub-components yet.
              </AdminTableEmptyRow>
            )}
          </tbody>
        </table>

        <AdminPaginationControls
          page={page}
          totalPages={totalPages}
          total={totalSubComponents}
          limit={limit}
          loading={loading}
          onPageChange={setPage}
          onLimitChange={setLimit}
        />
      </AdminListCard>

      {showModal && (
        <Modal title={editing ? 'Edit Sub-Component' : 'New Sub-Component'} onClose={closeModal}>
          {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Parent Component</label>
              <select
                required
                value={form.componentId}
                onChange={e => setForm(f => ({ ...f, componentId: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select component...</option>
                {(components || []).map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            </div>
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
              <button type="submit" disabled={saving || (Boolean(editing) && isUnchanged)} className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Saving...' : editing ? 'Update' : 'Create'}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  )
}
