import { useEffect, useRef, useState } from 'react'
import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClick: () => void
  isEdited?: boolean
  onRecompute: () => void
  onDownload: () => void
  onDelete: () => void
}

export default function MemeCard({ meme, onClick, isEdited, onRecompute, onDownload, onDelete }: Props) {
  const isVideo = meme.type === 'video'
  const thumb = meme.thumbnail_url || meme.original_url
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
    <div className="relative w-full h-full group rounded-lg">
      <button
        onClick={onClick}
        className="w-full h-full overflow-hidden rounded-lg bg-gray-800
                   hover:ring-2 hover:ring-purple-500 transition-all focus:outline-none focus:ring-2 focus:ring-purple-500"
      >
        {thumb ? (
          <img
            src={thumb}
            alt={meme.caption || 'meme'}
            className="w-full h-full object-cover block"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-24 flex items-center justify-center text-gray-600 text-xs">
            no preview
          </div>
        )}
        {isVideo && (
          <span className="absolute bottom-1 left-1 bg-black/60 rounded px-1 py-0.5 text-xs text-white">
            ▶
          </span>
        )}
        {isEdited && (
          <span className="absolute top-1 left-1 bg-yellow-500/80 rounded px-1 py-0.5 text-xs text-black font-medium">
            Edited
          </span>
        )}
      </button>

      <div ref={menuRef} className="absolute top-1 right-1 z-10">
        <button
          onClick={e => { e.stopPropagation(); setMenuOpen(v => !v) }}
          className="opacity-0 group-hover:opacity-100 transition-opacity
                     bg-black/70 hover:bg-black/90 text-white rounded px-2.5 py-1 text-base leading-none"
          title="More options"
        >
          ···
        </button>

        {menuOpen && (
          <div className="absolute top-6 right-0 w-36 bg-gray-800 border border-gray-700 rounded-lg shadow-xl overflow-hidden">
            <button
              onClick={() => { setMenuOpen(false); onClick() }}
              className="w-full text-left px-3 py-2 text-sm text-white hover:bg-gray-700"
            >
              Open
            </button>
            <button
              onClick={() => { setMenuOpen(false); onRecompute() }}
              className="w-full text-left px-3 py-2 text-sm text-white hover:bg-gray-700"
            >
              Recompute
            </button>
            <button
              onClick={() => { setMenuOpen(false); onDownload() }}
              className="w-full text-left px-3 py-2 text-sm text-white hover:bg-gray-700"
            >
              Download
            </button>
            <button
              onClick={() => { setMenuOpen(false); onDelete() }}
              className="w-full text-left px-3 py-2 text-sm text-red-400 hover:bg-gray-700"
            >
              Delete
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
