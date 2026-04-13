import React from 'react'

interface AdminPaginationControlsProps {
  page: number
  totalPages: number
  total: number
  limit: number
  loading?: boolean
  onPageChange: (page: number) => void
  onLimitChange: (limit: number) => void
  pageSizeOptions?: number[]
}

const DEFAULT_PAGE_SIZES = [10, 20, 50, 100]

function buildPageButtons(page: number, totalPages: number) {
  const pages = new Set<number>()
  pages.add(1)
  pages.add(totalPages)

  for (let current = page - 1; current <= page + 1; current += 1) {
    if (current >= 1 && current <= totalPages) {
      pages.add(current)
    }
  }

  return Array.from(pages).sort((a, b) => a - b)
}

export default function AdminPaginationControls({
  page,
  totalPages,
  total,
  limit,
  loading = false,
  onPageChange,
  onLimitChange,
  pageSizeOptions = DEFAULT_PAGE_SIZES,
}: AdminPaginationControlsProps) {
  if (total <= 0) {
    return null
  }

  const safeTotalPages = Math.max(1, totalPages)
  if (safeTotalPages <= 1) {
    return null
  }

  const safePage = Math.min(Math.max(page, 1), safeTotalPages)
  const canGoPrev = safePage > 1 && !loading
  const canGoNext = safePage < safeTotalPages && !loading

  const start = (safePage - 1) * limit + 1
  const end = Math.min(total, safePage * limit)
  const buttons = buildPageButtons(safePage, safeTotalPages)

  return (
    <div className="flex flex-col gap-3 border-t border-slate-200/80 bg-slate-50/70 px-6 py-4 md:flex-row md:items-center md:justify-between">
      <div className="text-xs text-slate-500">
        Showing <span className="font-semibold text-slate-700">{start}</span>-
        <span className="font-semibold text-slate-700">{end}</span> of{' '}
        <span className="font-semibold text-slate-700">{total}</span>
        {loading && <span className="ml-2 text-slate-400">Loading...</span>}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <label className="text-xs font-medium text-slate-500" htmlFor="admin-page-size">
          Rows
        </label>
        <select
          id="admin-page-size"
          value={limit}
          disabled={loading}
          onChange={(event) => onLimitChange(Number.parseInt(event.target.value, 10))}
          className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs text-slate-700 shadow-sm disabled:opacity-60"
        >
          {pageSizeOptions.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>

        <button
          type="button"
          onClick={() => onPageChange(safePage - 1)}
          disabled={!canGoPrev}
          className="rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Previous
        </button>

        <div className="flex items-center gap-1">
          {buttons.map((buttonPage, index) => {
            const previous = buttons[index - 1]
            const showGap = previous && buttonPage - previous > 1

            return (
              <React.Fragment key={buttonPage}>
                {showGap && <span className="px-1 text-xs text-slate-400">…</span>}
                <button
                  type="button"
                  onClick={() => onPageChange(buttonPage)}
                  disabled={loading || buttonPage === safePage}
                  className={`rounded-full px-3 py-1.5 text-xs font-medium shadow-sm transition-colors ${buttonPage === safePage
                      ? 'border border-blue-500/20 bg-gradient-to-b from-blue-500 to-blue-600 text-white'
                      : 'border border-slate-200 bg-white text-slate-700 hover:bg-slate-100'
                     } disabled:cursor-not-allowed disabled:opacity-70`}
                >
                  {buttonPage}
                </button>
              </React.Fragment>
            )
          })}
        </div>

        <button
          type="button"
          onClick={() => onPageChange(safePage + 1)}
          disabled={!canGoNext}
          className="rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  )
}
