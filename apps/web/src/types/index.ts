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
export type MonitorType = 'http' | 'tcp' | 'dns' | 'ping'
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
  intervalSeconds: number
  timeoutSeconds: number
  componentId: string
  subComponentId?: string
  lastStatus?: MonitorLogStatus
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

export interface Admin {
  id: string
  username: string
  email: string
}
