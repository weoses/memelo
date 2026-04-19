import { useState, useCallback } from 'react'
import SearchBar from './components/SearchBar'
import MemeGrid from './components/MemeGrid'
import UploadButton from './components/UploadButton'
import MemeModal from './components/MemeModal'
import { Meme } from './api/client'

export default function App() {
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Meme | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)

  const handleQueryChange = useCallback((q: string) => setQuery(q), [])
  const handleUploaded = useCallback(() => setRefreshKey(k => k + 1), [])

  return (
    <div className="min-h-screen bg-gray-950 text-white flex flex-col">
      <header className="sticky top-0 z-10 bg-gray-900 border-b border-gray-800 px-4 py-3 flex items-center gap-3">
        <h1 className="text-lg font-bold text-purple-400 shrink-0">Memelo</h1>
        <SearchBar onQueryChange={handleQueryChange} />
        <UploadButton onUploaded={handleUploaded} />
      </header>
      <main className="flex-1 p-4">
        <MemeGrid query={query} onSelect={setSelected} refreshKey={refreshKey} />
      </main>
      {selected && <MemeModal meme={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
