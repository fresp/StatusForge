import React, { useEffect, useMemo, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Layers,
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
  ChevronDown,
  ChevronRight,
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
const SIDEBAR_SECTION_STATE_KEY = 'admin_sidebar_section_state'

function sectionKey(label: string): string {
  return label.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

function isRouteActive(pathname: string, to: string, end: boolean): boolean {
  if (end) {
    return pathname === to
  }

  return pathname === to || pathname.startsWith(`${to}/`)
}

function isSectionActive(pathname: string, section: NavSection): boolean {
  return section.items.some(item => {
    if (isRouteActive(pathname, item.to, item.end)) {
      return true
    }

    if (!item.children || item.children.length === 0) {
      return false
    }

    return item.children.some(child => isRouteActive(pathname, child.to, child.end))
  })
}

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({})
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
  const visibleSectionKeys = useMemo(
    () => visibleNavSections.map(section => sectionKey(section.label)),
    [visibleNavSections],
  )

  useEffect(() => {
    let stored: Record<string, boolean> = {}

    try {
      const raw = localStorage.getItem(SIDEBAR_SECTION_STATE_KEY)
      if (raw) {
        const parsed = JSON.parse(raw) as Record<string, boolean>
        if (parsed && typeof parsed === 'object') {
          stored = parsed
        }
      }
    } catch {
      stored = {}
    }

    setOpenSections(prev => {
      const next: Record<string, boolean> = {}

      visibleSectionKeys.forEach(key => {
        if (typeof prev[key] === 'boolean') {
          next[key] = prev[key]
        } else if (typeof stored[key] === 'boolean') {
          next[key] = stored[key]
        } else {
          next[key] = true
        }
      })

      return next
    })
  }, [visibleSectionKeys])

  useEffect(() => {
    if (visibleSectionKeys.length === 0) return

    const hasCompleteSectionState = visibleSectionKeys.every(
      key => typeof openSections[key] === 'boolean',
    )

    if (!hasCompleteSectionState) return

    const next: Record<string, boolean> = {}
    visibleSectionKeys.forEach(key => {
      next[key] = openSections[key] ?? true
    })

    localStorage.setItem(SIDEBAR_SECTION_STATE_KEY, JSON.stringify(next))
  }, [openSections, visibleSectionKeys])

  useEffect(() => {
    setOpenSections(prev => {
      let changed = false
      const next = { ...prev }

      visibleNavSections.forEach(section => {
        const key = sectionKey(section.label)
        if (isSectionActive(location.pathname, section) && next[key] === false) {
          next[key] = true
          changed = true
        }
      })

      return changed ? next : prev
    })
  }, [location.pathname, visibleNavSections])

  function toggleSection(key: string) {
    setOpenSections(prev => ({
      ...prev,
      [key]: !(prev[key] ?? true),
    }))
  }

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

        <nav className="flex-1 min-h-0 overflow-y-auto px-3 pt-4 pb-8 [scrollbar-width:thin] [scrollbar-color:rgb(75_85_99)_transparent] [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-gray-700/70 hover:[&::-webkit-scrollbar-thumb]:bg-gray-600/80">
          {visibleNavSections.map(section => (
            <div key={section.label} className="mt-3 first:mt-0 space-y-1.5">
              {!isSidebarCollapsed && (
                <button
                  type="button"
                  onClick={() => toggleSection(sectionKey(section.label))}
                  aria-expanded={openSections[sectionKey(section.label)] ?? true}
                  className="sticky top-0 z-10 -mx-3 w-[calc(100%+1.5rem)] flex items-center justify-between px-3 pt-4 pb-2 text-[10px] font-semibold uppercase tracking-[0.12em] text-gray-500/90 bg-gray-900/95 backdrop-blur-sm"
                >
                  <span>{section.label}</span>
                  {(openSections[sectionKey(section.label)] ?? true) ? (
                    <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
                  )}
                </button>
              )}

              <div
                className={`grid transition-[grid-template-rows,opacity] duration-300 ease-in-out ${(openSections[sectionKey(section.label)] ?? true) || isSidebarCollapsed ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'}`}
              >
                <div className="overflow-hidden">
                  {section.items.map(({ to, label, icon: Icon, end, children }) => (
                    <div key={to} className="space-y-1">
                      <NavLink
                        to={to}
                        end={end}
                        title={isSidebarCollapsed ? label : undefined}
                        className={({ isActive }) =>
                          `flex items-center ${isSidebarCollapsed ? 'justify-center px-2' : 'gap-3 pl-3 pr-3'} py-2.5 rounded-lg border-l-2 text-sm leading-5 font-medium transition-colors ${
                            isActive
                              ? 'bg-blue-600/95 text-white border-blue-300 shadow-sm'
                              : 'text-gray-500 border-transparent hover:bg-gray-800/60 hover:text-gray-200'
                          }`
                        }
                      >
                        <Icon className="w-4 h-4 flex-shrink-0" />
                        {!isSidebarCollapsed && <span className="truncate">{label}</span>}
                      </NavLink>

                      {!isSidebarCollapsed && children && children.length > 0 && (
                        <div className="ml-6 pl-3 border-l border-gray-700/80 space-y-1.5">
                          {children.map(child => (
                            <NavLink
                              key={child.to}
                              to={child.to}
                              end={child.end}
                              className={({ isActive }) =>
                                `block px-3 py-2 rounded-md text-[13px] leading-5 font-medium transition-colors ${
                                  isActive
                                    ? 'bg-blue-600/90 text-white'
                                    : 'text-gray-400 hover:bg-gray-800/60 hover:text-gray-200'
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
              </div>
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
