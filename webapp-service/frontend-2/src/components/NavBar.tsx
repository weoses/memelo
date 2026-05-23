import SearchBar from './SearchBar'

type Tab = 'gallery' | 'statistics' | 'collections' | 'favorites'

interface Props {
  activeTab: Tab
  onTabChange: (tab: Tab) => void
  query: string
  onQueryChange: (q: string) => void
  onUploadClick: () => void
}

const TABS: { key: Tab; label: string }[] = [
  { key: 'gallery', label: 'Gallery' },
  // { key: 'statistics', label: 'Statistics' },
  // { key: 'collections', label: 'Collections' },
  // { key: 'favorites', label: 'Favorites' },
]

export default function NavBar({ activeTab, onTabChange, query, onQueryChange, onUploadClick }: Props) {
  return (
    <header className="sticky top-0 z-40 bg-[#141414] border-b border-white/[0.08]">
      {/* Main row */}
      <div className="flex items-center gap-3 px-4 py-3">
        <span className="text-base font-bold text-white shrink-0 tracking-tight">Meme Vault</span>

        {/* Desktop: tabs + search + upload */}
        <nav className="hidden sm:flex items-center gap-1 ml-2 shrink-0">
          {TABS.map(t => (
            <button
              key={t.key}
              onClick={() => onTabChange(t.key)}
              className={`px-3 py-1.5 text-sm font-medium transition-colors border-b-2
                ${activeTab === t.key
                  ? 'text-white border-white'
                  : 'text-gray-500 border-transparent hover:text-gray-300'}`}
            >
              {t.label}
            </button>
          ))}
        </nav>

        <div className="hidden sm:flex flex-1 items-center gap-2">
          <SearchBar onQueryChange={onQueryChange} initialValue={query} />
          <button
            onClick={onUploadClick}
            className="shrink-0 w-8 h-8 rounded-full bg-white text-black flex items-center justify-center
                       hover:bg-gray-100 transition-colors"
            aria-label="Upload"
          >
            <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
              <path d="M10 4v12M4 10h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
          </button>
          <button className="shrink-0 w-8 h-8 rounded-full bg-[#2a2a2a] border border-white/10
                             flex items-center justify-center text-gray-400 hover:border-white/25
                             hover:text-gray-300 transition-colors">
            <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="7" r="3.5" stroke="currentColor" strokeWidth="1.6"/>
              <path d="M3 17c0-3.314 3.134-6 7-6s7 2.686 7 6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
            </svg>
          </button>
        </div>

        {/* Mobile: plus + avatar */}
        <div className="flex sm:hidden flex-1 justify-end items-center gap-2">
          <button
            onClick={onUploadClick}
            className="w-8 h-8 rounded-full bg-white text-black flex items-center justify-center
                       hover:bg-gray-100 transition-colors"
            aria-label="Upload"
          >
            <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
              <path d="M10 4v12M4 10h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
          </button>
          <button className="w-8 h-8 rounded-full bg-[#2a2a2a] border border-white/10 flex items-center justify-center text-gray-400">
            <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="7" r="3.5" stroke="currentColor" strokeWidth="1.6"/>
              <path d="M3 17c0-3.314 3.134-6 7-6s7 2.686 7 6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
            </svg>
          </button>
        </div>
      </div>

      {/* Mobile search row */}
      <div className="sm:hidden px-4 pb-3">
        <SearchBar onQueryChange={onQueryChange} initialValue={query} />
      </div>
    </header>
  )
}
