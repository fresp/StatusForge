import React from 'react'

interface AdminListCardProps {
  children: React.ReactNode
}

/**
 * Shared admin card wrapper for list/table content.
 *
 * Preserves the existing white card + border + overflow clipping
 * used across admin list pages.
 */
export function AdminListCard({ children }: AdminListCardProps) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      {children}
    </div>
  )
}

interface AdminTableEmptyRowProps {
  colSpan: number
  children: React.ReactNode
}

/**
 * Standard empty-state row for admin tables.
 *
 * Keeps existing padding, alignment, and muted text styling
 * while letting callers control the message and colSpan.
 */
export function AdminTableEmptyRow({ colSpan, children }: AdminTableEmptyRowProps) {
  return (
    <tr>
      <td colSpan={colSpan} className="px-6 py-12 text-center text-gray-400">
        {children}
      </td>
    </tr>
  )
}

interface AdminListStateMessageProps {
  tone?: 'default' | 'error'
  children: React.ReactNode
}

/**
 * Lightweight helper for inline loading/error messaging inside
 * admin list containers.
 */
export function AdminListStateMessage({ tone = 'default', children }: AdminListStateMessageProps) {
  const colorClass = tone === 'error' ? 'text-red-600' : 'text-gray-500'

  return (
    <div className={`px-6 py-8 text-sm ${colorClass}`}>
      {children}
    </div>
  )
}

/**
 * Render user-facing text or a typographic em dash when the
 * value is empty, null, or only whitespace.
 */
export function textOrEmDash(value?: string | null): string {
  if (typeof value !== 'string') return '—'
  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : '—'
}
