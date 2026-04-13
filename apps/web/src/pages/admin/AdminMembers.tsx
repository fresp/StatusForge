import React, { useMemo, useState } from 'react'
import { Shield, RefreshCw, Plus, X } from 'lucide-react'
import api from '../../lib/api'
import { useApi } from '../../hooks/useApi'
import { useAdminPagination } from '../../hooks/useAdminPagination'
import type { UserInvitation, UserMember, UserRole, UserStatus } from '../../types'
import AdminPaginationControls from '../../components/AdminPaginationControls'
import Modal from '../../components/Modal'
import UserSearch from '../../components/UserSearch'

const ROLE_OPTIONS: UserRole[] = ['admin', 'operator']
const INVITE_ROLE_OPTIONS: Extract<UserRole, 'admin' | 'operator'>[] = ['admin', 'operator']

const STATUS_LABELS: Record<UserStatus, string> = {
  active: 'Active',
  disabled: 'Disabled',
  invited: 'Invited',
}

const STATUS_BADGE_CLASS: Record<UserStatus, string> = {
  active: 'badge badge-success',
  disabled: 'badge badge-error',
  invited: 'badge badge-warning',
}

interface MeResponse {
  userId?: string
  username?: string
}

interface MembersResponse {
  items?: UserMember[]
}

interface InvitationsResponse {
  items?: UserInvitation[]
}

interface InviteResponse {
  activationLink?: string
  activationUrl?: string
  inviteLink?: string
  token?: string
}

interface StoredAdminProfile {
  id?: string
  username?: string
  email?: string
  role?: UserRole
}

function readStoredAdminProfile(): StoredAdminProfile | null {
  try {
    const raw = localStorage.getItem('user_profile') || localStorage.getItem('admin_profile')
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredAdminProfile
    return parsed
  } catch {
    return null
  }
}

