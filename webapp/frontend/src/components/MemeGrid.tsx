import { useState, useEffect, useRef, useCallback } from 'react'
import { searchMemes, Meme, Pagination } from '../api/client'
import MemeCard from './MemeCard'

interface Props {
  query: string
  onSelect: (m: Meme) => void
  refreshKey?: number
}

export default function MemeGrid({ query, onSelect, refreshKey }: Props) {
  const [memes, setMemes] = useState<Meme[]>([])
  const [nextPage, setNextPage] = useState<Pagination | null>(null)
  const [loading, setLoading] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const loadingRef = useRef(false)
  const resetRef = useRef(true)

  // Reset state when query or refreshKey changes
  useEffect(() => {
    setMemes([])
    setNextPage(null)
    setHasMore(true)
    resetRef.current = true
  }, [query, refreshKey])

  const loadMore = useCallback(async (page: Pagination | null, isReset: boolean) => {
    if (loadingRef.current) return
    loadingRef.current = true
    setLoading(true)
    try {
      const resp = await searchMemes(query, page)
      setMemes(prev => isReset ? resp.memes : [...prev, ...resp.memes])
      setNextPage(resp.next_page)
      setHasMore(resp.memes.length > 0 && resp.next_page !== null)
    } catch (e) {
      console.error('search error', e)
    } finally {
      loadingRef.current = false
      setLoading(false)
    }
  }, [query])

  // Trigger initial load after reset
  useEffect(() => {
    if (resetRef.current) {
      resetRef.current = false
      loadMore(null, true)
    }
  }, [query, refreshKey, loadMore])

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const obs = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting && hasMore && !loadingRef.current) {
        setNextPage(page => {
          loadMore(page, false)
          return page
        })
      }
    }, { rootMargin: '300px' })
    obs.observe(el)
    return () => obs.disconnect()
  }, [hasMore, loadMore])

  if (!loading && memes.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-gray-500">
        {query ? 'No memes found.' : 'No memes yet. Upload one!'}
      </div>
    )
  }

  return (
    <div>
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2">
        {memes.map(m => (
          <MemeCard key={m.id} meme={m} onClick={() => onSelect(m)} />
        ))}
      </div>
      {loading && (
        <div className="flex justify-center py-8">
          <div className="w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full animate-spin" />
        </div>
      )}
      <div ref={sentinelRef} className="h-1" />
    </div>
  )
}
