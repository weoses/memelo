import { useEffect } from 'react'
import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClose: () => void
}

export default function MemeModal({ meme, onClose }: Props) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onClose])

  const isVideo = meme.type === 'video'

  return (
    <div
      className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="relative max-w-3xl w-full max-h-[90vh] bg-gray-900 rounded-xl overflow-hidden flex flex-col"
        onClick={e => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          aria-label="Close"
          className="absolute top-2 right-2 z-10 text-gray-400 hover:text-white text-2xl leading-none w-8 h-8 flex items-center justify-center"
        >
          ×
        </button>

        {isVideo ? (
          <video
            src={meme.original_url}
            controls
            autoPlay
            className="w-full max-h-[60vh] bg-black"
          />
        ) : (
          <img
            src={meme.original_url || meme.thumbnail_url}
            alt={meme.caption || 'meme'}
            className="w-full max-h-[60vh] object-contain bg-black"
          />
        )}

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
      </div>
    </div>
  )
}