export default function AdminMembers() {
  const membersPagination = useAdminPagination({ pageParam: 'usersPage', limitParam: 'usersLimit' })
  const invitationsPagination = useAdminPagination({
    pageParam: 'invitesPage',
    limitParam: 'invitesLimit',
  })

  const {
    data,
    total: membersTotal,
    totalPages: membersTotalPages,
    loading,
    error,
    refetch,
  } = useApi<UserMember[] | MembersResponse>('/users', [], membersPagination.apiParams)
  const {
    data: invitationsData,
    total: invitationsTotal,
    totalPages: invitationsTotalPages,
    loading: invitationsLoading,
    error: invitationsError,
    refetch: refetchInvitations,
  } = useApi<UserInvitation[] | InvitationsResponse>('/users/invitations', [], invitationsPagination.apiParams)
  const { data: meData } = useApi<MeResponse>('/auth/me')
  const [savingId, setSavingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string>('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<Extract<UserRole, 'operator'>>('operator')
  const [inviting, setInviting] = useState(false)
  const [inviteError, setInviteError] = useState('')
  const [activationLink, setActivationLink] = useState('')
  const [copySuccess, setCopySuccess] = useState('')
  const [inviteActionId, setInviteActionId] = useState<string | null>(null)
  const [inviteActionError, setInviteActionError] = useState('')
  const [activeTab, setActiveTab] = useState<'members' | 'invitations'>('members')

  const members = useMemo(() => {
    if (!data) return []
    if (Array.isArray(data)) return data
    if (Array.isArray(data.items)) return data.items
    return []
  }, [data])

  const currentUserId = useMemo(() => {
    if (meData?.userId) return meData.userId
    return readStoredAdminProfile()?.id || null
  }, [meData])

  const invitations = useMemo(() => {
    if (!invitationsData) return []
    if (Array.isArray(invitationsData)) return invitationsData
    if (Array.isArray(invitationsData.items)) return invitationsData.items
    return []
  }, [invitationsData])

  async function updateMember(id: string, payload: Partial<Pick<UserMember, 'role' | 'status' | 'ssoEnabled'>>) {
    setSavingId(id)
    setActionError('')
    try {
      await api.patch(`/users/${id}`, payload)
      await refetch()
    } catch (err: any) {
      setActionError(err.response?.data?.error || 'Failed to update user. Please try again.')
    } finally {
      setSavingId(null)
    }
  }

  async function deleteMember(member: UserMember) {
    const confirmed = window.confirm(`Remove user ${member.username} (${member.email})? This action cannot be undone.`)
    if (!confirmed) return

    setDeletingId(member.id)
    setActionError('')
    try {
      await api.delete(`/users/${member.id}`)
      await refetch()
    } catch (err: any) {
      setActionError(err.response?.data?.error || 'Failed to delete user. Please try again.')
    } finally {
      setDeletingId(null)
    }
  }

  function openInviteModal() {
    setInviteEmail('')
    setInviteRole('operator')
    setInviteError('')
    setActivationLink('')
    setCopySuccess('')
    setShowInviteModal(true)
  }

  async function handleInvite(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setInviting(true)
    setInviteError('')
    setCopySuccess('')
    try {
      const res = await api.post<InviteResponse>('/users/invitations', {
        email: inviteEmail,
        role: inviteRole,
      })

      const inviteData = res.data
      const link = inviteData.activationLink || inviteData.activationUrl || inviteData.inviteLink || ''

      if (link) {
        setActivationLink(link)
      } else if (inviteData.token) {
        const origin = window.location.origin
        setActivationLink(`${origin}/admin/activate?token=${encodeURIComponent(inviteData.token)}`)
      } else {
        setInviteError('Invitation sent, but activation link was not returned by API.')
      }

      await refetch()
    } catch (err: any) {
      setInviteError(err.response?.data?.error || 'Failed to invite user. Please try again.')
    } finally {
      setInviting(false)
    }
  }

  async function handleCopyActivationLink() {
    if (!activationLink) return
    try {
      await navigator.clipboard.writeText(activationLink)
      setCopySuccess('Activation link copied')
    } catch {
      setCopySuccess('Copy failed. Please copy manually.')
    }
  }

  async function refreshInvitationToken(invitation: UserInvitation) {
    setInviteActionId(invitation.id)
    setInviteActionError('')
    setCopySuccess('')
    try {
      const res = await api.post<InviteResponse>(`/users/invitations/${invitation.id}/refresh`)
      const inviteData = res.data
      const link = inviteData.activationLink || inviteData.activationUrl || inviteData.inviteLink || ''
      if (link) {
        setActivationLink(link)
      } else if (inviteData.token) {
        const origin = window.location.origin
        setActivationLink(`${origin}/admin/activate?token=${encodeURIComponent(inviteData.token)}`)
      } else {
        setInviteActionError('Token refreshed, but activation link was not returned.')
      }
      await refetchInvitations()
    } catch (err: any) {
      setInviteActionError(err.response?.data?.error || 'Failed to refresh token.')
    } finally {
      setInviteActionId(null)
    }
  }

  async function removeInvitation(invitationID: string) {
    setInviteActionId(invitationID)
    setInviteActionError('')
    try {
      await api.delete(`/users/invitations/${invitationID}`)
      await refetchInvitations()
    } catch (err: any) {
      setInviteActionError(err.response?.data?.error || 'Failed to remove invitation.')
    } finally {
      setInviteActionId(null)
    }
  }

  return (
    <div className="max-w-6xl">
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">Users</h1>
          <p className="mt-1 text-sm text-slate-500">Manage user accounts, roles, invitations, and access state.</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={() => {
              void refetch()
              void refetchInvitations()
            }}
            disabled={loading}
            className="admin-btn-secondary"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
          <button
            type="button"
            onClick={openInviteModal}
            className="admin-btn-primary"
          >
            <Plus className="h-4 w-4" />
            Invite User
          </button>
        </div>
      </div>

      {actionError && (
        <div className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
          {actionError}
        </div>
      )}

      {error && !loading && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
          Failed to load users.
        </div>
      )}

      {loading && members.length === 0 && (
        <div className="admin-surface p-10 text-center text-sm text-slate-500">
          Loading users...
        </div>
      )}

      {!loading && !error && members.length === 0 && (
        <div className="admin-surface p-10 text-center text-sm text-slate-500">
          No users found.
        </div>
      )}

      <div className="mb-6">
        <div className="inline-flex rounded-full border border-slate-200 bg-white p-1 shadow-sm">
          <button
            type="button"
            onClick={() => setActiveTab('members')}
            className={`rounded-full px-4 py-2 text-sm font-medium transition-all ${activeTab === 'members' ? 'bg-gradient-to-b from-blue-500 to-blue-600 text-white shadow-sm' : 'text-slate-600 hover:bg-slate-100'
              }`}
          >
            Users
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('invitations')}
            className={`rounded-full px-4 py-2 text-sm font-medium transition-all ${activeTab === 'invitations' ? 'bg-gradient-to-b from-blue-500 to-blue-600 text-white shadow-sm' : 'text-slate-600 hover:bg-slate-100'
              }`}
          >
            Invitations
          </button>
        </div>
      </div>

      {activeTab === 'members' && (
        <UserSearch />
      )}

      {activeTab === 'members' && members.length > 0 && (
        <div className="admin-surface">
          <table className="w-full text-sm">
            <thead className="bg-slate-50/80">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">User</th>
                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Role</th>
                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Status</th>
                <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Actions</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => {
                const isSaving = savingId === member.id
                const isDeleting = deletingId === member.id
                const isAdmin = member.role === 'admin'
                const isCurrentUser = currentUserId === member.id
                const canEditRole = !isAdmin && !isSaving && !isDeleting
                const canToggleStatus = !isAdmin && !isCurrentUser && member.status !== 'invited' && !isSaving && !isDeleting
                const canDelete = !isCurrentUser && !isSaving && !isDeleting
                const nextStatus: UserStatus = member.status === 'disabled' ? 'active' : 'disabled'

                return (
                  <tr key={member.id} className="border-t border-slate-100/80 transition-colors hover:bg-slate-50/80">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-slate-100 text-slate-500">
                          <Shield className="h-4 w-4" />
                        </div>
                        <div>
                          <p className="font-medium text-slate-900">{member.username}</p>
                          <p className="text-xs text-slate-500">{member.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <select
                        value={member.role}
                        disabled={!canEditRole}
                        onChange={(e) => {
                          const role = e.target.value as UserRole
                          void updateMember(member.id, { role })
                        }}
                        className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 shadow-sm focus:outline-none focus:ring-4 focus:ring-blue-500/10 focus:border-blue-500 disabled:opacity-60"
                      >
                        {ROLE_OPTIONS.map((role) => (
                          <option key={role} value={role}>
                            {role}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex flex-col gap-2">
                        <span className={STATUS_BADGE_CLASS[member.status]}>
                          {STATUS_LABELS[member.status]}
                        </span>
                        <label className="inline-flex items-center gap-2 text-xs text-slate-600">
                          <input
                            type="checkbox"
                            checked={member.ssoEnabled}
                            disabled={isSaving || isDeleting}
                            onChange={(e) => {
                              void updateMember(member.id, { ssoEnabled: e.target.checked })
                            }}
                          />
                          SSO enabled
                        </label>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => {
                            void updateMember(member.id, { status: nextStatus })
                          }}
                          disabled={!canToggleStatus}
                          className="rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {member.status === 'invited'
                            ? 'Pending Invite'
                            : member.status === 'disabled'
                              ? 'Enable'
                              : 'Disable'}
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            void deleteMember(member)
                          }}
                          disabled={!canDelete}
                          className="rounded-full border border-red-200 bg-white px-3 py-1.5 text-xs font-medium text-red-700 shadow-sm transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {isDeleting ? 'Removing...' : 'Remove'}
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>

          <AdminPaginationControls
            page={membersPagination.page}
            totalPages={membersTotalPages}
            total={membersTotal}
            limit={membersPagination.limit}
            loading={loading}
            onPageChange={membersPagination.setPage}
            onLimitChange={membersPagination.setLimit}
          />
        </div>
      )}

      <div className="mt-8" hidden={activeTab !== 'invitations'}>
        <h2 className="text-lg font-semibold tracking-tight text-slate-900">Invited Users</h2>
        <p className="mt-1 text-sm text-slate-500">Pending invitations, token refresh, and removal.</p>

        {inviteActionError && (
          <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
            {inviteActionError}
          </div>
        )}

        {invitationsError && !invitationsLoading && (
          <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
            Failed to load invitations.
          </div>
        )}

        {invitationsLoading && invitations.length === 0 && (
          <div className="admin-surface mt-4 p-8 text-center text-sm text-slate-500">
            Loading invitations...
          </div>
        )}

        {!invitationsLoading && !invitationsError && invitations.length === 0 && (
          <div className="admin-surface mt-4 p-8 text-center text-sm text-slate-500">
            No pending invitations.
          </div>
        )}

        {invitations.length > 0 && (
          <div className="admin-surface mt-4">
            <table className="w-full text-sm">
              <thead className="bg-slate-50/80">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Email</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Role</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Expires</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">Actions</th>
                </tr>
              </thead>
              <tbody>
                {invitations.map((invitation) => {
                  const isActing = inviteActionId === invitation.id
                  return (
                    <tr key={invitation.id} className="border-t border-slate-100/80 transition-colors hover:bg-slate-50/80">
                      <td className="px-6 py-4 font-medium text-slate-900">{invitation.email}</td>
                      <td className="px-6 py-4 text-slate-700">{invitation.role}</td>
                      <td className="px-6 py-4">
                        <span className={invitation.isExpired ? 'text-xs font-medium text-red-600' : 'text-xs text-slate-600'}>
                          {new Date(invitation.expiresAt).toLocaleString()}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex gap-2">
                          <button
                            type="button"
                            onClick={() => {
                              void refreshInvitationToken(invitation)
                            }}
                            disabled={isActing}
                            className="rounded-full border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-100 disabled:opacity-50"
                          >
                            Refresh Token
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              void removeInvitation(invitation.id)
                            }}
                            disabled={isActing}
                            className="rounded-full border border-red-200 bg-white px-3 py-1.5 text-xs font-medium text-red-700 shadow-sm transition-colors hover:bg-red-50 disabled:opacity-50"
                          >
                            Remove
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>

            <AdminPaginationControls
              page={invitationsPagination.page}
              totalPages={invitationsTotalPages}
              total={invitationsTotal}
              limit={invitationsPagination.limit}
              loading={invitationsLoading}
              onPageChange={invitationsPagination.setPage}
              onLimitChange={invitationsPagination.setLimit}
            />
          </div>
        )}
      </div>

      {showInviteModal && (
        <Modal title="Invite User" onClose={() => setShowInviteModal(false)} size="lg">
          <form onSubmit={handleInvite} className="space-y-5">
              {inviteError && (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
                  {inviteError}
                </div>
              )}

              <div>
                <label htmlFor="inviteEmail" className="mb-1.5 block text-sm font-medium text-slate-700">Email</label>
                <input
                  id="inviteEmail"
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  required
                  disabled={inviting}
                  className="admin-input"
                  placeholder="user@company.com"
                />
              </div>

              <div>
                <label htmlFor="inviteRole" className="mb-1.5 block text-sm font-medium text-slate-700">Role</label>
                <select
                  id="inviteRole"
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value as Extract<UserRole, 'operator'>)}
                  disabled={inviting}
                  className="admin-input"
                >
                  {INVITE_ROLE_OPTIONS.map((role) => (
                    <option key={role} value={role}>
                      {role}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={inviting}
                  className="admin-btn-primary"
                >
                  {inviting ? 'Inviting...' : 'Invite'}
                </button>
              </div>

              {activationLink && (
                <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-4 shadow-sm">
                  <p className="mb-3 text-sm font-medium text-slate-700">Activation Link</p>
                  <div className="flex gap-2">
                    <input
                      value={activationLink}
                      readOnly
                      className="admin-input text-xs"
                    />
                    <button
                      type="button"
                      onClick={() => void handleCopyActivationLink()}
                      className="admin-btn-secondary whitespace-nowrap"
                    >
                      Copy
                    </button>
                  </div>
                  {copySuccess && <p className="mt-2 text-xs text-slate-600">{copySuccess}</p>}
                </div>
              )}
            </form>
        </Modal>
      )}
    </div>
  )
}
