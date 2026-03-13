import React, { useState } from 'react'
import { Plus, Trash2, X, Activity, Edit2, CheckCircle2, XCircle, AlertCircle } from 'lucide-react'
import { useApi } from '../../hooks/useApi'
import api from '../../lib/api'
import type { Monitor, Component, SubComponent, MonitorType } from '../../types'
import { formatDate } from '../../lib/utils'

const MONITOR_TYPES: MonitorType[] = ['http', 'tcp', 'dns', 'ping']

interface FormState {
  name: string
  type: MonitorType
  target: string
  intervalSeconds: number
  timeoutSeconds: number
  componentId: string
  subComponentId: string
}


const DEFAULT_FORM: FormState = {
  name: '',
  type: 'http',
  target: '',
  intervalSeconds: 60,
  timeoutSeconds: 10,
  componentId: '',
  subComponentId: '',
}


function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold text-gray-900">{title}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600"><X className="w-5 h-5" /></button>
        </div>
        <div className="p-6">{children}</div>
      </div>
    </div>
  )
}

const TYPE_PLACEHOLDERS: Record<MonitorType, string> = {
  http: 'https://example.com/health',
  tcp: 'example.com:443',
  dns: 'example.com',
  ping: 'example.com',
}

export default function AdminMonitors() {
  const { data: monitors, refetch } = useApi<Monitor[]>('/monitors')
  const { data: components } = useApi<Component[]>('/components')
  const { data: subcomponents = [] } = useApi<SubComponent[]>('/subcomponents')
  const [showModal, setShowModal] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(DEFAULT_FORM)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{status: string, responseTime: number} | null>(null)
  const [error, setError] = useState('')
  
  function openCreate() {
    setEditingId(null)
    setForm({ ...DEFAULT_FORM, componentId: components?.[0]?.id || '', subComponentId: '' })
    setError('')
    setTestResult(null)
    setShowModal(true)
  }

  function openEdit(m: Monitor) {
    setEditingId(m.id || null)
    setForm({
      name: m.name,
      type: m.type,
      target: m.target,
      intervalSeconds: m.intervalSeconds,
      timeoutSeconds: m.timeoutSeconds,
      componentId: m.componentId || '',
      subComponentId: m.subComponentId || '',
    })
    setError('')
    setTestResult(null)
    setShowModal(true)
  }

  function closeModal() {
    setShowModal(false)
    setEditingId(null)
    setTestResult(null)
  }

  async function handleTest() {
    setTesting(true)
    setTestResult(null)
    setError('')
    try {
      const res = await api.post('/monitors/test', form)
      setTestResult(res.data)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to test monitor')
    } finally {
      setTesting(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      if (editingId) {
        await api.put(`/monitors/${editingId}`, form)
      } else {
        await api.post('/monitors', form)
      }
      await refetch()
      closeModal()
    } catch (err: any) {
      setError(err.response?.data?.error || `Failed to ${editingId ? 'update' : 'create'} monitor`)
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(m: Monitor) {
    if (!confirm(`Delete monitor "${m.name}"?`)) return
    try {
      await api.delete(`/monitors/${m.id}`)
      await refetch()
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete')
    }
  }

  function getComponentNameAndSubcomponentText(monitor: Monitor) {
    if (monitor.subComponentId) {
      // Monitor is associated with a subcomponent
      const subcomponent = (subcomponents || []).find(sc => sc.id === monitor.subComponentId);
      const component = (components || []).find(c => c.id === monitor.componentId);
      if (subcomponent && component) {
        return `${subcomponent.name} (Subcomponent of ${component.name})`; 
      } else if (subcomponent) {
        return `${subcomponent.name} (SubComponent)`; 
      } else {
        return `SubComponent: ${monitor.subComponentId}`; // Not found in the list
      }
    } else if (monitor.componentId) {
      // Traditional component-only case
      return (components || []).find(c => c.id === monitor.componentId)?.name || monitor.componentId;
    }
    return 'Unassigned';
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Monitors</h1>
          <p className="text-sm text-gray-500 mt-1">{monitors?.length ?? 0} active monitors</p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus className="w-4 h-4" /> Add Monitor
        </button>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-100">
            <tr>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Name</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Type</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Target</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Component</th>
              <th className="text-left px-6 py-3 font-medium text-gray-600">Interval</th>
              <th className="px-6 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {(monitors || []).map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 font-medium text-gray-900">{m.name}</td>
                <td className="px-6 py-4">
                  <span className="flex items-center gap-1.5">
                    <Activity className="w-3.5 h-3.5 text-purple-500" />
                    <span className="uppercase text-xs font-medium text-purple-700">{m.type}</span>
                  </span>
                </td>
                <td className="px-6 py-4 text-gray-500 max-w-xs truncate font-mono text-xs">{m.target}</td>
                <td className="px-6 py-4">
                  {m.lastStatus === 'up' ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-green-50 text-green-700 border border-green-200">
                      <CheckCircle2 className="w-3.5 h-3.5" />
                      Up
                    </span>
                  ) : m.lastStatus === 'down' ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-red-50 text-red-700 border border-red-200">
                      <XCircle className="w-3.5 h-3.5" />
                      Down
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-gray-50 text-gray-600 border border-gray-200">
                      <AlertCircle className="w-3.5 h-3.5" />
                      Pending
                    </span>
                  )}
                </td>
                <td className="px-6 py-4 text-gray-500">{getComponentNameAndSubcomponentText(m)}</td>
                <td className="px-6 py-4 text-gray-500">{m.intervalSeconds}s</td>
                <td className="px-6 py-4">
                  <div className="flex items-center justify-end gap-2">
                    <button onClick={() => openEdit(m)} className="text-gray-400 hover:text-blue-600 transition-colors">
                      <Edit2 className="w-4 h-4" />
                    </button>
                    <button onClick={() => handleDelete(m)} className="text-gray-400 hover:text-red-600 transition-colors">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {(monitors || []).length === 0 && (
              <tr>
                <td colSpan={7} className="px-6 py-12 text-center text-gray-400">No monitors configured. Add one to start tracking uptime.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {showModal && (
        <Modal title={editingId ? "Edit Monitor" : "New Monitor"} onClose={closeModal}>
          {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                required
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="API Health Check"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
              <select
                value={form.type}
                onChange={e => setForm(f => ({ ...f, type: e.target.value as MonitorType, target: '' }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {MONITOR_TYPES.map(t => (
                  <option key={t} value={t}>{t.toUpperCase()}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Target</label>
              <input
                required
                value={form.target}
                onChange={e => setForm(f => ({ ...f, target: e.target.value }))}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                placeholder={TYPE_PLACEHOLDERS[form.type]}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Component / Subcomponent</label>
              <select
                value={form.componentId}
                onChange={e => setForm(f => ({ ...f, componentId: e.target.value, subComponentId: '' }))} // Clear subComponentId when component changes
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 mb-2"
              >
                {(components || []).map(c => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
              
              {/* Subcomponent selection */}
              {form.componentId && (
                <select
                  value={form.subComponentId}
                  onChange={e => setForm(f => ({ ...f, subComponentId: e.target.value }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">Select subcomponent (optional)...</option>
                  {(subcomponents || [])
                    .filter(sc => sc.componentId === form.componentId)
                    .map(sc => (
                      <option key={sc.id} value={sc.id}>{sc.name}</option>
                    ))
                  }
                </select>
              )}
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Interval (seconds)</label>
                <input
                  type="number"
                  min={10}
                  max={3600}
                  value={form.intervalSeconds}
                  onChange={e => setForm(f => ({ ...f, intervalSeconds: parseInt(e.target.value) }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Timeout (seconds)</label>
                <input
                  type="number"
                  min={1}
                  max={60}
                  value={form.timeoutSeconds}
                  onChange={e => setForm(f => ({ ...f, timeoutSeconds: parseInt(e.target.value) }))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-3 pt-2">
              <button type="button" onClick={closeModal} className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50">
                Cancel
              </button>
              <button type="button" onClick={handleTest} disabled={testing || !form.target} className="flex-1 border border-blue-200 text-blue-700 bg-blue-50 hover:bg-blue-100 disabled:opacity-60 rounded-lg py-2 text-sm font-medium">
                {testing ? 'Testing...' : 'Test Target'}
              </button>
              <button type="submit" disabled={saving} className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg py-2 text-sm font-medium">
                {saving ? 'Saving...' : (editingId ? 'Update Monitor' : 'Create Monitor')}
              </button>
            </div>
            {testResult && (
              <div className={`px-3 py-2 rounded-lg text-sm flex items-center justify-between ${testResult.status === 'up' ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'}`}>
                <span className="font-medium">Test Result: {testResult.status.toUpperCase()}</span>
                <span>{testResult.responseTime}ms</span>
              </div>
            )}
          </form>
        </Modal>
      )}
    </div>
  )
}
