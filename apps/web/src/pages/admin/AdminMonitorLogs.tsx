import React from 'react'
import { Link, useParams } from 'react-router-dom'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import { AdminListCard, AdminListStateMessage } from '../../components/AdminTableShell'
import type { Monitor, MonitorLog } from '../../types'
import { formatDate } from '../../lib/utils'

export function statusBadgeClass(status: string): string {
  if (status === 'up') {
    return 'bg-green-50 text-green-700 border border-green-200'
  }

  if (status === 'down') {
    return 'bg-red-50 text-red-700 border border-red-200'
  }

  return 'bg-gray-50 text-gray-700 border border-gray-200'
}

export function latencyBarWidth(responseTime: number): number {
  const max = 2000
  const safe = Math.max(0, Math.min(responseTime, max))
  return Math.round((safe / max) * 100)
}

export default function AdminMonitorLogs() {
  const { id } = useParams<{ id: string }>()
  const { page, limit, apiParams, setPage, setLimit } = useAdminPagination()

  const {
    data: monitors,
    loading: monitorsLoading,
  } = useApi<Monitor[]>('/monitors', [], { page: 1, limit: 10 })

  const {
    data: logs,
    total,
    totalPages,
    loading,
    error,
  } = useApi<MonitorLog[]>(`/monitors/${id}/logs`, [id], apiParams)

  const monitor = (monitors || []).find((item) => item.id === id)

  if (!id) {
    return (
      <div className="p-8">
        <p className="text-sm text-red-600">Invalid monitor id.</p>
      </div>
    )
  }

  return (
    <div className="p-8">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Monitor Logs</h1>
          <p className="mt-1 text-sm text-gray-500">
            {monitorsLoading
              ? 'Loading monitor details...'
              : monitor
                ? `${monitor.name} (${monitor.type.toUpperCase()})`
                : `Monitor ${id}`}
          </p>
        </div>
        <Link
          to="/admin/monitors"
          className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Back to Monitors
        </Link>
      </div>

      <AdminListCard>
        {loading && (
          <AdminListStateMessage>
            Loading logs...
          </AdminListStateMessage>
        )}

        {!loading && error && (
          <AdminListStateMessage tone="error">
            Failed to load logs.
          </AdminListStateMessage>
        )}

        {!loading && !error && (logs || []).length === 0 && (
          <div className="px-6 py-12 text-center text-sm text-gray-500">
            No logs yet for this monitor.
          </div>
        )}

        {!loading && !error && (logs || []).length > 0 && (
          <>
            <table className="w-full text-sm">
              <thead className="border-b border-gray-100 bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left font-medium text-gray-600">Status</th>
                  <th className="px-6 py-3 text-left font-medium text-gray-600">Latency</th>
                  <th className="px-6 py-3 text-left font-medium text-gray-600">Status Code</th>
                  <th className="px-6 py-3 text-left font-medium text-gray-600">Region</th>
                  <th className="px-6 py-3 text-left font-medium text-gray-600">Checked At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {(logs || []).map((log) => (
                  <tr key={log.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${statusBadgeClass(log.status)}`}
                      >
                        {log.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <span className="w-20 text-xs text-gray-700">{log.responseTime}ms</span>
                        <div className="h-2 w-32 rounded bg-gray-100">
                          <div
                            className="h-2 rounded bg-blue-500"
                            style={{ width: `${latencyBarWidth(log.responseTime)}%` }}
                          />
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-gray-700">{log.statusCode || '-'}</td>
                    <td className="px-6 py-4 text-gray-500">{log.region || 'global'}</td>
                    <td className="px-6 py-4 text-gray-500">{formatDate(log.checkedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            <AdminPaginationControls
              page={page}
              totalPages={totalPages}
              total={total}
              limit={limit}
              loading={loading}
              onPageChange={setPage}
              onLimitChange={setLimit}
            />
          </>
        )}
      </AdminListCard>
    </div>
  )
}
