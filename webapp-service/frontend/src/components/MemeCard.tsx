import { Meme } from '../api/client'

interface Props {
  meme: Meme
  onClick: () => void
  isEdited?: boolean
}

export default function MemeCard({ meme, onClick, isEdited }: Props) {
  const isVideo = meme.type === 'video'
  const thumb = meme.thumbnail_url || meme.original_url

  return (
    <button
      onClick={onClick}
      className="relative w-full h-full overflow-hidden rounded-lg bg-gray-800
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
        <span className="absolute top-1 right-1 bg-black/60 rounded px-1 py-0.5 text-xs text-white">
          ▶
        </span>
      )}
      {isEdited && (
        <span className="absolute top-1 left-1 bg-yellow-500/80 rounded px-1 py-0.5 text-xs text-black font-medium">
          Edited
        </span>
      )}

    </button>
  )
}