import { useState, useEffect } from 'react'

interface Props {
  onQueryChange: (q: string) => void
}

export default function SearchBar({ onQueryChange }: Props) {
  const [value, setValue] = useState('')

  useEffect(() => {
    const t = setTimeout(() => onQueryChange(value), 400)
    return () => clearTimeout(t)
  }, [value, onQueryChange])

  return (
    <input
      type="search"
      value={value}
      onChange={e => setValue(e.target.value)}
      placeholder="Search memes..."
      className="flex-1 rounded-lg bg-gray-800 px-3 py-2 text-sm text-white
                 placeholder-gray-500 outline-none focus:ring-2 focus:ring-purple-500"
    />
  )
}
