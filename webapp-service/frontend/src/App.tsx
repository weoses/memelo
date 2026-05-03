import { useState, useCallback, useEffect } from 'react'
import SearchBar from './components/SearchBar'
import MemeGrid from './components/MemeGrid'
import UploadButton from './components/UploadButton'
import UploadModal from './components/UploadModal'
import MemeModal from './components/MemeModal'
import { Meme, getMeme } from './api/client'

export default function App() {
  const [query, setQuery] = useState('')
  const [initialQuery, setInitialQuery] = useState('')
  const [selected, setSelected] = useState<Meme | null>(null)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [memesList, setMemesList] = useState<Meme[]>([])
  const [uploadOpen, setUploadOpen] = useState(false)
  const [prependMeme, setPrependMeme] = useState<Meme | null>(null)
  const [updatedMeme, setUpdatedMeme] = useState<Meme | null>(null)
  const [notFound, setNotFound] = useState(false)

  useEffect(() => {
    if (window.location.pathname === '/') {
      window.history.replaceState(null, '', '/media')
    }
    const params = new URLSearchParams(window.location.search)
    const q = params.get('q') ?? ''
    const id = params.get('id')
    if (q) {
      setQuery(q)
      setInitialQuery(q)
    }
    if (id) {
      getMeme(id).then(meme => {
        if (meme) {
          setSelected(meme)
          setSelectedIndex(-1)
        } else {
          setNotFound(true)
        }
      })
    }
  }, [])

  useEffect(() => {
    const handler = (e: PopStateEvent) => {
      const s = e.state
      if (s?.meme) {
        setSelected(s.meme)
        setSelectedIndex(s.index ?? -1)
        setMemesList(s.list ?? [])
        setQuery(s.query ?? '')
      } else {
        setSelected(null)
        setSelectedIndex(-1)
        setQuery(s?.query ?? '')
      }
    }
    window.addEventListener('popstate', handler)
    return () => window.removeEventListener('popstate', handler)
  }, [])

  const handleQueryChange = useCallback((q: string) => {
    setQuery(q)
    setPrependMeme(null)
    const params = new URLSearchParams()
    if (q) params.set('q', q)
    window.history.replaceState({ query: q }, '', `/media${params.size ? '?' + params : ''}`)
  }, [])

  const handleUploaded = useCallback((meme: Meme) => setPrependMeme(meme), [])

  const handleSelect = useCallback((m: Meme, index: number, list: Meme[]) => {
    setSelected(m)
    setSelectedIndex(index)
    setMemesList(list)
    const params = new URLSearchParams()
    if (query) params.set('q', query)
    params.set('id', m.id)
    window.history.pushState({ meme: m, index, list, query }, '', `/media?${params}`)
  }, [query])

  const handleUpdate = useCallback((updated: Meme) => {
    setSelected(updated)
    setMemesList(prev => prev.map(m => m.id === updated.id ? updated : m))
    setUpdatedMeme(updated)
  }, [])

  const handlePrev = useCallback(() => {
    if (selectedIndex > 0) {
      const newIndex = selectedIndex - 1
      const newMeme = memesList[newIndex]
      setSelectedIndex(newIndex)
      setSelected(newMeme)
      const params = new URLSearchParams()
      if (query) params.set('q', query)
      params.set('id', newMeme.id)
      window.history.replaceState({ meme: newMeme, index: newIndex, list: memesList, query }, '', `/media?${params}`)
    }
  }, [selectedIndex, memesList, query])

  const handleNext = useCallback(() => {
    if (selectedIndex < memesList.length - 1) {
      const newIndex = selectedIndex + 1
      const newMeme = memesList[newIndex]
      setSelectedIndex(newIndex)
      setSelected(newMeme)
      const params = new URLSearchParams()
      if (query) params.set('q', query)
      params.set('id', newMeme.id)
      window.history.replaceState({ meme: newMeme, index: newIndex, list: memesList, query }, '', `/media?${params}`)
    }
  }, [selectedIndex, memesList, query])

  return (
    <div className="min-h-screen bg-gray-950 text-white flex flex-col">
      <header className="sticky top-0 z-10 bg-gray-900 border-b border-gray-800 px-4 py-3 flex items-center gap-3">
        <h1 className="hidden sm:block text-lg font-bold text-purple-400 shrink-0">Memelo</h1>
        <SearchBar onQueryChange={handleQueryChange} initialValue={initialQuery} />
        <UploadButton onOpen={() => setUploadOpen(true)} />
      </header>
      <main className="flex-1 p-4">
        <MemeGrid query={query} onSelect={handleSelect} prependMeme={prependMeme} updatedMeme={updatedMeme} />
      </main>
      {uploadOpen && (
        <UploadModal
          onClose={() => setUploadOpen(false)}
          onUploaded={handleUploaded}
        />
      )}
      {selected && (
        <MemeModal
          meme={selected}
          onClose={() => {
            setSelected(null)
            const params = new URLSearchParams()
            if (query) params.set('q', query)
            window.history.pushState({ query }, '', `/media${params.size ? '?' + params : ''}`)
          }}
          onPrev={selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex < memesList.length - 1 ? handleNext : undefined}
          onUpdate={handleUpdate}
          isEdited={selected.edited}
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
