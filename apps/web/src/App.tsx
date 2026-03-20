import React from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
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
import AdminProfile from './pages/admin/AdminProfile'
import { getStoredToken, getStoredProfile } from './lib/auth'
import type { UserRole } from './types'

interface StoredAdminProfile {
  role?: UserRole
  mfaVerified?: boolean
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

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = getStoredToken()
  const profile = getStoredProfile()
  const location = useLocation()
  
  if (!token) return <Navigate to="/admin/login" replace />
  
  const isProfileRoute = location.pathname === '/admin/profile'
  const isMfaComplete = profile?.mfaVerified ?? false
  
  if (!isMfaComplete && !isProfileRoute) {
    return <Navigate to="/admin/profile" replace />
  }
  
  return <>{children}</>
}

function RoleRoute({ allowed, children }: { allowed: UserRole[]; children: React.ReactNode }) {
  const role = readStoredRole()
  if (!role || !allowed.includes(role)) return <Navigate to="/admin/incidents" replace />
  return <>{children}</>
}

function AdminIndexRedirect() {
  const profile = getStoredProfile()
  const role = readStoredRole()
  
  if (!(profile?.mfaVerified ?? false)) {
    return <Navigate to="/admin/profile" replace />
  }
  
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
        <Route path="profile" element={<AdminProfile />} />
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
          path="users"
          element={
            <RoleRoute allowed={['admin']}>
              <AdminMembers />
            </RoleRoute>
          }
        />
        <Route path="members" element={<Navigate to="/admin/users" replace />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
