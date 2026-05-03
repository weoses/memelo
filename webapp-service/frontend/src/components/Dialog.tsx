import { ReactNode } from 'react'

interface Props {
  onClose: () => void
  title?: string
  canClose?: boolean
  children: ReactNode
  className?: string
}

export default function Dialog({ onClose, title, canClose = true, children, className = '' }: Props) {
  return (
    <div
      className={`relative bg-gray-900 rounded-xl shadow-2xl ${className}`}
      onClick={e => e.stopPropagation()}
    >
      <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
        {title ? <h2 className="text-base font-semibold text-white">{title}</h2> : <span />}
        {canClose && (
          <button
            onClick={onClose}
            aria-label="Close"
            className="text-gray-400 hover:text-white text-xl leading-none"
          >
            ×
          </button>
        )}
      </div>
      {children}
    </div>
  )
}