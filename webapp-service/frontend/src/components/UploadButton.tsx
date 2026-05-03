interface Props {
  onOpen: () => void
}

export default function UploadButton({ onOpen }: Props) {
  return (
    <button
      onClick={onOpen}
      className="shrink-0 rounded-lg bg-purple-600 hover:bg-purple-700
                 px-3 py-2 text-sm font-medium text-white transition-colors flex items-center justify-center"
    >
      <svg className="sm:hidden w-4 h-4" viewBox="0 0 20 20" fill="none">
        <path d="M10 3v10M5 8l5-5 5 5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
        <path d="M3 15h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
      </svg>
      <span className="hidden sm:inline">+ Upload</span>
    </button>
  )
}