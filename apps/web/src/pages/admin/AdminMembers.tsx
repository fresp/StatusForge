import React, { useMemo, useState } from 'react'
import { Shield, RefreshCw, Plus, X } from 'lucide-react'
import api from '../../lib/api'
import { useApi } from '../../hooks/useApi'
import type { AdminInvitation, AdminMember, AdminRole, AdminStatus } from '../../types'

const ROLE_OPTIONS: AdminRole[] = ['admin', 'operator']
const INVITE_ROLE_OPTIONS: Extract<AdminRole, 'admin' | 'operator'>[] = ['admin', 'operator']

const STATUS_LABELS: Record<AdminStatus, string> = {
  active: 'Active',
  disabled: 'Disabled',
  invited: 'Invited',
}

const STATUS_BADGE_CLASS: Record<AdminStatus, string> = {
  active: 'bg-green-100 text-green-700',
  disabled: 'bg-red-100 text-red-700',
  invited: 'bg-yellow-100 text-yellow-700',
}

interface MeResponse {
  adminId?: string
  username?: string
}

interface MembersResponse {
  items?: AdminMember[]
}

interface InvitationsResponse {
  items?: AdminInvitation[]
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
  role?: AdminRole
}

function readStoredAdminProfile(): StoredAdminProfile | null {
  try {
    const raw = localStorage.getItem('admin_profile')
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredAdminProfile
    return parsed
  } catch {
    return null
  }
}

