import { useState, useCallback, useEffect } from 'react'
import { useShallow } from 'zustand/react/shallow'
import NavBar, {Tab} from './components/NavBar'
// import BottomNav from './components/BottomNav'
import MemeGrid from './components/MemeGrid'
import DetailsDialog from './components/DetailsDialog'
import UploadDialog from './components/UploadDialog'
import DeleteConfirmDialog from './components/DeleteConfirmDialog'
import ProgressDialog from './components/ProgressDialog'
import { Meme, deleteMeme, recomputeMeme, apiLogout } from './api/client'
import { useMediaStore } from './store/mediaStore'
import { useAuthStore } from './store/authStore'

const _cfg = (window as Window & { __MEMELO_CONFIG__?: { baseUrl?: string } }).__MEMELO_CONFIG__
const BASE = _cfg?.baseUrl ? _cfg.baseUrl.replace(/\/$/, '') + '/' : ''
const MEDIA_PAGE_PATH = `${BASE}media`

function mediaUrl(query?: string, id?: string): string {
  const params = new URLSearchParams()
  if (query) params.set('q', query)
  if (id) params.set('id', id)
  return `${MEDIA_PAGE_PATH}${params.size ? '?' + params : ''}`
}

type MobileTab = 'gallery' | 'upload'

export default function MediaPage() {
  const [query, setQuery] = useState(() => new URLSearchParams(window.location.search).get('q') ?? '')
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [notFound, setNotFound] = useState(false)
  const [confirmDeleteMeme, setConfirmDeleteMeme] = useState<Meme | null>(null)
  const [recomputingMeme, setRecomputingMeme] = useState<Meme | null>(null)
  const [navTab, setNavTab] = useState<Tab>('gallery')
  const [mobileTab, setMobileTab] = useState<MobileTab>('gallery')

  const { memes, hasMore, prependMeme, getById, setDetachedMeme, removeMeme, updateMeme } = useMediaStore(
    useShallow(s => ({
      memes: s.memes,
      hasMore: s.hasMore,
      prependMeme: s.prependMeme,
      getById: s.getById,
      setDetachedMeme: s.setDetachedMeme,
      removeMeme: s.removeMeme,
      updateMeme: s.updateMeme,
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

  // Sync mobile upload tab → open dialog
  useEffect(() => {
    if (mobileTab === 'upload') {
      setUploadOpen(true)
      setMobileTab('gallery')
    }
  }, [mobileTab])

  const handleQueryChange = useCallback((q: string) => {
    setQuery(q)
    window.history.replaceState({ query: q }, '', mediaUrl(q))
  }, [])

  const handleUploaded = useCallback((meme: Meme) => prependMeme(meme), [prependMeme])

  const handleRecompute = useCallback(async (meme: Meme) => {
    setRecomputingMeme(meme)
    try {
      const updated = await recomputeMeme(meme.id)
      updateMeme(updated)
    } catch (err) {
      console.error('recompute failed', err)
    }
    setRecomputingMeme(null)
  }, [updateMeme])

  const handleDownload = useCallback((meme: Meme) => {
    window.open(meme.original_url, '_blank')
  }, [])

  const handleDelete = useCallback((meme: Meme) => {
    setConfirmDeleteMeme(meme)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    if (!confirmDeleteMeme) return
    try {
      await deleteMeme(confirmDeleteMeme.id)
      removeMeme(confirmDeleteMeme.id)
    } catch (err) {
      console.error('delete failed', err)
    }
    setConfirmDeleteMeme(null)
  }, [confirmDeleteMeme, removeMeme])

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

  const handleUploadClick = useCallback(() => setUploadOpen(true), [])

  const handleLogout = useCallback(async () => {
    await apiLogout()
    useAuthStore.getState().logout()
    window.location.replace('/login')
  }, [])

  return (
    <div className="min-h-screen bg-[#111111] text-white flex flex-col">
      <NavBar
        activeTab={navTab}
        onTabChange={setNavTab}
        query={query}
        onQueryChange={handleQueryChange}
        onUploadClick={handleUploadClick}
        onLogout={handleLogout}
      />

      <main className="flex-1 p-4">
        <MemeGrid
          query={query}
          onSelect={handleSelect}
          onCardDownload={handleDownload}
          onCardRecompute={handleRecompute}
          onCardDelete={handleDelete}
        />
      </main>

      {/* <BottomNav activeTab={mobileTab} onTabChange={setMobileTab} /> */}

      {uploadOpen && (
        <UploadDialog
          onClose={() => setUploadOpen(false)}
          onUploaded={handleUploaded}
        />
      )}

      {selectedIndex !== null && (
        <DetailsDialog
          index={selectedIndex}
          onClose={() => {
            setSelectedIndex(null)
            window.history.pushState({ query }, '', mediaUrl(query))
          }}
          onPrev={selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex < memes.length - 1 || hasMore ? handleNext : undefined}
          onRecompute={() => {
            const s = useMediaStore.getState()
            const meme = selectedIndex >= 0 ? s.memes[selectedIndex] : s.detachedMeme
            if (meme) void handleRecompute(meme)
          }}
          onDownload={() => {
            const s = useMediaStore.getState()
            const meme = selectedIndex >= 0 ? s.memes[selectedIndex] : s.detachedMeme
            if (meme) handleDownload(meme)
          }}
          onDelete={() => {
            const s = useMediaStore.getState()
            const meme = selectedIndex >= 0 ? s.memes[selectedIndex] : s.detachedMeme
            if (meme) handleDelete(meme)
          }}
        />
      )}

      {notFound && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 bg-[#1e1e1e] border border-white/10
                        text-white px-4 py-2.5 rounded-xl shadow-xl z-50 flex items-center gap-3 text-sm">
          Meme not found
          <button onClick={() => setNotFound(false)} className="text-gray-500 hover:text-white transition-colors">✕</button>
        </div>
      )}

      {confirmDeleteMeme && (
        <DeleteConfirmDialog
          onClose={() => setConfirmDeleteMeme(null)}
          onConfirm={handleConfirmDelete}
        />
      )}

      {recomputingMeme && <ProgressDialog title="Recomputing…" message="This may take a moment." />}
    </div>
  )
}
