import { useState, useCallback, useEffect } from 'react'
import { useShallow } from 'zustand/react/shallow'
import SearchBar from './components/SearchBar'
import MemeGrid from './components/MemeGrid'
import UploadButton from './components/UploadButton'
import UploadModal from './components/UploadModal'
import MemeModal from './components/MemeModal'
import { Meme } from './api/client'
import { useMediaStore } from './store/mediaStore'
const _cfg = (window as Window & { __MEMELO_CONFIG__?: { baseUrl?: string } }).__MEMELO_CONFIG__
const BASE = _cfg?.baseUrl ? _cfg.baseUrl.replace(/\/$/, '') + '/' : ''

const MEDIA_PAGE_PATH = `${BASE}media`

function mediaUrl(query?: string, id?: string): string {
  const params = new URLSearchParams()
  if (query) params.set('q', query)
  if (id) params.set('id', id)
  return `${MEDIA_PAGE_PATH}${params.size ? '?' + params : ''}`
}

export default function MediaPage() {
  const [query, setQuery] = useState(() => new URLSearchParams(window.location.search).get('q') ?? '')
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [notFound, setNotFound] = useState(false)

  const { memes, hasMore, prependMeme, getById, setDetachedMeme } = useMediaStore(
    useShallow(s => ({
      memes: s.memes,
      hasMore: s.hasMore,
      prependMeme: s.prependMeme,
      getById: s.getById,
      setDetachedMeme: s.setDetachedMeme,
    }))
  )

  useEffect(() => {
    if (window.location.pathname === '/') {
      window.history.replaceState(null, '', MEDIA_PAGE_PATH)
    }
    const id = new URLSearchParams(window.location.search).get('id')
    if (!id) return
    getById(id).then(meme => {
      if (!meme) { setNotFound(true); return }
      const idx = useMediaStore.getState().memes.findIndex(m => m.id === id)
      if (idx >= 0) {
        setSelectedIndex(idx)
      } else {
        setDetachedMeme(meme)
        setSelectedIndex(-1)
      }
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const handler = (e: PopStateEvent) => {
      const s = e.state
      if (s?.index !== undefined && s.index !== null) {
        setSelectedIndex(s.index)
        setQuery(s.query ?? '')
      } else {
        setSelectedIndex(null)
        setQuery(s?.query ?? '')
      }
    }
    window.addEventListener('popstate', handler)
    return () => window.removeEventListener('popstate', handler)
  }, [])

  const handleQueryChange = useCallback((q: string) => {
    setQuery(q)
    window.history.replaceState({ query: q }, '', mediaUrl(q))
  }, [])

  const handleUploaded = useCallback((meme: Meme) => prependMeme(meme), [prependMeme])

  const handleSelect = useCallback((index: number) => {
    const meme = useMediaStore.getState().memes[index]
    if (!meme) return
    setSelectedIndex(index)
    window.history.pushState({ index, query }, '', mediaUrl(query, meme.id))
  }, [query])

  const handlePrev = useCallback(() => {
    if (selectedIndex === null || selectedIndex <= 0) return
    const newIndex = selectedIndex - 1
    const meme = useMediaStore.getState().memes[newIndex]
    if (!meme) return
    setSelectedIndex(newIndex)
    window.history.replaceState({ index: newIndex, query }, '', mediaUrl(query, meme.id))
  }, [selectedIndex, query])

  const handleNext = useCallback(() => {
    if (selectedIndex === null) return
    const { memes: currentMemes, hasMore: currentHasMore, loadMore: doLoadMore } = useMediaStore.getState()
    const newIndex = selectedIndex + 1
    if (newIndex >= currentMemes.length - 3 && currentHasMore) void doLoadMore()
    if (newIndex >= currentMemes.length) return
    const meme = currentMemes[newIndex]
    setSelectedIndex(newIndex)
    window.history.replaceState({ index: newIndex, query }, '', mediaUrl(query, meme.id))
  }, [selectedIndex, query])

  return (
    <div className="min-h-screen bg-gray-950 text-white flex flex-col">
      <header className="sticky top-0 z-10 bg-gray-900 border-b border-gray-800 px-4 py-3 flex items-center gap-3">
        <h1 className="hidden sm:block text-lg font-bold text-purple-400 shrink-0">Memelo</h1>
        <SearchBar onQueryChange={handleQueryChange} initialValue={query} />
        <UploadButton onOpen={() => setUploadOpen(true)} />
      </header>
      <main className="flex-1 p-4">
        <MemeGrid query={query} onSelect={handleSelect} />
      </main>
      {uploadOpen && (
        <UploadModal
          onClose={() => setUploadOpen(false)}
          onUploaded={handleUploaded}
        />
      )}
      {selectedIndex !== null && (
        <MemeModal
          index={selectedIndex}
          onClose={() => {
            setSelectedIndex(null)
            window.history.pushState({ query }, '', mediaUrl(query))
          }}
          onPrev={selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex < memes.length - 1 || hasMore ? handleNext : undefined}
        />
      )}
      {notFound && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 bg-red-600 text-white px-4 py-2 rounded-lg shadow-lg z-50 flex items-center gap-2">
          Image not found
          <button onClick={() => setNotFound(false)} className="ml-2 text-white/70 hover:text-white">✕</button>
        </div>
      )}
    </div>
  )
}
