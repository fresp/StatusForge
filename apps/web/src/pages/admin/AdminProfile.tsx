import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Shield, Key, RefreshCw, AlertCircle, CheckCircle, Loader2, User, Lock } from 'lucide-react'
import api from '../../lib/api'
import { setAuthSession, updateStoredProfile } from '../../lib/auth'
import type { AuthMeResponse, MfaSetupResponse, MfaVerifyResponse, MfaVerifyRequest, ProfileUpdateRequest } from '../../types'

interface FormState {
  username: string
  email: string
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

interface MfaSetupState {
  secret: string
  otpauthUrl: string
  recoveryCodes: string[]
  verificationCode: string
  recoveryCode: string
}

type MfaStep = 'idle' | 'setup-pending' | 'setup-show-secret' | 'verify-pending' | 'enabled' | 'disable-pending'

export default function AdminProfile() {
  const navigate = useNavigate()
  
  const [profile, setProfile] = useState<AuthMeResponse | null>(null)
  const [formState, setFormState] = useState<FormState>({
    username: '',
    email: '',
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  
  const [mfaStep, setMfaStep] = useState<MfaStep>('idle')
  const [mfaSetup, setMfaSetup] = useState<MfaSetupState>({
    secret: '',
    otpauthUrl: '',
    recoveryCodes: [],
    verificationCode: '',
    recoveryCode: '',
  })
  
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [mfaError, setMfaError] = useState<string | null>(null)
  const [mfaSuccess, setMfaSuccess] = useState<string | null>(null)
  const [disablePassword, setDisablePassword] = useState('')
  const [disableCode, setDisableCode] = useState('')

  React.useEffect(() => {
    fetchProfile()
  }, [])

  async function fetchProfile() {
    try {
      setLoading(true)
      setError(null)
      const res = await api.get<AuthMeResponse>('/auth/me')
      setProfile(res.data)
      setFormState((prev: FormState) => ({
        ...prev,
        username: res.data.username,
        email: res.data.email,
        currentPassword: '',
      }))
      setMfaStep(res.data.mfaEnabled ? 'enabled' : 'idle')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load profile')
    } finally {
      setLoading(false)
    }
  }

  async function handleProfileUpdate(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)

    try {
      const updateData: ProfileUpdateRequest = {
        username: formState.username,
      }

      if (formState.newPassword) {
        if (formState.newPassword !== formState.confirmPassword) {
          throw new Error('Passwords do not match')
        }
        if (formState.newPassword.length < 8) {
          throw new Error('Password must be at least 8 characters')
        }
        if (!formState.currentPassword) {
          throw new Error('Current password is required to set a new password')
        }
        updateData.currentPassword = formState.currentPassword
        updateData.newPassword = formState.newPassword
      }

      if (formState.username === profile?.username && !formState.newPassword) {
        setSuccess('No changes to save')
        setSaving(false)
        return
      }

      await api.patch('/auth/me', updateData)
      setSuccess('Profile updated successfully')
      setFormState((prev: FormState) => ({
        ...prev,
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      }))
      await fetchProfile()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update profile')
    } finally {
      setSaving(false)
    }
  }

  async function handleMfaSetup() {
    try {
      setMfaStep('setup-pending')
      setMfaError(null)
      const res = await api.post<MfaSetupResponse>('/auth/mfa/setup')
      setMfaSetup((prev: MfaSetupState) => ({
        ...prev,
        secret: res.data.secret,
        otpauthUrl: res.data.otpauthUrl,
        recoveryCodes: res.data.recoveryCodes,
      }))
      setMfaStep('setup-show-secret')
      setMfaSuccess('MFA setup initiated. Scan the QR code or enter the secret manually.')
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : 'Failed to setup MFA')
      setMfaStep('idle')
    }
  }

  async function handleMfaVerify(e: React.FormEvent) {
    e.preventDefault()
    setMfaStep('verify-pending')
    setMfaError(null)
    setMfaSuccess(null)

    try {
      let res
      if (mfaSetup.verificationCode) {
        const verifyData: MfaVerifyRequest = { code: mfaSetup.verificationCode }
        res = await api.post<MfaVerifyResponse>('/auth/mfa/verify', verifyData)
      } else if (mfaSetup.recoveryCode) {
        const verifyData: MfaVerifyRequest = { code: mfaSetup.recoveryCode }
        res = await api.post<MfaVerifyResponse>('/auth/mfa/recovery/verify', verifyData)
      } else {
        throw new Error('Enter either a verification code or recovery code')
      }

      setAuthSession(res.data.token, {
        ...res.data.user,
        mfaEnabled: true,
        mfaVerified: res.data.mfaVerified,
      })
      updateStoredProfile({ mfaEnabled: true, mfaVerified: res.data.mfaVerified })
      setMfaStep('enabled')
      setMfaSetup((prev: MfaSetupState) => ({
        ...prev,
        verificationCode: '',
        recoveryCode: '',
      }))
      setMfaSuccess(profile?.mfaEnabled
        ? 'MFA verified successfully.'
        : 'MFA enabled successfully! Store your recovery codes in a safe place.')
      await fetchProfile()
      navigate('/admin')
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : 'Invalid verification code')
      setMfaStep(profile?.mfaEnabled ? 'enabled' : 'setup-show-secret')
    }
  }

