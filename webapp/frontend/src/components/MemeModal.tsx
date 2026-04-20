import { useEffect, useState, useMemo } from 'react'
import { Meme } from '../api/client'
import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  meme: Meme
  onClose: () => void
  onPrev?: () => void
  onNext?: () => void
}

function computeMediaHeight(meme: Meme): number | null {
  const srcW = meme.original_w || meme.thumbnail_w
  const srcH = meme.original_h || meme.thumbnail_h
  if (!srcW || !srcH) return null
  const dialogW = Math.min(window.innerWidth - 64, 768)
  const maxH = window.innerHeight * 0.6
  return Math.round(Math.min(dialogW / (srcW / srcH), maxH))
}

export default function MemeModal({ meme, onClose, onPrev, onNext }: Props) {
  const isVideo = meme.type === 'video'
  const [loaded, setLoaded] = useState(isVideo)
  const mediaHeight = useMemo(() => computeMediaHeight(meme), [meme.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!isVideo) setLoaded(false)
  }, [meme.id, isVideo]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') onPrev?.()
      if (e.key === 'ArrowRight') onNext?.()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onPrev, onNext])

  const mediaStyle = mediaHeight ? { height: mediaHeight } : {}

  return (
    <Modal onClose={onClose}>
      {onPrev && (
        <button
          onClick={e => { e.stopPropagation(); onPrev() }}
          className="fixed left-3 top-1/2 -translate-y-1/2 z-[60] flex items-center justify-center
                     w-11 h-11 rounded-full bg-black/60 hover:bg-purple-600/80 backdrop-blur-sm
                     text-white transition-all duration-150 hover:scale-110 focus:outline-none"
          aria-label="Previous"
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M13 4l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
      )}
      {onNext && (
        <button
          onClick={e => { e.stopPropagation(); onNext() }}
          className="fixed right-3 top-1/2 -translate-y-1/2 z-[60] flex items-center justify-center
                     w-11 h-11 rounded-full bg-black/60 hover:bg-purple-600/80 backdrop-blur-sm
                     text-white transition-all duration-150 hover:scale-110 focus:outline-none"
          aria-label="Next"
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M7 4l6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
      )}
      <Dialog onClose={onClose} className="max-w-3xl w-full max-h-[90vh] flex flex-col">
        <div className="relative w-full bg-black flex items-center justify-center overflow-hidden" style={mediaStyle}>
          {!loaded && (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-900">
              <div className="w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full animate-spin" />
            </div>
          )}
          {isVideo ? (
            <video
              key={meme.id}
              src={meme.original_url}
              controls
              autoPlay
              className="w-full h-full object-contain"
            />
          ) : (
            <img
              key={meme.id}
              src={meme.original_url || meme.thumbnail_url}
              alt={meme.caption || 'meme'}
              className="w-full h-full object-contain"
              style={{ opacity: loaded ? 1 : 0 }}
              onLoad={() => setLoaded(true)}
            />
          )}
        </div>

        <div className="p-4 space-y-2 overflow-y-auto">
          {meme.caption && (
            <p className="font-semibold text-white">{meme.caption}</p>
          )}
          {meme.ocr_result && (
            <p className="text-sm text-gray-400 italic">{meme.ocr_result}</p>
          )}
          {meme.tags && meme.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {meme.tags.map(t => (
                <span key={t} className="text-xs bg-gray-700 rounded px-2 py-0.5 text-gray-300">
                  {t}
                </span>
              ))}
            </div>
          )}
        </div>
      </Dialog>
    </Modal>
  )
}