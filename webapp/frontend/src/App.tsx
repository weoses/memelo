import { useState, useCallback } from 'react'
import SearchBar from './components/SearchBar'
import MemeGrid from './components/MemeGrid'
import UploadButton from './components/UploadButton'
import UploadModal from './components/UploadModal'
import MemeModal from './components/MemeModal'
import { Meme } from './api/client'

export default function App() {
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Meme | null>(null)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [memesList, setMemesList] = useState<Meme[]>([])
  const [uploadOpen, setUploadOpen] = useState(false)
  const [prependMeme, setPrependMeme] = useState<Meme | null>(null)

  const handleQueryChange = useCallback((q: string) => setQuery(q), [])
  const handleUploaded = useCallback((meme: Meme) => setPrependMeme(meme), [])

  const handleSelect = useCallback((m: Meme, index: number, list: Meme[]) => {
    setSelected(m)
    setSelectedIndex(index)
    setMemesList(list)
  }, [])

  const handlePrev = useCallback(() => {
    if (selectedIndex > 0) {
      setSelectedIndex(i => i - 1)
      setSelected(memesList[selectedIndex - 1])
    }
  }, [selectedIndex, memesList])

  const handleNext = useCallback(() => {
    if (selectedIndex < memesList.length - 1) {
      setSelectedIndex(i => i + 1)
      setSelected(memesList[selectedIndex + 1])
    }
  }, [selectedIndex, memesList])

  return (
    <div className="min-h-screen bg-gray-950 text-white flex flex-col">
      <header className="sticky top-0 z-10 bg-gray-900 border-b border-gray-800 px-4 py-3 flex items-center gap-3">
        <h1 className="text-lg font-bold text-purple-400 shrink-0">Memelo</h1>
        <SearchBar onQueryChange={handleQueryChange} />
        <UploadButton onOpen={() => setUploadOpen(true)} />
      </header>
      <main className="flex-1 p-4">
        <MemeGrid query={query} onSelect={handleSelect} prependMeme={prependMeme} />
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
          onClose={() => setSelected(null)}
          onPrev={selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex < memesList.length - 1 ? handleNext : undefined}
        />
      )}
    </div>
  )
}