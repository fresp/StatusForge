import type { User, StoredUserProfile } from '../types'

const TOKEN_KEY = 'user_token'
const PROFILE_KEY = 'user_profile'

export function setAuthSession(token: string, user: User) {
  localStorage.setItem(TOKEN_KEY, token)
  const profile: StoredUserProfile = {
    id: user.id,
    username: user.username,
    email: user.email,
    role: user.role,
    mfaEnabled: user.mfaEnabled,
    mfaVerified: user.mfaVerified,
  }
  localStorage.setItem(PROFILE_KEY, JSON.stringify(profile))
}

export function updateStoredProfile(updates: Partial<StoredUserProfile>) {
  const profile = getStoredProfile()
  if (!profile) return
  localStorage.setItem(PROFILE_KEY, JSON.stringify({ ...profile, ...updates }))
}

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getStoredProfile(): StoredUserProfile | null {
  try {
    const raw = localStorage.getItem(PROFILE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as StoredUserProfile
  } catch {
    return null
  }
}

export function isAuthenticated(): boolean {
  return !!getStoredToken()
}

export function isMfaVerified(): boolean {
  const profile = getStoredProfile()
  return !!profile?.mfaVerified
}

export function requiresMfa(): boolean {
  const profile = getStoredProfile()
  return !!profile && !profile.mfaVerified
}

export function clearAuthSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(PROFILE_KEY)
}
