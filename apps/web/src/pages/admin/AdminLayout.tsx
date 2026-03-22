import React, { useState } from 'react'
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
  Settings,
  LogOut,
  ExternalLink,
  User,
  Webhook,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'
import type { UserRole } from '../../types'

interface StoredAdminProfile {
  role?: UserRole
}

interface NavChildItem {
  to: string
  label: string
  end: boolean
}

interface NavItem {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  end: boolean
  children?: NavChildItem[]
}

interface NavSection {
  label: string
  items: NavItem[]
}

function readStoredRole(): UserRole | null {
  try {
    const raw = localStorage.getItem('user_profile') || localStorage.getItem('admin_profile')
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredAdminProfile
    return parsed.role ?? null
  } catch {
    return null
  }
}

const navSections: NavSection[] = [
  {
    label: 'Monitoring',
    items: [
      { to: '/admin', label: 'Dashboard', icon: LayoutDashboard, end: true },
      { to: '/admin/monitors', label: 'Monitors', icon: Activity, end: false },
      {
        to: '/admin/components',
        label: 'Components',
        icon: Layers,
        end: false,
        children: [{ to: '/admin/subcomponents', label: 'Sub-Components', end: false }],
      },
    ],
  },
  {
    label: 'Operations',
    items: [
      { to: '/admin/incidents', label: 'Incidents', icon: AlertTriangle, end: false },
      { to: '/admin/maintenance', label: 'Maintenance', icon: Wrench, end: false },
    ],
  },
  {
    label: 'Notifications',
    items: [
      { to: '/admin/subscribers', label: 'Subscribers', icon: Users, end: false },
      { to: '/admin/webhook-channels', label: 'Webhook Channels', icon: Webhook, end: false },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/admin/users', label: 'Users', icon: Shield, end: false },
      { to: '/admin/settings', label: 'Settings', icon: Settings, end: false },
      { to: '/admin/profile', label: 'My Profile', icon: User, end: true },
    ],
  },
]

const OPERATOR_ALLOWED = new Set(['/admin/incidents', '/admin/maintenance'])
const ALWAYS_ALLOWED = new Set(['/admin/profile'])

export default function AdminLayout() {
  const navigate = useNavigate()
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const role = readStoredRole()
  const visibleNavSections: NavSection[] = role === 'operator'
    ? navSections
        .map(section => {
          const items: NavItem[] = []

          section.items.forEach(item => {
            const visibleChildren = item.children?.filter(
              child => OPERATOR_ALLOWED.has(child.to) || ALWAYS_ALLOWED.has(child.to),
            )
            const isItemAllowed = OPERATOR_ALLOWED.has(item.to) || ALWAYS_ALLOWED.has(item.to)

            if (!isItemAllowed && (!visibleChildren || visibleChildren.length === 0)) {
              return
            }

            const nextItem: NavItem = visibleChildren
              ? { ...item, children: visibleChildren }
              : { ...item }

            items.push(nextItem)
          })

          return {
            ...section,
            items,
          }
        })
        .filter(section => section.items.length > 0)
    : navSections

  function handleLogout() {
    localStorage.removeItem('user_token')
    localStorage.removeItem('user_profile')
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_profile')
    navigate('/admin/login')
  }

  const sidebarWidthClass = isSidebarCollapsed ? 'w-16' : 'w-64'
  const sidebarOffsetClass = isSidebarCollapsed ? 'ml-16' : 'ml-64'

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-30 ${sidebarWidthClass} bg-gray-900 text-white flex flex-col overflow-hidden transition-[width] duration-300 ease-in-out`}
      >
        <div className="border-b border-gray-700">
          <div className="flex items-start justify-between gap-2 px-3 py-4 min-h-[76px]">
            <div className={`min-w-0 transition-opacity duration-200 ${isSidebarCollapsed ? 'opacity-0 pointer-events-none' : 'opacity-100'}`}>
              <h1 className="text-lg font-bold truncate">Status Platform</h1>
              <p className="text-xs text-gray-400 mt-0.5 truncate">User Console</p>
            </div>
            <button
              type="button"
              onClick={() => setIsSidebarCollapsed(prev => !prev)}
              aria-label={isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg text-gray-300 hover:text-white hover:bg-gray-800/80 transition-colors"
            >
              {isSidebarCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
            </button>
          </div>
        </div>

        <nav className="flex-1 py-4 px-3 overflow-y-auto min-h-0">
          {visibleNavSections.map(section => (
            <div key={section.label} className="space-y-1">
              {!isSidebarCollapsed && (
                <p className="sticky top-0 z-10 -mx-3 px-3 pt-3 pb-1 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-500 bg-gray-900/95 backdrop-blur-sm">
                  {section.label}
                </p>
              )}

              {section.items.map(({ to, label, icon: Icon, end, children }) => (
                <div key={to} className="space-y-1">
                  <NavLink
                    to={to}
                    end={end}
                    title={isSidebarCollapsed ? label : undefined}
                    className={({ isActive }) =>
                      `flex items-center ${isSidebarCollapsed ? 'justify-center px-2' : 'gap-3 px-3'} py-2 rounded-lg text-sm font-medium transition-all ${
                        isActive
                          ? 'bg-blue-600 text-white'
                          : 'text-gray-400 hover:bg-gray-800/70 hover:text-gray-200'
                      }`
                    }
                  >
                    <Icon className="w-4 h-4 flex-shrink-0" />
                    {!isSidebarCollapsed && <span className="truncate">{label}</span>}
                  </NavLink>

                  {!isSidebarCollapsed && children && children.length > 0 && (
                    <div className="ml-7 pl-3 border-l border-gray-800 space-y-1">
                      {children.map(child => (
                        <NavLink
                          key={child.to}
                          to={child.to}
                          end={child.end}
                          className={({ isActive }) =>
                            `block px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                              isActive
                                ? 'bg-blue-600 text-white'
                                : 'text-gray-500 hover:bg-gray-800/60 hover:text-gray-200'
                            }`
                          }
                        >
                          {child.label}
                        </NavLink>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ))}
        </nav>

        <footer className="px-3 py-4 border-t border-gray-700 space-y-1">
          <a
            href="/"
            target="_blank"
            rel="noopener noreferrer"
            title={isSidebarCollapsed ? 'View Status Page' : undefined}
            className={`flex items-center ${isSidebarCollapsed ? 'justify-center px-2' : 'gap-3 px-3'} py-2 rounded-lg text-sm text-gray-400 hover:bg-gray-800/70 hover:text-gray-200 transition-colors`}
          >
            <ExternalLink className="w-4 h-4" />
            {!isSidebarCollapsed && 'View Status Page'}
          </a>
          <button
            onClick={handleLogout}
            title={isSidebarCollapsed ? 'Logout' : undefined}
            className={`w-full flex items-center ${isSidebarCollapsed ? 'justify-center px-2' : 'gap-3 px-3'} py-2 rounded-lg text-sm text-gray-400 hover:bg-red-700/70 hover:text-white transition-colors`}
          >
            <LogOut className="w-4 h-4" />
            {!isSidebarCollapsed && 'Logout'}
          </button>
        </footer>
      </aside>

      {/* Main content */}
      <main className={`${sidebarOffsetClass} h-screen overflow-auto transition-[margin] duration-300 ease-in-out`}>
        <Outlet />
      </main>
    </div>
  )
}
