import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../../lib/api'
import { clearAuthSession, setAuthSession } from '../../lib/auth'
import type { LoginResponse, MfaVerifyResponse, User } from '../../types'

export default function AdminLogin() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [mfaCode, setMfaCode] = useState('')
  const [mfaError, setMfaError] = useState('')
  const [mfaLoading, setMfaLoading] = useState(false)
  const [pendingToken, setPendingToken] = useState<string | null>(null)
  const [pendingUser, setPendingUser] = useState<User | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await api.post<LoginResponse>('/auth/login', { email, password })

      if (res.data.mfaRequired) {
        setPendingToken(res.data.token)
        setPendingUser(res.data.user)
        setMfaCode('')
        setMfaError('')
      } else {
        setAuthSession(res.data.token, {
          ...res.data.user,
          mfaVerified: true,
        })
        navigate('/admin')
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Invalid credentials')
    } finally {
      setLoading(false)
    }
  }

  async function handleMfaVerify(e: React.FormEvent) {
    e.preventDefault()
    if (!pendingToken) {
      setMfaError('Your MFA session expired. Please sign in again.')
      return
    }

    setMfaError('')
    setMfaLoading(true)

    try {
      const res = await api.post<MfaVerifyResponse>(
        '/auth/mfa/verify',
        { code: mfaCode },
        { headers: { Authorization: `Bearer ${pendingToken}` } }
      )

      setAuthSession(res.data.token, {
        ...res.data.user,
        mfaEnabled: true,
        mfaVerified: res.data.mfaVerified,
      })

      setPendingToken(null)
      setPendingUser(null)
      navigate('/admin')
    } catch (err: any) {
      setMfaError(err.response?.data?.error || 'Invalid verification code')
    } finally {
      setMfaLoading(false)
    }
  }

  function handleMfaCancel() {
    setPendingToken(null)
    setPendingUser(null)
    setMfaCode('')
    setMfaError('')
    clearAuthSession()
  }

  return (
    <>
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="bg-white rounded-xl shadow-md w-full max-w-sm p-8">
          <h1 className="text-2xl font-bold text-gray-900 mb-2">User Login</h1>
          <p className="text-sm text-gray-500 mb-6">Status Platform</p>

          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-3 text-sm">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                required
                value={email}
                onChange={e => setEmail(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="admin@statusplatform.com"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                required
                value={password}
                onChange={e => setPassword(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="••••••••"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg text-sm transition-colors"
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        </div>
      </div>

      {pendingToken && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center px-4">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-sm p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-1">MFA Verification Required</h2>
            <p className="text-sm text-gray-600 mb-4">
              Enter the 6-digit code from your authenticator app for {pendingUser?.email ?? 'your account'}.
            </p>

            {mfaError && (
              <div className="mb-4 bg-red-50 border border-red-200 text-red-700 rounded-lg px-3 py-2 text-sm">
                {mfaError}
              </div>
            )}

            <form onSubmit={handleMfaVerify} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Authenticator Code</label>
                <input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  required
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm tracking-widest focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="123456"
                />
              </div>

              <div className="flex items-center gap-3">
                <button
                  type="submit"
                  disabled={mfaLoading}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg text-sm transition-colors"
                >
                  {mfaLoading ? 'Verifying...' : 'Verify and Continue'}
                </button>
                <button
                  type="button"
                  onClick={handleMfaCancel}
                  disabled={mfaLoading}
                  className="px-3 py-2 rounded-lg text-sm border border-gray-300 text-gray-700 hover:bg-gray-50"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  )
}
