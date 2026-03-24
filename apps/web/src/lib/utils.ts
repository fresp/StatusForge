import type { ComponentStatus } from '../types'
import type { Incident } from '../types'

export const STATUS_LABELS: Record<ComponentStatus, string> = {
  operational: 'Operational',
  degraded_performance: 'Degraded Performance',
  partial_outage: 'Partial Outage',
  major_outage: 'Major Outage',
  maintenance: 'Under Maintenance',
}

export const STATUS_COLORS: Record<ComponentStatus, string> = {
  operational: 'bg-green-500',
  degraded_performance: 'bg-yellow-400',
  partial_outage: 'bg-orange-500',
  major_outage: 'bg-red-500',
  maintenance: 'bg-blue-500',
}

export const STATUS_TEXT_COLORS: Record<ComponentStatus, string> = {
  operational: 'text-green-600',
  degraded_performance: 'text-yellow-600',
  partial_outage: 'text-orange-600',
  major_outage: 'text-red-600',
  maintenance: 'text-blue-600',
}

export const STATUS_BG_LIGHT: Record<ComponentStatus, string> = {
  operational: 'bg-green-50 border-green-200',
  degraded_performance: 'bg-yellow-50 border-yellow-200',
  partial_outage: 'bg-orange-50 border-orange-200',
  major_outage: 'bg-red-50 border-red-200',
  maintenance: 'bg-blue-50 border-blue-200',
}

export const INCIDENT_STATUS_LABELS: Record<string, string> = {
  investigating: 'Investigating',
  identified: 'Identified',
  monitoring: 'Monitoring',
  resolved: 'Resolved',
}

export const INCIDENT_IMPACT_LABELS: Record<string, string> = {
  none: 'None',
  minor: 'Minor',
  major: 'Major',
  critical: 'Critical',
}

export function getOverallStatusLabel(status: ComponentStatus): string {
  const labels: Record<ComponentStatus, string> = {
    operational: 'All Systems Operational',
    degraded_performance: 'Degraded System Performance',
    partial_outage: 'Partial System Outage',
    major_outage: 'Major System Outage',
    maintenance: 'System Under Maintenance',
  }
  return labels[status] || 'Unknown Status'
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function formatDateShort(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}

function formatLocalDateKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')

  return `${year}-${month}-${day}`
}

function toDateKey(dateStr: string): string {
  return formatLocalDateKey(new Date(dateStr))
}

function getQuarterLabel(monthIndex: number): 'Q1' | 'Q2' | 'Q3' | 'Q4' {
  if (monthIndex <= 2) return 'Q1'
  if (monthIndex <= 5) return 'Q2'
  if (monthIndex <= 8) return 'Q3'
  return 'Q4'
}

export interface RecentIncidentDayGroup {
  dateKey: string
  incidents: Incident[]
}

export interface IncidentStatusGroup {
  key: string
  label: string
  incidents: Incident[]
}

export interface IncidentDateGroup {
  date: string
  incidents: Incident[]
}

export interface HistoryMonthGroup {
  monthIndex: number
  monthLabel: string
  incidents: Incident[]
}

export interface HistoryQuarterGroup {
  quarter: 'Q1' | 'Q2' | 'Q3' | 'Q4'
  months: HistoryMonthGroup[]
}

export interface HistoryYearGroup {
  year: number
  quarters: HistoryQuarterGroup[]
}

export function getRecentDateKeys(days: number): string[] {
  const keys: string[] = []
  const today = new Date()

  for (let offset = 0; offset < days; offset += 1) {
    const cursor = new Date(today)
    cursor.setHours(0, 0, 0, 0)
    cursor.setDate(today.getDate() - offset)
    keys.push(formatLocalDateKey(cursor))
  }

  return keys
}

export function groupIncidentsByRecentDays(incidents: Incident[], days = 7): RecentIncidentDayGroup[] {
  const incidentMap = new Map<string, Incident[]>()

  incidents.forEach((incident) => {
    const key = toDateKey(incident.createdAt)
    const current = incidentMap.get(key) ?? []
    current.push(incident)
    incidentMap.set(key, current)
  })

  return getRecentDateKeys(days).map((dateKey) => ({
    dateKey,
    incidents: (incidentMap.get(dateKey) ?? []).sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    ),
  }))
}

export function groupIncidentsByStatus(incidents: Incident[]): IncidentStatusGroup[] {
  const groupOrder = ['investigating', 'identified', 'monitoring', 'resolved']
  const groups = new Map<string, Incident[]>()

  incidents.forEach((incident) => {
    const current = groups.get(incident.status) ?? []
    current.push(incident)
    groups.set(incident.status, current)
  })

  return groupOrder
    .filter((status) => (groups.get(status) ?? []).length > 0)
    .map((status) => {
      const groupedIncidents = (groups.get(status) ?? []).sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      )

      return {
        key: status,
        label: `${INCIDENT_STATUS_LABELS[status] ?? status} Incidents`,
        incidents: groupedIncidents,
      }
    })
}

export function groupIncidentsByDate(incidents: Incident[]): IncidentDateGroup[] {
  const incidentMap = new Map<string, Incident[]>()

  incidents.forEach((incident) => {
    const key = toDateKey(incident.createdAt)
    const current = incidentMap.get(key) ?? []
    current.push(incident)
    incidentMap.set(key, current)
  })

  return Array.from(incidentMap.entries())
    .sort(([dateA], [dateB]) => dateB.localeCompare(dateA))
    .map(([date, groupedIncidents]) => ({
      date,
      incidents: groupedIncidents.sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      ),
    }))
}

export function groupIncidentsByYearQuarterMonth(incidents: Incident[]): HistoryYearGroup[] {
  const byYear = new Map<number, Incident[]>()

  incidents.forEach((incident) => {
    const year = new Date(incident.createdAt).getFullYear()
    const current = byYear.get(year) ?? []
    current.push(incident)
    byYear.set(year, current)
  })

  const sortedYears = Array.from(byYear.keys()).sort((a, b) => b - a)

  return sortedYears.map((year) => {
    const yearIncidents = byYear.get(year) ?? []

    const quarterMap = new Map<'Q1' | 'Q2' | 'Q3' | 'Q4', HistoryMonthGroup[]>()
    quarterMap.set('Q1', [])
    quarterMap.set('Q2', [])
    quarterMap.set('Q3', [])
    quarterMap.set('Q4', [])

    for (let monthIndex = 0; monthIndex < 12; monthIndex += 1) {
      const monthIncidents = yearIncidents
        .filter((incident) => {
          const createdAt = new Date(incident.createdAt)
          return createdAt.getMonth() === monthIndex
        })
        .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())

      const monthLabel = new Date(year, monthIndex, 1).toLocaleString('en-US', { month: 'long' })
      const quarter = getQuarterLabel(monthIndex)
      const quarterMonths = quarterMap.get(quarter) ?? []

      quarterMonths.push({
        monthIndex,
        monthLabel,
        incidents: monthIncidents,
      })

      quarterMap.set(quarter, quarterMonths)
    }

    return {
      year,
      quarters: [
        { quarter: 'Q1', months: quarterMap.get('Q1') ?? [] },
        { quarter: 'Q2', months: quarterMap.get('Q2') ?? [] },
        { quarter: 'Q3', months: quarterMap.get('Q3') ?? [] },
        { quarter: 'Q4', months: quarterMap.get('Q4') ?? [] },
      ],
    }
  })
}
