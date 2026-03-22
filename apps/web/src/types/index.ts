// API types for the status platform

export type ComponentStatus =
  | 'operational'
  | 'degraded_performance'
  | 'partial_outage'
  | 'major_outage'
  | 'maintenance'

export type IncidentStatus = 'investigating' | 'identified' | 'monitoring' | 'resolved'
export type IncidentImpact = 'none' | 'minor' | 'major' | 'critical'
export type MaintenanceStatus = 'scheduled' | 'in_progress' | 'completed'
export type MonitorType = 'http' | 'tcp' | 'dns' | 'ping' | 'ssl'
export type MonitorLogStatus = 'up' | 'down'

export interface Component {
  id: string
  name: string
  description: string
  status: ComponentStatus
  createdAt: string
  updatedAt: string
}

export interface SubComponent {
  id: string
  componentId: string
  name: string
  description: string
  status: ComponentStatus
}

export interface UptimeBar {
  date: string
  uptimePercent: number
  status: ComponentStatus
}

export interface ComponentWithSubs extends Component {
  subComponents: SubComponent[]
  uptimeHistory: UptimeBar[]
}

export interface Incident {
  id: string
  title: string
  description: string
  status: IncidentStatus
  impact: IncidentImpact
  creatorId?: string
  creatorUsername?: string
  affectedComponents: string[]
  createdAt: string
  updatedAt: string
  resolvedAt?: string
  updates?: IncidentUpdate[]
}

export interface IncidentUpdate {
  id: string
  incidentId: string
  message: string
  status: IncidentStatus
  createdAt: string
}

export interface Maintenance {
  id: string
  title: string
  description: string
  creatorId?: string
  creatorUsername?: string
  components: string[]
  startTime: string
  endTime: string
  status: MaintenanceStatus
}

export interface Monitor {
  id: string
  name: string
  type: MonitorType
  target: string
  sslThresholds?: number[]
  intervalSeconds: number
  timeoutSeconds: number
  componentId: string
  subComponentId?: string
  lastStatus?: MonitorLogStatus
  sslWarning?: boolean
  sslDaysRemaining?: number
  sslTriggeredThreshold?: number
  lastCheckedAt?: string
  createdAt: string
}

export interface MonitorLog {
  id: string
  monitorId: string
  status: MonitorLogStatus
  responseTime: number
  statusCode: number
  checkedAt: string
}

export interface DailyUptime {
  id: string
  monitorId: string
  date: string
  totalChecks: number
  successfulChecks: number
  uptimePercent: number
}

export interface Subscriber {
  id: string
  email: string
  verified: boolean
  createdAt: string
}

export interface StatusSummary {
  overallStatus: ComponentStatus
  componentCounts: Record<ComponentStatus, number>
  activeIncidents: number
  scheduledMaintenance: number
}

export interface User {
  id: string
  username: string
  email: string
  role: UserRole
  mfaEnabled?: boolean
  mfaVerified?: boolean
}

export type UserRole = 'admin' | 'operator'

export type UserStatus = 'active' | 'disabled' | 'invited'

export interface UserMember {
  id: string
  username: string
  email: string
  role: UserRole
  status: UserStatus
}

export interface UserInvitation {
  id: string
  email: string
  role: UserRole
  expiresAt: string
  createdAt: string
  isExpired: boolean
}

export interface LoginResponse {
  token: string
  user: User
  mfaRequired?: boolean
}

export interface AuthMeResponse {
  userId: string
  username: string
  email: string
  role: UserRole
  mfaEnabled: boolean
  mfaVerified: boolean
}

export interface StoredUserProfile {
  id: string
  username: string
  email: string
  role?: UserRole
  mfaEnabled?: boolean
  mfaVerified?: boolean
}

export interface MfaSetupResponse {
  secret: string
  otpauthUrl: string
  recoveryCodes: string[]
}

export interface MfaVerifyRequest {
  code: string
}

export interface MfaVerifyResponse {
  token: string
  mfaVerified: boolean
  user: User
}

export interface ProfileUpdateRequest {
  username: string
  currentPassword?: string
  newPassword?: string
}

export interface StatusPageSettings {
  head: {
    title: string
    description: string
    keywords: string
    faviconUrl: string
    metaTags: Record<string, string>
  }
  branding: {
    siteName: string
    logoUrl: string
  }
  theme: {
    primaryColor: string
    backgroundColor: string
    textColor: string
  }
  layout: {
    variant: 'classic' | 'compact'
  }
  footer: {
    text: string
    showPoweredBy: boolean
  }
  customCss: string
  updatedAt: string
  createdAt: string
}

export interface StatusPageSettingsPatchRequest {
  head?: {
    title?: string
    description?: string
    keywords?: string
    faviconUrl?: string
    metaTags?: Record<string, string>
  }
  branding?: {
    siteName?: string
    logoUrl?: string
  }
  theme?: {
    primaryColor?: string
    backgroundColor?: string
    textColor?: string
  }
  layout?: {
    variant?: 'classic' | 'compact'
  }
  footer?: {
    text?: string
    showPoweredBy?: boolean
  }
  customCss?: string
}
export interface WebhookChannel {
  id: string
  name: string
  url: string
  enabled: boolean
  createdAt: string
}
