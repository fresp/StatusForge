import type { StatusPageThemePresetSummary } from '../types'

export const DEFAULT_THEME_PRESET = 'default.css'
export const THEME_STYLESHEET_ID = 'status-page-theme-stylesheet'

const THEME_FILE_PATTERN = /^[a-z0-9-]+\.css$/

type ThemePresetCatalog = {
  presets: StatusPageThemePresetSummary[]
  hasErrors: boolean
}

type ThemeLoadResult = {
  appliedPreset: string
  usedFallback: boolean
}

function toLabel(themeFile: string): string {
  const withoutExt = themeFile.replace(/\.css$/i, '')
  return withoutExt
    .split('-')
    .filter(Boolean)
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join(' ')
}

function findPresetByKey(presets: StatusPageThemePresetSummary[], key: string): StatusPageThemePresetSummary | null {
  return presets.find((preset) => preset.key === key) || null
}

function normalizeThemeFileKey(input: string): string {
  const trimmed = input.trim().toLowerCase()
  if (!trimmed) {
    return DEFAULT_THEME_PRESET
  }

  if (trimmed.endsWith('.css')) {
    return trimmed
  }

  return `${trimmed}.css`
}

function getPresetHrefMap(): Record<string, string> {
  const modules = import.meta.glob('../themes/*.css', {
    eager: true,
    import: 'default',
    query: '?url',
  }) as Record<string, string>

  const hrefByFile: Record<string, string> = {}
  for (const [path, href] of Object.entries(modules)) {
    if (typeof href !== 'string') {
      continue
    }

    const fileName = path.split('/').pop() || ''
    const normalized = normalizeThemeFileKey(fileName)
    if (!THEME_FILE_PATTERN.test(normalized)) {
      continue
    }

    hrefByFile[normalized] = href
  }

  return hrefByFile
}

export function getThemePresets(): ThemePresetCatalog {
  const hrefByFile = getPresetHrefMap()
  const presets = Object.keys(hrefByFile)
    .filter((fileName) => THEME_FILE_PATTERN.test(fileName))
    .map<StatusPageThemePresetSummary>((fileName) => ({
      key: fileName,
      label: toLabel(fileName),
    }))
    .sort((a, b) => {
      if (a.key === DEFAULT_THEME_PRESET) return -1
      if (b.key === DEFAULT_THEME_PRESET) return 1
      return a.label.localeCompare(b.label)
    })

  return {
    presets,
    hasErrors: !hrefByFile[DEFAULT_THEME_PRESET],
  }
}

type ThemeCssVariables = Record<`--${string}`, string>

type ThemeResolvedPalette = {
  primaryColor: string
  backgroundColor: string
  textColor: string
  accentColor: string
}

type ThemeResolved = {
  light: ThemeResolvedPalette
  dark: ThemeResolvedPalette
  typography: {
    fontFamily: string
    fontScale: 'sm' | 'md' | 'lg'
  }
  ui: {
    cardBackground: string
    borderColor: string
  }
}

export function buildThemeCssVariables(resolved: ThemeResolved, mode: 'light' | 'dark'): ThemeCssVariables {
  const palette = mode === 'dark' ? resolved.dark : resolved.light

  return {
    '--primary': palette.primaryColor,
    '--bg': palette.backgroundColor,
    '--text': palette.textColor,
    '--color-accent': palette.accentColor,
    '--font-family': resolved.typography.fontFamily,
    '--surface': resolved.ui.cardBackground,
    '--border': resolved.ui.borderColor,
    '--on-primary': mode === 'dark' ? '#052e16' : '#f0fdf4',
    '--on-primary-subtle': mode === 'dark' ? '#dcfce7' : '#166534',
    '--text-muted': mode === 'dark' ? '#9ca3af' : '#6b7280',
    '--text-subtle': mode === 'dark' ? '#6b7280' : '#9ca3af',
    '--surface-incident': mode === 'dark' ? '#450a0a' : '#fef2f2',
    '--border-incident': mode === 'dark' ? '#7f1d1d' : '#fecaca',
    '--surface-maintenance': mode === 'dark' ? '#1e1b4b' : '#eef2ff',
    '--border-maintenance': mode === 'dark' ? '#312e81' : '#c7d2fe',
    '--surface-uptime': mode === 'dark' ? '#0f172a' : '#f8fafc',
    '--hero-image-border': mode === 'dark' ? '#334155' : '#e2e8f0',
    '--subcomponent-divider': mode === 'dark' ? '#1e293b' : '#f1f5f9',
    '--status-operational': '#16a34a',
    '--status-degraded': '#d97706',
    '--status-partial': '#ea580c',
    '--status-major': '#dc2626',
    '--status-maintenance': '#4f46e5',
    '--status-resolved-bg': mode === 'dark' ? '#064e3b' : '#d1fae5',
    '--status-resolved-text': mode === 'dark' ? '#6ee7b7' : '#059669',
    '--bg-image-overlay': mode === 'dark' ? 'rgba(0,0,0,0.7)' : 'rgba(255,255,255,0.5)',
  }
}

export function normalizeThemePresetSelection(requestedPreset: string, presets: StatusPageThemePresetSummary[]): string {
  const requested = normalizeThemeFileKey(requestedPreset)
  if (findPresetByKey(presets, requested)) {
    return requested
  }

  if (findPresetByKey(presets, DEFAULT_THEME_PRESET)) {
    return DEFAULT_THEME_PRESET
  }

  return presets[0]?.key || DEFAULT_THEME_PRESET
}

function ensureThemeStylesheetElement(): HTMLLinkElement {
  const existing = document.getElementById(THEME_STYLESHEET_ID)
  if (existing instanceof HTMLLinkElement) {
    return existing
  }

  const link = document.createElement('link')
  link.id = THEME_STYLESHEET_ID
  link.rel = 'stylesheet'
  document.head.appendChild(link)
  return link
}

function loadStylesheetHref(link: HTMLLinkElement, href: string, preset: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (link.getAttribute('href') === href && link.dataset.loadedPreset === preset) {
      resolve()
      return
    }

    const cleanup = () => {
      link.onload = null
      link.onerror = null
    }

    link.onload = () => {
      cleanup()
      link.dataset.loadedPreset = preset
      resolve()
    }

    link.onerror = () => {
      cleanup()
      reject(new Error(`Failed to load theme stylesheet: ${preset}`))
    }

    link.setAttribute('href', href)
  })
}

export async function loadThemePresetStylesheet(
  requestedPreset: string,
  presets: StatusPageThemePresetSummary[],
): Promise<ThemeLoadResult> {
  const hrefByFile = getPresetHrefMap()
  const normalizedRequested = normalizeThemePresetSelection(requestedPreset, presets)
  const defaultPreset = normalizeThemePresetSelection(DEFAULT_THEME_PRESET, presets)
  const link = ensureThemeStylesheetElement()

  const requestedHref = hrefByFile[normalizedRequested]
  if (requestedHref) {
    try {
      await loadStylesheetHref(link, requestedHref, normalizedRequested)
      return { appliedPreset: normalizedRequested, usedFallback: false }
    } catch {}
  }

  const fallbackHref = hrefByFile[defaultPreset]
  if (!fallbackHref) {
    return { appliedPreset: normalizedRequested, usedFallback: true }
  }

  await loadStylesheetHref(link, fallbackHref, defaultPreset)
  return { appliedPreset: defaultPreset, usedFallback: true }
}
