import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getSetupStatus, saveSetupConfig, validateMongoSetup } from '../../lib/api'
import type { DBEngine, SetupStatusResponse } from '../../types'

const DEFAULT_SQLITE_PATH = './data/statusforge.db'

export default function DatabaseSetup() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [validatingMongo, setValidatingMongo] = useState(false)
  const [status, setStatus] = useState<SetupStatusResponse | null>(null)
  const [engine, setEngine] = useState<DBEngine>('mongodb')
  const [mongoUri, setMongoUri] = useState('mongodb://localhost:27017')
  const [mongoDbName, setMongoDbName] = useState('statusplatform')
  const [sqlitePath, setSqlitePath] = useState(DEFAULT_SQLITE_PATH)
  const [error, setError] = useState('')
  const [mongoValidationMessage, setMongoValidationMessage] = useState('')
  const [mongoValidated, setMongoValidated] = useState(false)

  useEffect(() => {
    document.title = 'StatusForge - Initial Setup'
  }, [])

  useEffect(() => {
    const run = async () => {
      setLoading(true)
      setError('')
      try {
        const setup = await getSetupStatus()
        setStatus(setup)
        if (setup.setupDone) {
          navigate('/admin/login', { replace: true })
          return
        }

        if (setup.engine === 'sqlite') {
          setEngine('sqlite')
        } else {
          setEngine('mongodb')
        }
      } catch (err: any) {
        setError(err?.response?.data?.error || err?.message || 'Failed to load setup status')
      } finally {
        setLoading(false)
      }
    }

    void run()
  }, [navigate])

  const canSubmit = useMemo(() => {
    if (engine === 'mongodb') {
      return mongoUri.trim().length > 0 && mongoDbName.trim().length > 0
    }
    return sqlitePath.trim().length > 0
  }, [engine, mongoUri, mongoDbName, sqlitePath])

  async function handleValidateMongo() {
    setMongoValidationMessage('')
    setError('')
    setMongoValidated(false)
    setValidatingMongo(true)
    try {
      const result = await validateMongoSetup(mongoUri.trim(), mongoDbName.trim())
      if (result.valid) {
        setMongoValidated(true)
        setMongoValidationMessage('MongoDB connection is valid.')
      }
    } catch (err: any) {
      setMongoValidationMessage(err?.response?.data?.error || err?.message || 'MongoDB validation failed')
    } finally {
      setValidatingMongo(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return

    setSaving(true)
    setError('')
    setMongoValidationMessage('')

    try {
      const payload =
        engine === 'mongodb'
          ? {
              engine,
              mongoUri: mongoUri.trim(),
              mongoDbName: mongoDbName.trim(),
            }
          : {
              engine,
              sqlitePath: sqlitePath.trim(),
            }

      const saved = await saveSetupConfig(payload)
      setStatus(saved)
      if (saved.setupDone) {
        navigate('/admin/login', { replace: true })
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to save setup configuration')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="text-sm text-gray-600">Loading setup status...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center px-4 py-8">
      <div className="w-full max-w-xl bg-white rounded-xl shadow-md p-8">
        <h1 className="text-2xl font-bold text-gray-900">Database Setup</h1>
        <p className="text-sm text-gray-600 mt-2 mb-6">
          Choose your database engine to finish first-run configuration.
        </p>

        {status && (
          <div className="mb-4 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-700">
            Current engine: <span className="font-medium">{status.engine || 'not set'}</span>
          </div>
        )}

        {error && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSave} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Database Engine</label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => {
                  setEngine('mongodb')
                  setMongoValidationMessage('')
                  setMongoValidated(false)
                }}
                className={`rounded-lg border px-4 py-3 text-left transition-colors ${
                  engine === 'mongodb'
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300'
                }`}
              >
                <div className="font-medium">MongoDB</div>
                <div className="text-xs mt-1">Full runtime support</div>
              </button>
              <button
                type="button"
                onClick={() => {
                  setEngine('sqlite')
                  setMongoValidationMessage('')
                  setMongoValidated(false)
                }}
                className={`rounded-lg border px-4 py-3 text-left transition-colors ${
                  engine === 'sqlite'
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300'
                }`}
              >
                <div className="font-medium">SQLite</div>
                <div className="text-xs mt-1">Local file-based setup</div>
              </button>
            </div>
          </div>

          {engine === 'mongodb' ? (
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Mongo URI</label>
                <input
                  type="text"
                  value={mongoUri}
                  onChange={(e) => {
                    setMongoUri(e.target.value)
                    setMongoValidated(false)
                    setMongoValidationMessage('')
                  }}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="mongodb://localhost:27017"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Mongo Database Name</label>
                <input
                  type="text"
                  value={mongoDbName}
                  onChange={(e) => {
                    setMongoDbName(e.target.value)
                    setMongoValidated(false)
                    setMongoValidationMessage('')
                  }}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="statusplatform"
                  required
                />
              </div>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => void handleValidateMongo()}
                  disabled={validatingMongo}
                  className="px-3 py-2 rounded-lg text-sm border border-gray-300 text-gray-700 hover:bg-gray-50 disabled:opacity-60"
                >
                  {validatingMongo ? 'Validating...' : 'Validate MongoDB'}
                </button>
                {mongoValidationMessage && (
                  <span className={`text-sm ${mongoValidated ? 'text-green-600' : 'text-red-600'}`}>
                    {mongoValidationMessage}
                  </span>
                )}
              </div>
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">SQLite File Path</label>
              <input
                type="text"
                value={sqlitePath}
                onChange={(e) => setSqlitePath(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder={DEFAULT_SQLITE_PATH}
                required
              />
            </div>
          )}

          <button
            type="submit"
            disabled={saving || !canSubmit}
            className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg text-sm transition-colors"
          >
            {saving ? 'Saving Setup...' : 'Save and Continue'}
          </button>
        </form>
      </div>
    </div>
  )
}
