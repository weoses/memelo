import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClick: () => void
}

export default function MemeCard({ meme, onClick }: Props) {
  const isVideo = meme.type === 'video'
  const thumb = meme.thumbnail_url || meme.original_url

  return (
    <button
      onClick={onClick}
      className="relative aspect-square overflow-hidden rounded-lg bg-gray-800
                 hover:ring-2 hover:ring-purple-500 transition-all focus:outline-none focus:ring-2 focus:ring-purple-500"
    >
      {thumb ? (
        <img
          src={thumb}
          alt={meme.caption || 'meme'}
          className="w-full h-full object-cover"
          loading="lazy"
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center text-gray-600 text-xs">
          no preview
        </div>
      )}
      {isVideo && (
        <span className="absolute bottom-1 right-1 bg-black/60 rounded px-1 py-0.5 text-xs text-white">
          ▶
        </span>
      )}
      {meme.caption && (
        <div className="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/70 p-1">
          <p className="text-xs text-white truncate">{meme.caption}</p>
        </div>
      )}
    </button>
  )
}
