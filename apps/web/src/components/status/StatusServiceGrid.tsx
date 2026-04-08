import React from 'react'
import { Activity, BarChart3, CheckCircle2, Layers3, Mail, MessageSquareMore, PackageCheck, PlugZap, RadioTower, Send, ShieldCheck, Siren, Wrench } from 'lucide-react'
import type { ComponentStatus, ComponentWithSubs } from '../../types'
import { STATUS_LABELS } from '../../lib/utils'
import UptimeStrip from './UptimeStrip'

type Props = {
  components: ComponentWithSubs[]
  onSelectComponent: (component: ComponentWithSubs) => void
}

function getStatusTone(status: ComponentStatus) {
  switch (status) {
    case 'major_outage':
      return {
        pillBackground: 'color-mix(in srgb, var(--status-major) 12%, transparent)',
        pillColor: 'var(--status-major)',
      }
    case 'partial_outage':
      return {
        pillBackground: 'color-mix(in srgb, var(--status-partial) 12%, transparent)',
        pillColor: 'var(--status-partial)',
      }
    case 'degraded_performance':
      return {
        pillBackground: 'color-mix(in srgb, var(--status-degraded) 12%, transparent)',
        pillColor: 'var(--status-degraded)',
      }
    case 'maintenance':
      return {
        pillBackground: 'color-mix(in srgb, var(--status-maintenance) 14%, transparent)',
        pillColor: 'var(--status-maintenance)',
      }
    default:
      return {
        pillBackground: 'var(--status-pill-bg, rgba(16,185,129,0.1))',
        pillColor: 'var(--status-pill-text, #14815f)',
      }
  }
}

function getStatusIcon(status: ComponentStatus) {
  switch (status) {
    case 'major_outage':
      return <Siren className="h-4 w-4" />
    case 'partial_outage':
      return <ShieldCheck className="h-4 w-4" />
    case 'degraded_performance':
      return <RadioTower className="h-4 w-4" />
    case 'maintenance':
      return <Wrench className="h-4 w-4" />
    default:
      return <CheckCircle2 className="h-4 w-4" />
  }
}

function getComponentIcon(name: string) {
  const normalized = name.toLowerCase()

  if (normalized.includes('message') || normalized.includes('whatsapp') || normalized.includes('chat')) {
    return MessageSquareMore
  }
  if (normalized.includes('mail') || normalized.includes('email')) {
    return Mail
  }
  if (normalized.includes('delivery') || normalized.includes('webhook')) {
    return Send
  }
  if (normalized.includes('dashboard')) {
    return BarChart3
  }
  if (normalized.includes('api') || normalized.includes('auth')) {
    return Activity
  }
  if (normalized.includes('platform')) {
    return Layers3
  }
  if (normalized.includes('monitor')) {
    return RadioTower
  }
  if (normalized.includes('status')) {
    return PackageCheck
  }
  return PlugZap
}

function StatusServiceCard({ component, onSelectComponent }: { component: ComponentWithSubs; onSelectComponent: (component: ComponentWithSubs) => void }) {
  const Icon = getComponentIcon(component.name)
  const tone = getStatusTone(component.status)

  return (
    <button
      type="button"
      onClick={() => onSelectComponent(component)}
      className="group flex h-full w-full flex-col rounded-[1.6rem] border p-6 text-left transition duration-200 hover:-translate-y-1"
      style={{
        backgroundColor: 'var(--status-card-bg, rgba(255,255,255,0.88))',
        borderColor: 'var(--status-card-border, rgba(181,197,226,0.32))',
        boxShadow: 'var(--status-card-shadow, 0 24px 60px rgba(20,37,63,0.08))',
      }}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 items-center gap-4">
          <div
            className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl"
            style={{ backgroundColor: 'var(--status-icon-surface, #eef4ff)', color: 'var(--primary)' }}
          >
            <Icon className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <h3 className="truncate text-2xl font-black tracking-[-0.03em]" style={{ color: 'var(--text)' }}>
              {component.name}
            </h3>
            {component.description && (
              <p className="mt-1 truncate text-sm" style={{ color: 'var(--text-subtle)' }}>
                {component.description}
              </p>
            )}
          </div>
        </div>
        <span
          className="inline-flex shrink-0 items-center gap-2 rounded-full px-3 py-1 text-[11px] font-black uppercase tracking-[0.22em]"
          style={{ backgroundColor: tone.pillBackground, color: tone.pillColor }}
        >
          {getStatusIcon(component.status)}
          {component.status === 'operational' ? 'Active' : STATUS_LABELS[component.status]}
        </span>
      </div>

      <div className="mt-8 space-y-4">
        {(component.subComponents.length > 0 ? component.subComponents : [{ id: component.id, name: component.name, status: component.status }]).map((subcomponent) => {
          const subTone = getStatusTone(subcomponent.status)
          return (
            <div key={subcomponent.id} className="flex items-center justify-between gap-3">
              <span className="min-w-0 truncate text-sm font-medium" style={{ color: 'var(--text-muted)' }}>
                {subcomponent.name}
              </span>
              <div className="flex shrink-0 items-center gap-2">
                <span className="text-xs font-semibold" style={{ color: subTone.pillColor }}>
                  {STATUS_LABELS[subcomponent.status]}
                </span>
                <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: subTone.pillColor }} />
              </div>
            </div>
          )
        })}
      </div>

      <div className="mt-8 border-t pt-6" style={{ borderColor: 'color-mix(in srgb, var(--status-card-border, rgba(181,197,226,0.32)) 80%, transparent)' }}>
        <UptimeStrip history={component.uptimeHistory} />
      </div>
    </button>
  )
}

export default function StatusServiceGrid({ components, onSelectComponent }: Props) {
  return (
    <section className="px-4 pt-8 md:px-6 md:pt-10">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 md:grid-cols-2 md:gap-7">
        {components.map((component) => (
          <StatusServiceCard key={component.id} component={component} onSelectComponent={onSelectComponent} />
        ))}
      </div>
    </section>
  )
}