export default function AdminMembers() {
  const { data, loading, error, refetch } = useApi<AdminMember[] | MembersResponse>('/admins')
  const {
    data: invitationsData,
    loading: invitationsLoading,
    error: invitationsError,
    refetch: refetchInvitations,
  } = useApi<AdminInvitation[] | InvitationsResponse>('/admins/invitations')
  const { data: meData } = useApi<MeResponse>('/auth/me')
  const [savingId, setSavingId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string>('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<Extract<AdminRole, 'admin' | 'operator'>>('admin')
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

  const currentAdminId = useMemo(() => {
    if (meData?.adminId) return meData.adminId
    return readStoredAdminProfile()?.id || null
  }, [meData])

  const invitations = useMemo(() => {
    if (!invitationsData) return []
    if (Array.isArray(invitationsData)) return invitationsData
    if (Array.isArray(invitationsData.items)) return invitationsData.items
    return []
  }, [invitationsData])

  async function updateMember(id: string, payload: Partial<Pick<AdminMember, 'role' | 'status'>>) {
    setSavingId(id)
    setActionError('')
    try {
      await api.patch(`/admins/${id}`, payload)
      await refetch()
    } catch (err: any) {
      setActionError(err.response?.data?.error || 'Failed to update member. Please try again.')
    } finally {
      setSavingId(null)
    }
  }

  function openInviteModal() {
    setInviteEmail('')
    setInviteRole('admin')
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
      const res = await api.post<InviteResponse>('/admins/invitations', {
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
      setInviteError(err.response?.data?.error || 'Failed to invite member. Please try again.')
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

  async function refreshInvitationToken(invitation: AdminInvitation) {
    setInviteActionId(invitation.id)
    setInviteActionError('')
    setCopySuccess('')
    try {
      const res = await api.post<InviteResponse>(`/admins/invitations/${invitation.id}/refresh`)
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
      await api.delete(`/admins/invitations/${invitationID}`)
      await refetchInvitations()
    } catch (err: any) {
      setInviteActionError(err.response?.data?.error || 'Failed to remove invitation.')
    } finally {
      setInviteActionId(null)
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Members</h1>
          <p className="text-sm text-gray-500 mt-1">Manage admin accounts, roles, and status</p>
        </div>
        <button
          onClick={() => {
            void refetch()
            void refetchInvitations()
          }}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-60"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
        <button
          onClick={openInviteModal}
          className="ml-2 inline-flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-sm text-white hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          Invite Member
        </button>
      </div>

      {actionError && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {actionError}
        </div>
      )}

      {error && !loading && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          Failed to load members.
        </div>
      )}

      {loading && members.length === 0 && (
        <div className="rounded-xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          Loading members...
        </div>
      )}

      {!loading && !error && members.length === 0 && (
        <div className="rounded-xl border border-gray-200 bg-white p-8 text-center text-gray-500">
          No members found.
        </div>
      )}

      <div className="mb-5">
        <div className="inline-flex rounded-lg border border-gray-200 bg-white p-1">
          <button
            onClick={() => setActiveTab('members')}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              activeTab === 'members' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            Members
          </button>
          <button
            onClick={() => setActiveTab('invitations')}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              activeTab === 'invitations' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            Invitations
          </button>
        </div>
      </div>

      {activeTab === 'members' && members.length > 0 && (
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-100">
              <tr>
                <th className="text-left px-6 py-3 font-medium text-gray-600">Member</th>
                <th className="text-left px-6 py-3 font-medium text-gray-600">Role</th>
                <th className="text-left px-6 py-3 font-medium text-gray-600">Status</th>
                <th className="text-left px-6 py-3 font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {members.map((member) => {
                const isSaving = savingId === member.id
                const isAdmin = member.role === 'admin'
                const isCurrentUser = currentAdminId === member.id
                const canEditRole = !isAdmin && !isSaving
                const canToggleStatus = !isAdmin && !isCurrentUser && member.status !== 'invited' && !isSaving
                const nextStatus: AdminStatus = member.status === 'disabled' ? 'active' : 'disabled'

                return (
                  <tr key={member.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <Shield className="w-4 h-4 text-gray-400" />
                        <div>
                          <p className="font-medium text-gray-900">{member.username}</p>
                          <p className="text-xs text-gray-500">{member.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <select
                        value={member.role}
                        disabled={!canEditRole}
                        onChange={(e) => {
                          const role = e.target.value as AdminRole
                          void updateMember(member.id, { role })
                        }}
                        className="rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-60"
                      >
                        {ROLE_OPTIONS.map((role) => (
                          <option key={role} value={role}>
                            {role}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="px-6 py-4">
                      <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_BADGE_CLASS[member.status]}`}>
                        {STATUS_LABELS[member.status]}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <button
                        onClick={() => {
                          void updateMember(member.id, { status: nextStatus })
                        }}
                        disabled={!canToggleStatus}
                        className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {member.status === 'invited'
                          ? 'Pending Invite'
                          : member.status === 'disabled'
                            ? 'Enable'
                            : 'Disable'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-8" hidden={activeTab !== 'invitations'}>
        <h2 className="text-lg font-semibold text-gray-900">Invited Members</h2>
        <p className="text-sm text-gray-500 mt-1">Pending invitations, token refresh, and removal.</p>

        {inviteActionError && (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {inviteActionError}
          </div>
        )}

        {invitationsError && !invitationsLoading && (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            Failed to load invitations.
          </div>
        )}

        {invitationsLoading && invitations.length === 0 && (
          <div className="mt-4 rounded-xl border border-gray-200 bg-white p-6 text-center text-gray-500">
            Loading invitations...
          </div>
        )}

        {!invitationsLoading && !invitationsError && invitations.length === 0 && (
          <div className="mt-4 rounded-xl border border-gray-200 bg-white p-6 text-center text-gray-500">
            No pending invitations.
          </div>
        )}

        {invitations.length > 0 && (
          <div className="mt-4 bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-100">
                <tr>
                  <th className="text-left px-6 py-3 font-medium text-gray-600">Email</th>
                  <th className="text-left px-6 py-3 font-medium text-gray-600">Role</th>
                  <th className="text-left px-6 py-3 font-medium text-gray-600">Expires</th>
                  <th className="text-left px-6 py-3 font-medium text-gray-600">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {invitations.map((invitation) => {
                  const isActing = inviteActionId === invitation.id
                  return (
                    <tr key={invitation.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 text-gray-900">{invitation.email}</td>
                      <td className="px-6 py-4 text-gray-700">{invitation.role}</td>
                      <td className="px-6 py-4">
                        <span className={invitation.isExpired ? 'text-red-600 text-xs' : 'text-gray-600 text-xs'}>
                          {new Date(invitation.expiresAt).toLocaleString()}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex gap-2">
                          <button
                            onClick={() => {
                              void refreshInvitationToken(invitation)
                            }}
                            disabled={isActing}
                            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                          >
                            Refresh Token
                          </button>
                          <button
                            onClick={() => {
                              void removeInvitation(invitation.id)
                            }}
                            disabled={isActing}
                            className="rounded-lg border border-red-200 bg-white px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
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
          </div>
        )}
      </div>

      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-xl bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Invite Member</h2>
              <button
                onClick={() => setShowInviteModal(false)}
                className="text-gray-400 hover:text-gray-600"
                aria-label="Close invite modal"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleInvite} className="space-y-4 px-6 py-5">
              {inviteError && (
                <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {inviteError}
                </div>
              )}

              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Email</label>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  required
                  disabled={inviting}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-60"
                  placeholder="member@company.com"
                />
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Role</label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value as Extract<AdminRole, 'admin' | 'operator'>)}
                  disabled={inviting}
                  className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-60"
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
                  className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-60"
                >
                  {inviting ? 'Inviting...' : 'Invite'}
                </button>
              </div>

              {activationLink && (
                <div className="rounded-lg border border-gray-200 bg-gray-50 p-3">
                  <p className="mb-2 text-sm font-medium text-gray-700">Activation Link</p>
                  <div className="flex gap-2">
                    <input
                      value={activationLink}
                      readOnly
                      className="w-full rounded-md border border-gray-300 bg-white px-2.5 py-2 text-xs text-gray-700"
                    />
                    <button
                      type="button"
                      onClick={() => void handleCopyActivationLink()}
                      className="rounded-md border border-gray-300 bg-white px-3 py-2 text-xs font-medium text-gray-700 hover:bg-gray-100"
                    >
                      Copy
                    </button>
                  </div>
                  {copySuccess && <p className="mt-2 text-xs text-gray-600">{copySuccess}</p>}
                </div>
              )}
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
