import { ReactNode, useRef, useEffect } from 'react'

interface Props {
  onClose: () => void
  title?: string
  canClose?: boolean
  children: ReactNode
  className?: string
  onSwipeLeft?: () => void
  onSwipeRight?: () => void
}

export default function Dialog({ onClose, title, canClose = true, children, className = '', onSwipeLeft, onSwipeRight }: Props) {
  const dragRef = useRef<HTMLDivElement>(null)
  const touchState = useRef<{
    startX: number
    startY: number
    atTop: boolean
    dir: 'none' | 'h' | 'v'
    dragging: boolean
  } | null>(null)

  // Non-passive so we can preventDefault and block iOS overscroll rubber-banding during drag
  useEffect(() => {
    const el = dragRef.current
    if (!el) return
    const onMove = (e: TouchEvent) => {
      if (!touchState.current) return
      const dx = e.touches[0].clientX - touchState.current.startX
      const dy = e.touches[0].clientY - touchState.current.startY
      const ax = Math.abs(dx), ay = Math.abs(dy)
      if (touchState.current.dir === 'none' && (ax > 8 || ay > 8))
        touchState.current.dir = ay > ax ? 'v' : 'h'
      if (touchState.current.dir === 'v' && dy > 0 && touchState.current.atTop && canClose) {
        e.preventDefault()
        touchState.current.dragging = true
        el.style.transform = `translateY(${dy}px)`
        el.style.opacity = String(Math.max(0, 1 - dy / 350))
      }
    }
    el.addEventListener('touchmove', onMove, { passive: false })
    return () => el.removeEventListener('touchmove', onMove)
  }, [canClose])

  const handleTouchStart = (e: React.TouchEvent) => {
    touchState.current = {
      startX: e.touches[0].clientX,
      startY: e.touches[0].clientY,
      atTop: (dragRef.current?.scrollTop ?? 0) === 0,
      dir: 'none',
      dragging: false,
    }
    if (dragRef.current) dragRef.current.style.transition = 'none'
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (!touchState.current) return
    const { startX, startY, dragging, dir } = touchState.current
    const dx = e.changedTouches[0].clientX - startX
    const dy = e.changedTouches[0].clientY - startY
    const ax = Math.abs(dx), ay = Math.abs(dy)
    touchState.current = null
    const el = dragRef.current
    if (dragging && el) {
      if (dy > 150) {
        el.style.transition = 'transform 0.25s ease, opacity 0.25s ease'
        el.style.transform = 'translateY(100%)'
        el.style.opacity = '0'
        setTimeout(onClose, 250)
      } else {
        el.style.transition = 'transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1), opacity 0.25s ease'
        el.style.transform = 'translateY(0)'
        el.style.opacity = '1'
      }
    } else if (dir === 'h' && ax > 50 && ax > ay) {
      if (dx < 0) onSwipeLeft?.()
      else onSwipeRight?.()
    }
  }

  return (
    <div
      ref={dragRef}
      className={`relative bg-gray-900 rounded-t-2xl sm:rounded-xl shadow-2xl ${className}`}
      onClick={e => e.stopPropagation()}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      <div className="sm:hidden flex justify-center pt-2">
        <div className="w-10 h-1 rounded-full bg-white/30" />
      </div>
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
