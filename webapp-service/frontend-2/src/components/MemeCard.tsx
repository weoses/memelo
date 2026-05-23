import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClick: () => void
}

export default function MemeCard({ meme, onClick }: Props) {
  const isVideo = meme.type === 'video'
  const thumb = meme.thumbnail_url || meme.original_url
  const hasCaption = Boolean(meme.caption)

  return (
    <button
      onClick={onClick}
      className="w-full h-full flex flex-col rounded-xl overflow-hidden bg-[#1a1a1a]
                 hover:ring-1 hover:ring-white/20 transition-all focus:outline-none focus:ring-1 focus:ring-white/30
                 text-left"
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
      </div>
      {hasCaption && (
        <div className="px-2 py-1.5 shrink-0">
          <p className="text-xs text-gray-400 truncate leading-snug">{meme.caption}</p>
        </div>
      )}
    </button>
  )
}
