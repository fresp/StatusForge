import { Link } from 'react-router-dom'

export default function UnsupportedRuntime() {
  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center px-4 py-8">
      <div className="w-full max-w-xl bg-white rounded-xl shadow-md p-8">
        <h1 className="text-2xl font-bold text-gray-900">SQLite setup saved</h1>
        <p className="text-sm text-gray-600 mt-2">
          StatusForge saved your SQLite configuration, and the admin runtime is supported with SQLite.
        </p>
        <p className="text-sm text-gray-600 mt-3">
          You can continue using SQLite for your admin setup and monitoring workflow.
        </p>

        <div className="mt-6 flex flex-col sm:flex-row gap-3">
          <Link
            to="/admin/setup"
            className="inline-flex items-center justify-center rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Back to database setup
          </Link>
          <Link
            to="/"
            className="inline-flex items-center justify-center rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Open public status page
          </Link>
        </div>
      </div>
    </div>
  )
}
