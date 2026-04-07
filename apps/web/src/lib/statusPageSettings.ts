import type { StatusPageSettings } from '../types'
import { loadThemePresetStylesheet, getThemePresets, DEFAULT_THEME_PRESET } from './themePresetLoader'

export const STATUS_PAGE_SETTINGS_CACHE_KEY = 'Statora:status-page-settings'

export const DEFAULT_STATUS_PAGE_SETTINGS: StatusPageSettings = {
  head: {
    title: 'Statora',
    description: 'Live system status and incident updates.',
    keywords: 'status, uptime, incidents, maintenance',
    faviconUrl: '/vite.svg',
    metaTags: {},
  },
  branding: {
    siteName: 'Statora',
    logoUrl: '',
    backgroundImageUrl: '',
    heroImageUrl: '',
  },
  theme: {
    preset: DEFAULT_THEME_PRESET,
  },
  footer: {
    text: '',
    showPoweredBy: true,
  },
  customCss: '',
  updatedAt: '',
  createdAt: '',
}

let bootstrappedStatusPageSettings: StatusPageSettings | null = null

export function normalizeStatusPageSettings(settings?: Partial<StatusPageSettings> | null): StatusPageSettings {
  if (!settings) {
    return DEFAULT_STATUS_PAGE_SETTINGS
  }

  const preset = settings.theme?.preset?.trim() || DEFAULT_THEME_PRESET
  const normalizedPreset = preset.endsWith('.css') ? preset : `${preset}.css`

  return {
    head: {
      title: settings.head?.title ?? DEFAULT_STATUS_PAGE_SETTINGS.head.title,
      description: settings.head?.description ?? DEFAULT_STATUS_PAGE_SETTINGS.head.description,
      keywords: settings.head?.keywords ?? DEFAULT_STATUS_PAGE_SETTINGS.head.keywords,
      faviconUrl: settings.head?.faviconUrl ?? DEFAULT_STATUS_PAGE_SETTINGS.head.faviconUrl,
      metaTags: settings.head?.metaTags || {},
    },
    branding: {
      siteName: settings.branding?.siteName ?? DEFAULT_STATUS_PAGE_SETTINGS.branding.siteName,
      logoUrl: settings.branding?.logoUrl ?? '',
      backgroundImageUrl: settings.branding?.backgroundImageUrl ?? '',
      heroImageUrl: settings.branding?.heroImageUrl ?? '',
    },
    theme: {
      preset: normalizedPreset,
      appliedPreset: settings.theme?.appliedPreset,
      mode: settings.theme?.mode,
      overrides: settings.theme?.overrides,
      resolved: settings.theme?.resolved,
    },
    layout: settings.layout,
    footer: {
      text: settings.footer?.text ?? '',
      showPoweredBy: settings.footer?.showPoweredBy ?? true,
    },
    customCss: settings.customCss ?? '',
    updatedAt: settings.updatedAt ?? '',
    createdAt: settings.createdAt ?? '',
  }
}

function upsertMetaTag(selector: string, content: string) {
  const existing = document.head.querySelector(`meta[${selector}]`)
  if (content) {
    if (existing) {
      existing.setAttribute('content', content)
      return
    }

    const meta = document.createElement('meta')
    const [attr, value] = selector.split('=')
    meta.setAttribute(attr, value.replace(/"/g, ''))
    meta.setAttribute('content', content)
    document.head.appendChild(meta)
    return
  }

  if (existing) {
    existing.remove()
  }
}

function setCustomMetaTags(metaTags: Record<string, string>) {
  const existing = document.head.querySelectorAll('meta[data-status-page-meta="true"]')
  existing.forEach(node => node.remove())

  Object.entries(metaTags).forEach(([key, value]) => {
    if (!key || !value) {
      return
    }

    const meta = document.createElement('meta')
    if (key.startsWith('og:') || key.startsWith('twitter:')) {
      meta.setAttribute('property', key)
    } else {
      meta.setAttribute('name', key)
    }
    meta.setAttribute('content', value)
    meta.setAttribute('data-status-page-meta', 'true')
    document.head.appendChild(meta)
  })
}

function upsertFavicon(url: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = url
}

function upsertCustomCss(css: string) {
  const id = 'status-page-custom-css'
  let styleEl = document.getElementById(id) as HTMLStyleElement | null
  if (!styleEl) {
    styleEl = document.createElement('style')
    styleEl.id = id
    document.head.appendChild(styleEl)
  }
  styleEl.textContent = css
}

export function applyStatusPageDocumentSettings(settings: StatusPageSettings): void {
  if (typeof document === 'undefined') {
    return
  }

  document.title = settings.head.title
  upsertMetaTag('name="description"', settings.head.description)
  upsertMetaTag('name="keywords"', settings.head.keywords)
  setCustomMetaTags(settings.head.metaTags)
  upsertFavicon(settings.head.faviconUrl)
  upsertCustomCss(settings.customCss)
}

export function applyStatusPageThemePreset(settings: StatusPageSettings): void {
  const normalized = normalizeStatusPageSettings(settings)
  const presets = getThemePresets().presets
  loadThemePresetStylesheet(normalized.theme.preset, presets).catch(() => {})
}

export function applyStatusPageHeadSettings(settings: StatusPageSettings): void {
  applyStatusPageDocumentSettings(normalizeStatusPageSettings(settings))
}

export function readCachedStatusPageSettings(): StatusPageSettings | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const cached = window.localStorage.getItem(STATUS_PAGE_SETTINGS_CACHE_KEY)
    if (!cached) {
      return null
    }
    const parsed = JSON.parse(cached) as Partial<StatusPageSettings>
    return normalizeStatusPageSettings(parsed)
  } catch {
    return null
  }
}

export function cacheStatusPageSettings(settings: StatusPageSettings): void {
  if (typeof window === 'undefined') {
    return
  }

  const normalized = normalizeStatusPageSettings(settings)
  bootstrappedStatusPageSettings = normalized

  try {
    window.localStorage.setItem(STATUS_PAGE_SETTINGS_CACHE_KEY, JSON.stringify(normalized))
  } catch {
  }
}

export function getBootstrappedStatusPageSettings(): StatusPageSettings | null {
  return bootstrappedStatusPageSettings
}

export function parseStatusPageSettingsPayload(payload: unknown): StatusPageSettings | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  const candidate = payload as Partial<StatusPageSettings>
  if (!candidate.head || !candidate.branding || !candidate.theme || !candidate.footer) {
    return null
  }

  return normalizeStatusPageSettings(candidate)
}

export function preloadCachedStatusPageSettings(): StatusPageSettings | null {
  const cachedSettings = readCachedStatusPageSettings()
  if (!cachedSettings) {
    return null
  }

  bootstrappedStatusPageSettings = cachedSettings
  applyStatusPageDocumentSettings(cachedSettings)
  applyStatusPageThemePreset(cachedSettings)

  return cachedSettings
}
