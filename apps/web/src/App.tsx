import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import StatusPage from './pages/StatusPage'
import AdminLogin from './pages/admin/AdminLogin'
import AdminLayout from './pages/admin/AdminLayout'
import AdminDashboard from './pages/admin/AdminDashboard'
import AdminComponents from './pages/admin/AdminComponents'
import AdminSubComponents from './pages/admin/AdminSubComponents'
import AdminIncidents from './pages/admin/AdminIncidents'
import AdminMaintenance from './pages/admin/AdminMaintenance'
import AdminMonitors from './pages/admin/AdminMonitors'
import AdminSubscribers from './pages/admin/AdminSubscribers'
import AdminMembers from './pages/admin/AdminMembers'
import AdminActivate from './pages/admin/AdminActivate'
import type { AdminRole } from './types'

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

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('admin_token')
  if (!token) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}

function RoleRoute({ allowed, children }: { allowed: AdminRole[]; children: React.ReactNode }) {
  const role = readStoredRole()
  if (!role || !allowed.includes(role)) return <Navigate to="/admin/incidents" replace />
  return <>{children}</>
}

function AdminIndexRedirect() {
  const role = readStoredRole()
  if (role === 'operator') return <Navigate to="/admin/incidents" replace />
  return <AdminDashboard />
}

export default function App() {
  return (
    <Routes>
      {/* Public status page */}
      <Route path="/" element={<StatusPage />} />

      {/* Admin auth */}
      <Route path="/admin/login" element={<AdminLogin />} />
      <Route path="/admin/activate" element={<AdminActivate />} />

      {/* Admin protected routes */}
      <Route
        path="/admin"
        element={
          <ProtectedRoute>
            <AdminLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<AdminIndexRedirect />} />
        <Route
          path="components"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminComponents />
            </RoleRoute>
          }
        />
        <Route
          path="subcomponents"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminSubComponents />
            </RoleRoute>
          }
        />
        <Route path="incidents" element={<AdminIncidents />} />
        <Route path="maintenance" element={<AdminMaintenance />} />
        <Route
          path="monitors"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminMonitors />
            </RoleRoute>
          }
        />
        <Route
          path="subscribers"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminSubscribers />
            </RoleRoute>
          }
        />
        <Route
          path="members"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminMembers />
            </RoleRoute>
          }
        />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
