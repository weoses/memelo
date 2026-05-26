import { useRef, useState, useEffect } from 'react'
import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClick: () => void
  onDownload?: () => void
  onRecompute?: () => void
  onDelete?: () => void
}

export default function MemeCard({ meme, onClick, onDownload, onRecompute, onDelete }: Props) {
  const isVideo = meme.type === 'video'
  const thumb = meme.thumbnail_url || meme.original_url
  const hasCaption = Boolean(meme.caption)
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  return (
    <div
      onClick={onClick}
      className="group w-full h-full flex flex-col rounded-xl overflow-hidden bg-[#1a1a1a]
                 hover:ring-1 hover:ring-white/20 transition-all cursor-pointer"
    >
      <div className="relative flex-1 overflow-hidden" style={{ minHeight: 0 }}>
        {thumb ? (
          <img
            src={thumb}
            alt={meme.caption || 'meme'}
            className="w-full h-full object-cover block"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-gray-600 text-xs">
            no preview
          </div>
        )}
        {isVideo && (
          <span className="absolute bottom-1.5 left-1.5 bg-black/70 rounded px-1.5 py-0.5 text-xs text-white leading-none">
            ▶
          </span>
        )}
        {meme.edited && (
          <span className="absolute top-1.5 left-1.5 bg-white/10 backdrop-blur-sm border border-white/20
                           rounded px-1.5 py-0.5 text-xs text-white/80 leading-none">
            Edited
          </span>
        )}

        {/* Three-dot menu */}
        <div
          ref={menuRef}
          className="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={e => e.stopPropagation()}
        >
          <button
            onClick={() => setMenuOpen(o => !o)}
            className="w-6 h-6 flex items-center justify-center rounded-md bg-black/60 backdrop-blur-sm
                       text-white/80 hover:text-white hover:bg-black/80 transition-colors text-sm leading-none"
          >
            ···
          </button>
          {menuOpen && (
            <div className="absolute right-0 top-full mt-1 w-36 bg-[#1e1e1e] border border-white/10
                            rounded-xl shadow-2xl overflow-hidden z-20">
              <button
                onClick={() => { onDownload?.(); setMenuOpen(false) }}
                className="w-full text-left px-3.5 py-2 text-sm text-gray-300 hover:bg-white/5 transition-colors"
              >
                Download
              </button>
              <button
                onClick={() => { onRecompute?.(); setMenuOpen(false) }}
                className="w-full text-left px-3.5 py-2 text-sm text-gray-300 hover:bg-white/5 transition-colors"
              >
                Recompute
              </button>
              <button
                onClick={() => { onDelete?.(); setMenuOpen(false) }}
                className="w-full text-left px-3.5 py-2 text-sm text-red-400 hover:bg-white/5 transition-colors"
              >
                Delete
              </button>
            </div>
          )}
        </div>
      </div>
      {hasCaption && (
        <div className="px-2 py-1.5 shrink-0">
          <p className="text-xs text-gray-400 truncate leading-snug">{meme.caption}</p>
        </div>
      )}
    </div>
  )
}