  async function handleMfaDisable() {
    try {
      setMfaStep('disable-pending')
      setMfaError(null)
      if (!disablePassword || !disableCode) {
        throw new Error('Current password and MFA code are required to disable MFA')
      }
      await api.post('/auth/mfa/disable', {
        password: disablePassword,
        code: disableCode,
      })
      updateStoredProfile({ mfaEnabled: false, mfaVerified: false })
      setDisablePassword('')
      setDisableCode('')
      setMfaSetup((prev: MfaSetupState) => ({ ...prev, verificationCode: '', recoveryCode: '' }))
      setMfaStep('idle')
      setMfaSuccess('MFA disabled successfully')
      await fetchProfile()
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : 'Failed to disable MFA')
      setMfaStep('enabled')
    }
  }

  function copyToClipboard(text: string, label: string) {
    navigator.clipboard.writeText(text)
    setMfaSuccess(`${label} copied to clipboard`)
    setTimeout(() => setMfaSuccess(null), 3000)
  }

  const isVerifying = mfaStep === 'verify-pending' || mfaStep === 'setup-pending'

  if (loading) {
    return (
      <div className="p-8 flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    )
  }

  return (
    <div className="p-8 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">My Profile</h1>
        <p className="text-sm text-gray-500 mt-1">
          Manage your account details, password, and multi-factor authentication.
        </p>
      </div>

      {/* General notifications */}
      {error && (
        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-red-900">Error</p>
            <p className="text-sm text-red-700 mt-0.5">{error}</p>
          </div>
        </div>
      )}
      {success && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg flex items-start gap-3">
          <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-green-900">Success</p>
            <p className="text-sm text-green-700 mt-0.5">{success}</p>
          </div>
        </div>
      )}

      {/* Account Settings */}
      <div className="bg-white rounded-xl border border-gray-200 p-6 mb-6">
        <div className="flex items-center gap-3 mb-6">
          <User className="w-5 h-5 text-gray-600" />
          <h2 className="text-lg font-semibold text-gray-900">Account Settings</h2>
        </div>

        <form onSubmit={handleProfileUpdate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Username
            </label>
            <input
              type="text"
              value={formState.username}
              onChange={(e) => setFormState(prev => ({ ...prev, username: e.target.value }))}
              placeholder="Enter your display username"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              type="email"
              value={formState.email}
              disabled
              className="w-full px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-gray-500 text-sm cursor-not-allowed"
            />
            <p className="text-xs text-gray-400 mt-1">Email cannot be changed</p>
          </div>

           <div className="grid grid-cols-2 gap-4">
             <div>
               <label className="block text-sm font-medium text-gray-700 mb-1">
                 Current Password
               </label>
               <input
                type="password"
                value={formState.currentPassword}
                onChange={(e) => setFormState(prev => ({ ...prev, currentPassword: e.target.value }))}
                placeholder="Only needed when changing password"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
              <p className="text-xs text-gray-500 mt-1">
                Leave this blank unless you are setting a new password.
              </p>
             </div>
           </div>

           <div className="grid grid-cols-2 gap-4">
             <div>
               <label className="block text-sm font-medium text-gray-700 mb-1">
                 New Password (optional)
              </label>
              <input
                type="password"
                value={formState.newPassword}
                onChange={(e) => setFormState(prev => ({ ...prev, newPassword: e.target.value }))}
                placeholder="Leave blank to keep current"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Confirm Password
              </label>
              <input
                type="password"
                value={formState.confirmPassword}
                onChange={(e) => setFormState(prev => ({ ...prev, confirmPassword: e.target.value }))}
                placeholder="Confirm new password"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
          </div>

          <div className="pt-4">
            <button
              type="submit"
              disabled={saving}
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
                {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                {saving ? 'Saving Changes...' : 'Save Profile Changes'}
              </button>
            </div>
          </form>
      </div>

      {/* MFA Settings */}
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <div className="flex items-center gap-3 mb-6">
          <Shield className="w-5 h-5 text-gray-600" />
          <h2 className="text-lg font-semibold text-gray-900">Multi-Factor Authentication</h2>
        </div>

        {/* MFA notifications */}
        {mfaError && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-900">MFA Error</p>
              <p className="text-sm text-red-700 mt-0.5">{mfaError}</p>
            </div>
          </div>
        )}
        {mfaSuccess && (
          <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-green-900">MFA Status</p>
              <p className="text-sm text-green-700 mt-0.5">{mfaSuccess}</p>
            </div>
          </div>
        )}

        {/* MFA not enabled - show setup CTA */}
        {!profile?.mfaEnabled && mfaStep === 'idle' && (
          <div className="text-center py-8">
            <div className="w-16 h-16 bg-blue-50 rounded-full flex items-center justify-center mx-auto mb-4">
              <Shield className="w-8 h-8 text-blue-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-2">Secure Your Account</h3>
            <p className="text-sm text-gray-500 mb-6 max-w-md mx-auto">
              Add an authenticator app to protect this account with a time-based verification code.
              After setup, every password login will require a second factor.
            </p>
            <button
              onClick={handleMfaSetup}
              className="px-6 py-2.5 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 inline-flex items-center gap-2"
            >
              <Key className="w-4 h-4" />
              Enable MFA
            </button>
          </div>
        )}

        {/* MFA setup in progress - show secret */}
        {mfaStep === 'setup-show-secret' && (
          <div className="space-y-6">
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-blue-900">Setup Instructions</p>
                  <ol className="text-sm text-blue-700 mt-2 space-y-1 list-decimal list-inside">
                    <li>Open your authenticator app (Google Authenticator, Authy, etc.)</li>
                    <li>Manually enter the secret below if QR scanning is unavailable</li>
                    <li>Save your recovery codes before finishing setup</li>
                    <li>Enter the current 6-digit code from the app to complete enrollment</li>
                  </ol>
                </div>
              </div>
            </div>

            {/* Secret display */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                TOTP Secret
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={mfaSetup.secret}
                  readOnly
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg bg-gray-50 text-sm font-mono"
                />
                <button
                  onClick={() => copyToClipboard(mfaSetup.secret, 'Secret')}
                  className="px-3 py-2 text-sm text-blue-600 hover:bg-blue-50 rounded-lg border border-gray-300"
                >
                  Copy
                </button>
              </div>
            </div>

            {/* Recovery codes */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Recovery Codes
              </label>
              <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
                <p className="text-xs text-gray-600 mb-3">
                  Store these codes in a safe place. Each code can only be used once.
                </p>
                <div className="grid grid-cols-2 gap-2">
                  {mfaSetup.recoveryCodes.map((code, idx) => (
                    <div
                      key={idx}
                      className="px-3 py-2 bg-white border border-gray-200 rounded text-xs font-mono text-center"
                    >
                      {code}
                    </div>
                  ))}
                </div>
                <button
                  onClick={() => copyToClipboard(mfaSetup.recoveryCodes.join('\n'), 'Recovery codes')}
                  className="mt-3 px-3 py-1.5 text-xs text-blue-600 hover:bg-blue-50 rounded border border-gray-300"
                >
                  Copy All Codes
                </button>
              </div>
            </div>

            {/* Verification form */}
            <form onSubmit={handleMfaVerify} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Verification Code
                </label>
                <input
                  type="text"
                  value={mfaSetup.verificationCode}
                  onChange={(e) => setMfaSetup(prev => ({ ...prev, verificationCode: e.target.value }))}
                  placeholder="Enter 6-digit code from authenticator app"
                  maxLength={6}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>
              <div className="flex items-center gap-3">
                <button
                  type="submit"
                  disabled={isVerifying}
                  className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  {isVerifying ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                  Verify and Enable
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setMfaStep('idle')
                    setMfaSetup({
                      secret: '',
                      otpauthUrl: '',
                      recoveryCodes: [],
                      verificationCode: '',
                      recoveryCode: '',
                    })
                    setMfaError(null)
                    setMfaSuccess(null)
                  }}
                  className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg border border-gray-300"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}

        {profile?.mfaEnabled && !profile.mfaVerified && (
          <div className="space-y-6">
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <AlertCircle className="w-5 h-5 text-amber-600 flex-shrink-0 mt-0.5" />
                <div>
                 <p className="text-sm font-medium text-amber-900">MFA verification required</p>
                  <p className="text-sm text-amber-700 mt-1">
                    Your password was accepted, but this session is still restricted.
                    Enter a code from your authenticator app to finish signing in, or use a recovery code if needed.
                  </p>
                </div>
              </div>
            </div>

            <form onSubmit={handleMfaVerify} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Authenticator Code
                </label>
                <input
                  type="text"
                  value={mfaSetup.verificationCode}
                  onChange={(e) => setMfaSetup(prev => ({ ...prev, verificationCode: e.target.value, recoveryCode: '' }))}
                  placeholder="Enter 6-digit code"
                  maxLength={6}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Recovery Code
                </label>
                <input
                  type="text"
                  value={mfaSetup.recoveryCode}
                  onChange={(e) => setMfaSetup(prev => ({ ...prev, recoveryCode: e.target.value, verificationCode: '' }))}
                  placeholder="Enter single-use recovery code"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm font-mono"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Use a recovery code only if you cannot access your authenticator app. Each code works once.
                </p>
              </div>

              <button
                type="submit"
                disabled={isVerifying}
                className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isVerifying ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
                Verify MFA
              </button>
            </form>
          </div>
        )}

        {/* MFA enabled - show status and disable option */}
        {profile?.mfaEnabled && profile.mfaVerified && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 p-4 bg-green-50 border border-green-200 rounded-lg">
              <CheckCircle className="w-6 h-6 text-green-600 flex-shrink-0" />
              <div>
                <p className="text-sm font-medium text-green-900">MFA is Enabled</p>
                <p className="text-sm text-green-700 mt-0.5">
                  Your account is protected with multi-factor authentication and requires a second factor at login.
                </p>
              </div>
            </div>

            <div className="pt-4 border-t border-gray-200">
              <h3 className="text-sm font-medium text-gray-900 mb-3">Disable MFA</h3>
              <p className="text-sm text-gray-500 mb-4">
                To disable MFA, confirm your current password and provide a current authenticator code.
                This removes the extra login protection from your account.
              </p>
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Current Password
                  </label>
                  <input
                    type="password"
                    value={disablePassword}
                    onChange={(e) => setDisablePassword(e.target.value)}
                    placeholder="Confirm your password"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Authenticator Code
                  </label>
                  <input
                    type="text"
                    value={disableCode}
                    onChange={(e) => setDisableCode(e.target.value)}
                    placeholder="Enter 6-digit code"
                    maxLength={10}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent text-sm"
                  />
                </div>
              </div>
              <button
                onClick={handleMfaDisable}
                disabled={mfaStep === 'disable-pending'}
                className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-lg hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {mfaStep === 'disable-pending' ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
                Disable MFA Protection
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
