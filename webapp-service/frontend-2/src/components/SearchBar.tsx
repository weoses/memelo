import { useState, useEffect } from 'react'

interface Props {
  onQueryChange: (q: string) => void
  initialValue?: string
}

export default function SearchBar({ onQueryChange, initialValue }: Props) {
  const [value, setValue] = useState(initialValue ?? '')

  useEffect(() => {
    const t = setTimeout(() => onQueryChange(value), 400)
    return () => clearTimeout(t)
  }, [value, onQueryChange])

  return (
    <div className="relative flex-1">
      <svg
        className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 pointer-events-none"
        width="15" height="15" viewBox="0 0 20 20" fill="none"
      >
        <circle cx="8.5" cy="8.5" r="5.5" stroke="currentColor" strokeWidth="1.8"/>
        <path d="M13 13l3.5 3.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/>
      </svg>
      <input
        type="search"
        value={value}
        onChange={e => setValue(e.target.value)}
        placeholder="Search memes, OCR text, or tags..."
        className="w-full rounded-lg bg-[#222] border border-white/[0.08] pl-9 pr-3 py-2 text-sm text-white
                   placeholder-gray-500 outline-none focus:border-white/20 transition-colors"
      />
    </div>
  )
}
