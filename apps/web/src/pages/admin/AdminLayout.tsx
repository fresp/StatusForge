import React, { useEffect, useMemo, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import api from '../../lib/api'
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
import type { StatusPageSettings } from '../../types'
import { useApi } from '../../hooks/useApi'

const DEFAULT_PAGE_TITLE = 'Statora'

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
  const { data: settingsData } = useApi<StatusPageSettings>('/settings/status-page')
  const pageTitle = settingsData?.head?.title?.trim() || DEFAULT_PAGE_TITLE
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

  async function handleLogout() {
    try {
      await api.post('/auth/logout')
    } catch {
      // ignore logout request failures and still clear local state
    }

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

  useEffect(() => {
    document.title = `${pageTitle} - Admin Panel`
  }, [pageTitle])

  function toggleSection(key: string) {
    setOpenSections(prev => ({
      ...prev,
      [key]: !(prev[key] ?? true),
    }))
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-30 ${sidebarWidthClass} bg-[#0e1526] text-slate-300 flex flex-col overflow-hidden transition-[width] duration-300 ease-in-out border-r border-slate-800/50 shadow-2xl shadow-slate-900/20`}
      >
        <div className="border-b border-slate-800/50 bg-[#0b101e]/50 backdrop-blur-sm">
          <div className="flex items-start justify-between gap-2 px-4 py-5 min-h-[80px]">
            <div className={`min-w-0 transition-opacity duration-200 ${isSidebarCollapsed ? 'opacity-0 pointer-events-none' : 'opacity-100'}`}>
              <h1 className="text-base font-bold text-white tracking-wide truncate">{pageTitle}</h1>
              <p className="text-[11px] font-medium text-blue-400 mt-1 uppercase tracking-wider truncate">Operations Console</p>
            </div>
            <button
              type="button"
              onClick={() => setIsSidebarCollapsed(prev => !prev)}
              aria-label={isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              className="inline-flex h-8 w-8 items-center justify-center rounded-md text-slate-400 hover:text-white hover:bg-slate-800/80 transition-all duration-200"
            >
              {isSidebarCollapsed ? <PanelLeftOpen className="h-[18px] w-[18px]" /> : <PanelLeftClose className="h-[18px] w-[18px]" />}
            </button>
          </div>
        </div>

        <nav className="flex-1 min-h-0 overflow-y-auto px-3 pt-5 pb-8 [scrollbar-width:thin] [scrollbar-color:rgb(51_65_85)_transparent] [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-slate-700/50 hover:[&::-webkit-scrollbar-thumb]:bg-slate-600/80">
          {visibleNavSections.map(section => (
            <div key={section.label} className="mb-6 last:mb-0 space-y-1">
              {!isSidebarCollapsed && (
                <button
                  type="button"
                  onClick={() => toggleSection(sectionKey(section.label))}
                  aria-expanded={openSections[sectionKey(section.label)] ?? true}
                  className="w-full flex items-center justify-between px-3 py-2 text-[10px] font-bold uppercase tracking-[0.15em] text-slate-500 hover:text-slate-300 transition-colors group"
                >
                  <span>{section.label}</span>
                  {(openSections[sectionKey(section.label)] ?? true) ? (
                    <ChevronDown className="h-3 w-3 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
                  ) : (
                    <ChevronRight className="h-3 w-3 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
                  )}
                </button>
              )}

              <div
                className={`grid transition-[grid-template-rows,opacity] duration-300 ease-in-out ${(openSections[sectionKey(section.label)] ?? true) || isSidebarCollapsed ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'}`}
              >
                <div className="overflow-hidden space-y-0.5">
                  {section.items.map(({ to, label, icon: Icon, end, children }) => (
                    <div key={to} className="space-y-0.5">
                      <NavLink
                        to={to}
                        end={end}
                        title={isSidebarCollapsed ? label : undefined}
                        className={({ isActive }) =>
                          `flex items-center ${isSidebarCollapsed ? 'justify-center px-0 w-10 mx-auto' : 'gap-3 px-3'} py-2 rounded-lg text-[13px] leading-5 font-medium transition-all duration-200 ${isActive
                            ? 'bg-blue-600/10 text-blue-400 shadow-[inset_2px_0_0_0_rgb(96,165,250)]'
                            : 'text-slate-400 border-transparent hover:bg-slate-800/50 hover:text-slate-200'
                          }`
                        }
                      >
                        <Icon className={`w-[18px] h-[18px] flex-shrink-0 ${isSidebarCollapsed ? '' : 'opacity-80'}`} />
                        {!isSidebarCollapsed && <span className="truncate">{label}</span>}
                      </NavLink>

                      {!isSidebarCollapsed && children && children.length > 0 && (
                        <div className="ml-[22px] pl-3 border-l border-slate-800/80 space-y-0.5 mt-0.5">
                          {children.map(child => (
                            <NavLink
                              key={child.to}
                              to={child.to}
                              end={child.end}
                              className={({ isActive }) =>
                                `block px-3 py-1.5 rounded-md text-[13px] leading-5 font-medium transition-colors ${isActive
                                  ? 'text-blue-400 bg-blue-600/10'
                                  : 'text-slate-500 hover:bg-slate-800/30 hover:text-slate-300'
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

        <footer className="p-4 bg-[#0b101e]/30 border-t border-slate-800/50 space-y-1">
          <a
            href="/"
            target="_blank"
            rel="noopener noreferrer"
            title={isSidebarCollapsed ? 'View Status Page' : undefined}
            className={`flex items-center ${isSidebarCollapsed ? 'justify-center px-0 w-10 mx-auto' : 'gap-3 px-3'} py-2 rounded-lg text-[13px] font-medium text-slate-400 hover:bg-slate-800/50 hover:text-slate-200 transition-colors`}
          >
            <ExternalLink className="w-[18px] h-[18px] opacity-80" />
            {!isSidebarCollapsed && 'Status Page'}
          </a>
          <button
            type="button"
            onClick={handleLogout}
            title={isSidebarCollapsed ? 'Logout' : undefined}
            className={`w-full flex items-center ${isSidebarCollapsed ? 'justify-center px-0 w-10 mx-auto' : 'gap-3 px-3'} py-2 rounded-lg text-[13px] font-medium text-slate-400 hover:bg-rose-500/10 hover:text-rose-400 transition-colors`}
          >
            <LogOut className="w-[18px] h-[18px] opacity-80" />
            {!isSidebarCollapsed && 'Logout'}
          </button>
        </footer>
      </aside>

      {/* Main content */}
      <main className={`${sidebarOffsetClass} h-screen overflow-auto transition-[margin] duration-300 ease-in-out`}>
        <div className="max-w-7xl mx-auto p-8 md:p-10 lg:p-12">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
