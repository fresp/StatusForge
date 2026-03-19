import React from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Layers,
  GitBranch,
  AlertTriangle,
  Wrench,
  Activity,
  Users,
  Shield,
  LogOut,
  ExternalLink,
} from 'lucide-react'
import type { AdminRole } from '../../types'

interface StoredAdminProfile {
  role?: AdminRole
}

function readStoredRole(): AdminRole | null {
  try {
    const raw = localStorage.getItem('admin_profile')
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredAdminProfile
    return parsed.role ?? null
  } catch {
    return null
  }
}

const navItems = [
  { to: '/admin', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/admin/components', label: 'Components', icon: Layers, end: false },
  { to: '/admin/subcomponents', label: 'Sub-Components', icon: GitBranch, end: false },
  { to: '/admin/incidents', label: 'Incidents', icon: AlertTriangle, end: false },
  { to: '/admin/maintenance', label: 'Maintenance', icon: Wrench, end: false },
  { to: '/admin/monitors', label: 'Monitors', icon: Activity, end: false },
  { to: '/admin/subscribers', label: 'Subscribers', icon: Users, end: false },
  { to: '/admin/members', label: 'Members', icon: Shield, end: false },
]

const OPERATOR_ALLOWED = new Set(['/admin/incidents', '/admin/maintenance'])

export default function AdminLayout() {
  const navigate = useNavigate()
  const role = readStoredRole()
  const visibleNavItems = role === 'operator' ? navItems.filter(item => OPERATOR_ALLOWED.has(item.to)) : navItems

  function handleLogout() {
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_profile')
    navigate('/admin/login')
  }

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Sidebar */}
      <aside className="w-60 bg-gray-900 text-white flex flex-col flex-shrink-0">
        <div className="px-6 py-5 border-b border-gray-700">
          <h1 className="text-lg font-bold">Status Platform</h1>
          <p className="text-xs text-gray-400 mt-0.5">Admin Console</p>
        </div>

        <nav className="flex-1 py-4 space-y-0.5 px-3">
          {visibleNavItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`
              }
            >
              <Icon className="w-4 h-4 flex-shrink-0" />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="px-3 py-4 border-t border-gray-700 space-y-1">
          <a
            href="/"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-gray-300 hover:bg-gray-800 hover:text-white transition-colors"
          >
            <ExternalLink className="w-4 h-4" />
            View Status Page
          </a>
          <button
            onClick={handleLogout}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-gray-300 hover:bg-red-700 hover:text-white transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Logout
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
