import { useEffect, ReactNode } from 'react'

interface Props {
  onClose: () => void
  canClose?: boolean
  children: ReactNode
}

export default function Modal({ onClose, canClose = true, children }: Props) {
  useEffect(() => {
    const html = document.documentElement
    const scrollbarWidth = window.innerWidth - html.clientWidth
    const prevOverflow = html.style.overflow
    const prevPadding = html.style.paddingRight
    html.style.overflow = 'hidden'
    html.style.paddingRight = `${scrollbarWidth}px`
    return () => {
      html.style.overflow = prevOverflow
      html.style.paddingRight = prevPadding
    }
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && canClose) onClose()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [canClose, onClose])

  return (
    <div
      className="fixed inset-0 z-50 bg-black/75 flex items-end sm:items-center justify-center sm:p-4"
      onClick={() => { if (canClose) onClose() }}
    >
      {children}
    </div>
  )
}