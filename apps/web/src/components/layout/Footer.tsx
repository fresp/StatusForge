import { Link } from 'react-router-dom'

interface FooterProps {
  centerText?: string
  showPoweredBy?: boolean
  showHistoryLink?: boolean
}

export default function Footer({ centerText, showPoweredBy, showHistoryLink = true }: FooterProps) {
  const trimmedCenterText = centerText?.trim() ?? ''
  const hasCenterText = trimmedCenterText.length > 0

  return (
    <footer style={{ borderColor: 'var(--border)' }}>
      <div className="max-w-5xl mx-auto px-4 py-4">
        <div className="flex flex-col gap-3 text-sm sm:flex-row sm:items-center sm:justify-between">
          {showPoweredBy && (
            <div className="font-medium" style={{ color: 'var(--text-muted)' }}>
              Powered by{" "}
              <a href="https://github.com/fresp/Statora">
                Statora
              </a>
            </div>
          )}

          {hasCenterText && (
            <div className="text-sm" style={{ color: 'var(--text-muted)' }}>
              {trimmedCenterText}
            </div>
          )}

          {showHistoryLink && (
            <div>
              <Link
                to="/history"
                className="inline-flex items-center rounded-lg px-3 py-1.5 text-sm font-medium border transition-colors"
                style={{
                  borderColor: 'var(--border)',
                  color: 'var(--text)',
                  backgroundColor: 'var(--surface)',
                }}
              >
                View History
              </Link>
            </div>
          )}
        </div>
      </div>
    </footer>
  )
}
