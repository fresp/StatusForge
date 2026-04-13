import React, { useEffect } from 'react'
import { X } from 'lucide-react'

type ModalSize = 'md' | 'lg'

interface ModalProps {
  title: string
  onClose: () => void
  children: React.ReactNode
  footer?: React.ReactNode
  size?: ModalSize
}

const SIZE_CLASS: Record<ModalSize, string> = {
  md: 'max-w-md',
  lg: 'max-w-lg',
}

export default function Modal({ title, onClose, children, footer, size = 'md' }: ModalProps) {
  useEffect(() => {
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    return () => {
      document.body.style.overflow = previousOverflow
    }
  }, [])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/45 p-4 backdrop-blur-sm">
      <div className={`w-full ${SIZE_CLASS[size]} flex max-h-[90vh] flex-col overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-2xl shadow-slate-900/10`}>
        <div className="flex flex-none items-center justify-between border-b border-slate-100 bg-slate-50/70 px-6 py-4">
          <h2 className="font-semibold tracking-tight text-slate-900">{title}</h2>
          <button type="button" onClick={onClose} className="rounded-full p-1 text-slate-400 transition-colors hover:bg-white hover:text-slate-600">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 flex-1 min-h-0 overflow-y-auto">{children}</div>

        {footer && <div className="flex-none border-t border-slate-100 bg-slate-50/70 px-6 py-4">{footer}</div>}
      </div>
    </div>
  )
}
