import { Link } from 'react-router-dom'
import { Activity, History } from 'lucide-react'

type Props = {
  siteName: string
  logoUrl?: string
  statusLabel: string
  activeView?: 'dashboard' | 'history'
}

export default function StatusTopNav({ siteName, logoUrl, statusLabel, activeView = 'dashboard' }: Props) {
  const dashboardActive = activeView === 'dashboard'
  const historyActive = activeView === 'history'
  const mobileDestination = historyActive
    ? { to: '/', label: 'Open dashboard', icon: <Activity className="h-5 w-5" style={{ color: 'var(--text)' }} /> }
    : { to: '/history', label: 'Open incident history', icon: <History className="h-5 w-5" style={{ color: 'var(--text)' }} /> }

  return (
    <header className="sticky top-0 z-40 px-4 pt-4 md:px-6 md:pt-6">
      <div
        className="mx-auto flex max-w-6xl items-center justify-between rounded-[1.75rem] border px-4 py-3 shadow-[0_16px_48px_rgba(20,37,63,0.08)] backdrop-blur-xl md:px-6"
        style={{
          backgroundColor: 'var(--nav-surface, rgba(248,250,255,0.74))',
          borderColor: 'var(--nav-border, rgba(194,207,229,0.44))',
        }}
      >
        <Link to="/" className="flex min-w-0 items-center gap-3">
          <div
            className="flex h-11 w-11 items-center justify-center overflow-hidden rounded-2xl border"
            style={{
              backgroundColor: 'var(--status-icon-surface, rgba(238,244,255,0.8))',
              borderColor: 'var(--status-card-border, rgba(181,197,226,0.32))',
            }}
          >
            {logoUrl ? (
              <img src={logoUrl} alt={`${siteName} logo`} className="h-7 w-7 object-contain" />
            ) : (
              <Activity className="h-5 w-5" style={{ color: 'var(--primary)' }} />
            )}
          </div>
          <div className="min-w-0">
            <p className="truncate text-[11px] font-semibold uppercase tracking-[0.32em]" style={{ color: 'var(--text-subtle)' }}>
              Status Center
            </p>
            <p className="truncate text-base font-black md:text-lg" style={{ color: 'var(--text)' }}>
              {siteName}
            </p>
          </div>
        </Link>

        <div className="hidden items-center gap-2 md:flex">
          <Link
            to="/"
            aria-current={dashboardActive ? 'page' : undefined}
            className="rounded-full px-4 py-2 text-sm font-semibold transition-colors"
            style={{
              color: dashboardActive ? 'var(--primary)' : 'var(--text-muted)',
              backgroundColor: dashboardActive ? 'var(--status-pill-bg, rgba(16,185,129,0.1))' : 'transparent',
            }}
          >
            Dashboard
          </Link>
          <Link
            to="/history"
            aria-current={historyActive ? 'page' : undefined}
            className="inline-flex items-center gap-2 rounded-full px-4 py-2 text-sm font-semibold transition-colors"
            style={{
              color: historyActive ? 'var(--primary)' : 'var(--text-muted)',
              backgroundColor: historyActive ? 'var(--status-pill-bg, rgba(16,185,129,0.1))' : 'transparent',
            }}
          >
            <History className="h-4 w-4" />
            Incident History
          </Link>
        </div>

        <div className="flex items-center gap-3">
          <div className="hidden items-center gap-2 rounded-full px-3 py-2 md:inline-flex" style={{ backgroundColor: 'var(--surface)' }}>
            <span className="h-2.5 w-2.5 rounded-full animate-pulse" style={{ backgroundColor: 'var(--status-operational)' }} />
            <span className="text-xs font-bold uppercase tracking-[0.24em]" style={{ color: 'var(--text-muted)' }}>
              {statusLabel}
            </span>
          </div>
          <Link
            to={mobileDestination.to}
            className="inline-flex h-11 w-11 items-center justify-center rounded-2xl md:hidden"
            style={{ backgroundColor: 'var(--surface)' }}
            aria-label={mobileDestination.label}
          >
            {mobileDestination.icon}
          </Link>
        </div>
      </div>
    </header>
  )
}
