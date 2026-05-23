type MobileTab = 'gallery' | 'upload'

interface Props {
  activeTab: MobileTab
  onTabChange: (tab: MobileTab) => void
}

export default function BottomNav({ activeTab, onTabChange }: Props) {
  return (
    <nav className="sm:hidden fixed bottom-0 inset-x-0 z-40 bg-[#141414] border-t border-white/[0.08] flex">
      <button
        onClick={() => onTabChange('gallery')}
        className={`flex-1 flex flex-col items-center justify-center gap-1 py-3 text-xs font-medium transition-colors
          ${activeTab === 'gallery' ? 'text-white' : 'text-gray-500'}`}
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <rect x="3" y="3" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.7"/>
          <rect x="14" y="3" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.7"/>
          <rect x="3" y="14" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.7"/>
          <rect x="14" y="14" width="7" height="7" rx="1.5" stroke="currentColor" strokeWidth="1.7"/>
        </svg>
        Gallery
      </button>
      <button
        onClick={() => onTabChange('upload')}
        className={`flex-1 flex flex-col items-center justify-center gap-1 py-3 text-xs font-medium transition-colors
          ${activeTab === 'upload' ? 'text-white' : 'text-gray-500'}`}
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <path d="M12 16V4m0 0L8 8m4-4 4 4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"/>
          <path d="M4 20h16" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round"/>
        </svg>
        Upload
      </button>
    </nav>
  )
}
