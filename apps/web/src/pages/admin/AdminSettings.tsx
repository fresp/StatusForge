import React, { useEffect, useState } from 'react'
import type {
  StatusPageSettings,
  StatusPageSettingsPatchRequest,
  StatusPageThemePresetSummary,
} from '../../types'
import api from '../../lib/api'
import { getThemePresets, loadThemePresetStylesheet, normalizeThemePresetSelection } from '../../lib/themePresetLoader'
import { CheckCircle, AlertTriangle, AlertCircle, XCircle, Wrench } from 'lucide-react'

const ADMIN_TITLE_SUFFIX = ' - Admin Panel'

const DEFAULT_SETTINGS: StatusPageSettings = {
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
    preset: 'default.css',
  },
  footer: {
    text: '',
    showPoweredBy: true,
  },
  customCss: '',
  updatedAt: new Date().toISOString(),
  createdAt: new Date().toISOString(),
}

function normalizeSettings(input: StatusPageSettings | null | undefined, presets: StatusPageThemePresetSummary[]): StatusPageSettings {
  if (!input) {
    return DEFAULT_SETTINGS
  }

  const normalizedPreset = normalizeThemePresetSelection(input.theme?.preset || '', presets)

  return {
    head: {
      title: input.head?.title ?? DEFAULT_SETTINGS.head.title,
      description: input.head?.description ?? DEFAULT_SETTINGS.head.description,
      keywords: input.head?.keywords ?? DEFAULT_SETTINGS.head.keywords,
      faviconUrl: input.head?.faviconUrl ?? DEFAULT_SETTINGS.head.faviconUrl,
      metaTags: input.head?.metaTags || {},
    },
    branding: {
      siteName: input.branding?.siteName ?? DEFAULT_SETTINGS.branding.siteName,
      logoUrl: input.branding?.logoUrl ?? '',
      backgroundImageUrl: input.branding?.backgroundImageUrl ?? '',
      heroImageUrl: input.branding?.heroImageUrl ?? '',
    },
    theme: {
      preset: normalizedPreset,
    },
    footer: {
      text: input.footer?.text ?? '',
      showPoweredBy: input.footer?.showPoweredBy ?? true,
    },
    customCss: input.customCss ?? '',
    updatedAt: input.updatedAt ?? new Date().toISOString(),
    createdAt: input.createdAt ?? new Date().toISOString(),
  }
}

function parseMetaTagsText(value: string): Record<string, string> {
  const lines = value
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)

  const tags: Record<string, string> = {}
  for (const line of lines) {
    const separatorIndex = line.indexOf(':')
    if (separatorIndex <= 0) {
      continue
    }
    const key = line.slice(0, separatorIndex).trim()
    const tagValue = line.slice(separatorIndex + 1).trim()
    if (!key) {
      continue
    }
    tags[key] = tagValue
  }
  return tags
}

function metaTagsToText(metaTags: Record<string, string>): string {
  return Object.entries(metaTags)
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n')
}

