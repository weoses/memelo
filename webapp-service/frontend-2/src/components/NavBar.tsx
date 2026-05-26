import { useState, useRef, useEffect } from 'react'
import SearchBar from './SearchBar'

export type Tab = 'gallery' | 'statistics' | 'collections' | 'favorites'

interface Props {
  activeTab: Tab
  onTabChange: (tab: Tab) => void
  query: string
  onQueryChange: (q: string) => void
  onUploadClick: () => void
  onLogout: () => void
}

const TABS: { key: Tab; label: string }[] = [
  { key: 'gallery', label: 'Gallery' },
  // { key: 'statistics', label: 'Statistics' },
  // { key: 'collections', label: 'Collections' },
  // { key: 'favorites', label: 'Favorites' },
]

function UserIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none">
      <circle cx="10" cy="7" r="3.5" stroke="currentColor" strokeWidth="1.6"/>
      <path d="M3 17c0-3.314 3.134-6 7-6s7 2.686 7 6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
    </svg>
  )
}

export default function NavBar({ activeTab, onTabChange, query, onQueryChange, onUploadClick, onLogout }: Props) {
  const [desktopOpen, setDesktopOpen] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!desktopOpen) return
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDesktopOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [desktopOpen])

  return (
    <header className="sticky top-0 z-40 bg-[#141414] border-b border-white/[0.08]">
      {/* Main row */}
      <div className="flex items-center gap-3 px-4 py-3">
        <span className="text-base font-bold text-white shrink-0 tracking-tight">Meme Vault</span>

        {/* Desktop: tabs + search + upload + user */}
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

          {/* Desktop user icon + dropdown */}
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setDesktopOpen(v => !v)}
              className="shrink-0 w-8 h-8 rounded-full bg-[#2a2a2a] border border-white/10
                         flex items-center justify-center text-gray-400 hover:border-white/25
                         hover:text-gray-300 transition-colors"
              aria-label="User menu"
            >
              <UserIcon />
            </button>

            {desktopOpen && (
              <div className="absolute right-0 top-10 w-36 bg-[#1e1e1e] border border-white/10
                              rounded-xl shadow-xl overflow-hidden z-50">
                <button
                  onClick={() => { setDesktopOpen(false); onLogout() }}
                  className="w-full px-4 py-2.5 text-sm text-left text-gray-200 hover:bg-white/5 transition-colors"
                >
                  Logout
                </button>
              </div>
            )}
          </div>
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
          <button
            onClick={() => setMobileOpen(true)}
            className="w-8 h-8 rounded-full bg-[#2a2a2a] border border-white/10 flex items-center justify-center text-gray-400"
            aria-label="User menu"
          >
            <UserIcon />
          </button>
        </div>
      </div>

      {/* Mobile search row */}
      <div className="sm:hidden px-4 pb-3">
        <SearchBar onQueryChange={onQueryChange} initialValue={query} />
      </div>

      {/* Mobile drawer */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 sm:hidden">
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => setMobileOpen(false)}
          />
          {/* Panel */}
          <div className="absolute right-0 top-0 h-full w-56 bg-[#1a1a1a] border-l border-white/10
                          flex flex-col pt-14 shadow-2xl">
            <button
              onClick={() => { setMobileOpen(false); onLogout() }}
              className="w-full px-6 py-3.5 text-sm text-left text-gray-200 hover:bg-white/5 transition-colors"
            >
              Logout
            </button>
          </div>
        </div>
      )}
    </header>
  )
}
