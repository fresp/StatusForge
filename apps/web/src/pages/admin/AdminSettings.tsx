import React, { useEffect, useState } from 'react'
import type { StatusPageSettings, StatusPageSettingsPatchRequest } from '../../types'
import api from '../../lib/api'

const DEFAULT_SETTINGS: StatusPageSettings = {
  head: {
    title: 'Status Platform',
    description: 'Live system status and incident updates.',
    keywords: 'status, uptime, incidents, maintenance',
    faviconUrl: '/vite.svg',
    metaTags: {},
  },
  branding: {
    siteName: 'System Status',
    logoUrl: '',
  },
  theme: {
    primaryColor: '#16a34a',
    backgroundColor: '#f9fafb',
    textColor: '#111827',
  },
  layout: {
    variant: 'classic',
  },
  footer: {
    text: '',
    showPoweredBy: true,
  },
  customCss: '',
  updatedAt: new Date().toISOString(),
  createdAt: new Date().toISOString(),
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
  const [metaTagsText, setMetaTagsText] = useState('')

  async function loadSettings() {
    try {
      setLoading(true)
      setError(null)
      const res = await api.get<StatusPageSettings>('/settings/status-page')
      setSettings(res.data)
      setMetaTagsText(metaTagsToText(res.data.head.metaTags || {}))
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSettings()
  }, [])

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
        },
        theme: {
          primaryColor: settings.theme.primaryColor,
          backgroundColor: settings.theme.backgroundColor,
          textColor: settings.theme.textColor,
        },
        layout: {
          variant: settings.layout.variant,
        },
        footer: {
          text: settings.footer.text,
          showPoweredBy: settings.footer.showPoweredBy,
        },
        customCss: settings.customCss,
      }

      const res = await api.patch<StatusPageSettings>('/settings/status-page', payload)
      setSettings(res.data)
      setMetaTagsText(metaTagsToText(res.data.head.metaTags || {}))
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
          Configure SEO, branding, layout, footer, theme, and custom CSS for the public status page.
        </p>
      </div>

      {error && <p className="mb-4 text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>}
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
              placeholder={'og:title: My Status Page\nog:site_name: StatusForge'}
              value={metaTagsText}
              onChange={(e) => setMetaTagsText(e.target.value)}
            />
            <p className="text-xs text-gray-500 mt-1">One tag per line using format: key: value</p>
          </div>
        </section>

        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Branding & Layout</h2>
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
            <label className="block text-sm font-medium text-gray-700 mb-1">Layout Variant</label>
            <select
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
              value={settings.layout.variant}
              onChange={(e) => setSettings(prev => ({ ...prev, layout: { variant: e.target.value as 'classic' | 'compact' } }))}
            >
              <option value="classic">Classic</option>
              <option value="compact">Compact</option>
            </select>
          </div>
        </section>

        <section className="bg-white border border-gray-200 rounded-xl p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Theme</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Primary Color</label>
              <input
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
                value={settings.theme.primaryColor}
                onChange={(e) => setSettings(prev => ({ ...prev, theme: { ...prev.theme, primaryColor: e.target.value } }))}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Background Color</label>
              <input
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
                value={settings.theme.backgroundColor}
                onChange={(e) => setSettings(prev => ({ ...prev, theme: { ...prev.theme, backgroundColor: e.target.value } }))}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Text Color</label>
              <input
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm"
                value={settings.theme.textColor}
                onChange={(e) => setSettings(prev => ({ ...prev, theme: { ...prev.theme, textColor: e.target.value } }))}
              />
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
