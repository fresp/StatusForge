import React from 'react'
import { Activity, CheckCircle2, AlertTriangle, ShieldAlert, Wrench } from 'lucide-react'
import type { ComponentStatus } from '../../types'

type Props = {
  status: ComponentStatus
  title: string
  description: string
  siteName: string
  activeIncidents: number
  scheduledMaintenance: number
}

function HeroStatusIcon({ status }: { status: ComponentStatus }) {
  const className = 'h-10 w-10 md:h-12 md:w-12'

  switch (status) {
    case 'degraded_performance':
      return <AlertTriangle className={className} />
    case 'partial_outage':
    case 'major_outage':
      return <ShieldAlert className={className} />
    case 'maintenance':
      return <Wrench className={className} />
    default:
      return <CheckCircle2 className={className} />
  }
}

export default function StatusHero({
  status,
  title,
  description,
  siteName,
  activeIncidents,
  scheduledMaintenance,
}: Props) {
  return (
    <section className="px-4 pt-5 md:px-6 md:pt-8">
      <div
        className="relative mx-auto max-w-6xl overflow-hidden rounded-[2rem] px-6 py-8 text-white shadow-[0_34px_80px_rgba(15,143,103,0.24)] md:px-10 md:py-12"
        style={{ backgroundImage: 'var(--hero-gradient)' }}
      >
        <div
          className="absolute right-0 top-0 h-52 w-52 -translate-y-1/3 translate-x-1/4 rounded-full blur-3xl md:h-72 md:w-72"
          style={{ backgroundColor: 'var(--hero-glow, rgba(255,255,255,0.18))' }}
        />
        <div className="relative z-10 flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
          <div className="max-w-3xl">
            <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/15 px-4 py-2 backdrop-blur-md">
              <span className="h-2.5 w-2.5 rounded-full bg-white animate-pulse" />
              <span className="text-[11px] font-extrabold uppercase tracking-[0.32em] text-white/90">
                {siteName} pulse
              </span>
            </div>
            <h2 className="max-w-3xl text-4xl font-black leading-[0.95] tracking-[-0.04em] md:text-6xl">
              {title}
            </h2>
            <p className="mt-5 max-w-2xl text-sm font-medium leading-6 text-white/88 md:text-lg md:leading-8">
              {description}
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <div className="inline-flex items-center gap-2 rounded-full bg-white/12 px-4 py-2 backdrop-blur-sm">
                <Activity className="h-4 w-4" />
                <span className="text-xs font-bold uppercase tracking-[0.22em]">{activeIncidents} active incidents</span>
              </div>
              <div className="inline-flex items-center gap-2 rounded-full bg-white/12 px-4 py-2 backdrop-blur-sm">
                <Wrench className="h-4 w-4" />
                <span className="text-xs font-bold uppercase tracking-[0.22em]">{scheduledMaintenance} planned maintenance</span>
              </div>
            </div>
          </div>

          <div className="hidden shrink-0 lg:block">
            <div className="flex h-40 w-40 items-center justify-center rounded-full border-4 border-white/15 bg-white/10 backdrop-blur-md xl:h-48 xl:w-48">
              <HeroStatusIcon status={status} />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