export default function AdminSettings() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [settings, setSettings] = useState<StatusPageSettings>(DEFAULT_SETTINGS)
  const [themePresets, setThemePresets] = useState<StatusPageThemePresetSummary[]>(() => getThemePresets().presets)
  const [themePresetNotice, setThemePresetNotice] = useState<string | null>(null)
  const [metaTagsText, setMetaTagsText] = useState('')

  const previewStyle: React.CSSProperties = {
    backgroundColor: 'var(--bg)',
    color: 'var(--text)',
    fontFamily: 'var(--font-family)',
    backgroundImage: settings.branding.backgroundImageUrl
      ? `linear-gradient(var(--bg-image-overlay), var(--bg-image-overlay)), url(${settings.branding.backgroundImageUrl})`
      : undefined,
    backgroundSize: settings.branding.backgroundImageUrl ? 'cover' : undefined,
    backgroundPosition: settings.branding.backgroundImageUrl ? 'center' : undefined,
  }

  async function loadSettings() {
    try {
      setLoading(true)
      setError(null)
      const settingsRes = await api.get<StatusPageSettings>('/settings/status-page')
      const { presets, hasErrors } = getThemePresets()
      setThemePresetNotice(hasErrors ? 'Some local theme files are invalid or missing fields. Falling back to default values.' : null)
      setThemePresets(presets)
      const normalized = normalizeSettings(settingsRes.data, presets)
      setSettings(normalized)
      setMetaTagsText(metaTagsToText(normalized.head.metaTags || {}))
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSettings()
  }, [])

  useEffect(() => {
    document.title = `${settings.head.title}${ADMIN_TITLE_SUFFIX}`
  }, [settings.head.title])

  useEffect(() => {
    if (themePresets.length === 0) {
      return
    }

    loadThemePresetStylesheet(settings.theme.preset, themePresets).catch(() => { })
  }, [settings.theme.preset, themePresets])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)

    try {
      const payload: StatusPageSettingsPatchRequest = {
        head: {
          title: settings.head.title,
          description: settings.head.description,
          keywords: settings.head.keywords,
          faviconUrl: settings.head.faviconUrl,
          metaTags: parseMetaTagsText(metaTagsText),
        },
        branding: {
          siteName: settings.branding.siteName,
          logoUrl: settings.branding.logoUrl,
          backgroundImageUrl: settings.branding.backgroundImageUrl,
          heroImageUrl: settings.branding.heroImageUrl,
        },
        theme: {
          preset: settings.theme.preset,
        },
        footer: {
          text: settings.footer.text,
          showPoweredBy: settings.footer.showPoweredBy,
        },
        customCss: settings.customCss,
      }

      const res = await api.patch<StatusPageSettings>('/settings/status-page', payload)
      const normalized = normalizeSettings(res.data, themePresets)
      setSettings(normalized)
      setMetaTagsText(metaTagsToText(normalized.head.metaTags || {}))
      setSuccess('Settings saved successfully')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <div className="p-8 text-sm text-gray-500">Loading settings...</div>
  }

  return (
    <div className="p-8 max-w-5xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Status Page Settings</h1>
        <p className="text-sm text-gray-500 mt-1">
          Configure SEO, branding, footer, theme, and custom CSS for the public status page.
        </p>
      </div>

      {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
      {themePresetNotice && <p className="mb-4 text-sm text-amber-700 bg-amber-50 rounded-lg px-3 py-2">{themePresetNotice}</p>}
      {success && <p className="mb-4 text-sm text-green-700 bg-green-50 rounded-lg px-3 py-2">{success}</p>}

      <form onSubmit={handleSave} className="space-y-6">
        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Head & SEO</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Page Title</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.head.title}
              onChange={(e) => setSettings(prev => ({ ...prev, head: { ...prev.head, title: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.head.description}
              onChange={(e) => setSettings(prev => ({ ...prev, head: { ...prev.head, description: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Keywords</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.head.keywords}
              onChange={(e) => setSettings(prev => ({ ...prev, head: { ...prev.head, keywords: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Favicon URL</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.head.faviconUrl}
              onChange={(e) => setSettings(prev => ({ ...prev, head: { ...prev.head, faviconUrl: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Additional Meta Tags</label>
            <textarea
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm min-h-[110px]"
              placeholder={'og:title: My Status Page\nog:site_name: Statora'}
              value={metaTagsText}
              onChange={(e) => setMetaTagsText(e.target.value)}
            />
            <p className="text-xs text-gray-500 mt-1">One tag per line using format: key: value</p>
          </div>
        </section>

        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Branding Assets</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Site Name</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.branding.siteName}
              onChange={(e) => setSettings(prev => ({ ...prev, branding: { ...prev.branding, siteName: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Logo URL</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.branding.logoUrl}
              onChange={(e) => setSettings(prev => ({ ...prev, branding: { ...prev.branding, logoUrl: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Background Image URL</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.branding.backgroundImageUrl}
              onChange={(e) => setSettings(prev => ({ ...prev, branding: { ...prev.branding, backgroundImageUrl: e.target.value } }))}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Hero Image URL</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.branding.heroImageUrl}
              onChange={(e) => setSettings(prev => ({ ...prev, branding: { ...prev.branding, heroImageUrl: e.target.value } }))}
            />
          </div>
        </section>

        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Visual Theme</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Preset</label>
            <select
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.theme.preset}
              onChange={(e) => {
                const selectedPreset = normalizeThemePresetSelection(e.target.value, themePresets)
                setSettings(prev => ({
                  ...prev,
                  theme: {
                    preset: selectedPreset,
                  },
                }))
              }}
            >
              {themePresets.map((preset) => (
                <option key={preset.key} value={preset.key}>{preset.label}</option>
              ))}
            </select>
          </div>

          <div className="rounded-xl border border-gray-200 p-4 space-y-3">
            <h3 className="text-sm font-semibold text-gray-900">Live Preview</h3>
            <div className="rounded-lg p-4" style={previewStyle}>
              <div
                className="rounded-lg p-3  items-center justify-between"
                style={{
                  backgroundColor: 'var(--status-operational)',
                  color: 'var(--on-primary)',
                  boxShadow: 'inset 0 -3px 0 var(--color-accent)',
                }}
              >
                <div className="flex items-center gap-2">
                  {settings.branding.logoUrl && <img src={settings.branding.logoUrl} alt="logo" className="w-6 h-6 rounded object-contain" />}
                  <span className="font-semibold">{settings.branding.siteName || 'Statora'}</span>
                </div>
                <div className="flex items-center gap-2 text-xs">
                  <CheckCircle className="w-4 h-4" />
                  <span>All systems operational</span>
                  <div className="flex items-center gap-3 text-xl">
                    {settings.branding.heroImageUrl && (
                      <img
                        src={settings.branding.heroImageUrl}
                        alt="hero"
                        className="w-full h-24 object-cover rounded-md mt-3 border"
                        style={{ borderColor: 'var(--hero-image-border)' }}
                      />
                    )}
                  </div>
                </div>
              </div>


            </div>
          </div>
        </section>

        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Footer & Custom CSS</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Footer Text</label>
            <input
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.footer.text}
              onChange={(e) => setSettings(prev => ({ ...prev, footer: { ...prev.footer, text: e.target.value } }))}
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              id="show-powered"
              type="checkbox"
              checked={settings.footer.showPoweredBy}
              onChange={(e) => setSettings(prev => ({ ...prev, footer: { ...prev.footer, showPoweredBy: e.target.checked } }))}
            />
            <label htmlFor="show-powered" className="text-sm text-gray-700">Show “Powered by” text</label>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Custom CSS</label>
            <textarea
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm min-h-[160px] font-mono"
              value={settings.customCss}
              onChange={(e) => setSettings(prev => ({ ...prev, customCss: e.target.value }))}
            />
          </div>
        </section>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-lg px-5 py-2 text-sm font-medium"
          >
            {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </form>
    </div>
  )
}
