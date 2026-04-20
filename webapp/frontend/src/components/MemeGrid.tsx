import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { searchMemes, Meme, Pagination } from '../api/client'
import MemeCard from './MemeCard'

const ROW_HEIGHT = 220
const GUTTER = 4

interface Props {
  query: string
  onSelect: (m: Meme, index: number, list: Meme[]) => void
  prependMeme?: Meme | null
}

interface RowItem { meme: Meme; w: number; h: number }

function naturalWidth(m: Meme): number {
  const ratio = m.thumbnail_w && m.thumbnail_h ? m.thumbnail_w / m.thumbnail_h : 1
  return Math.round(ratio * ROW_HEIGHT)
}

function buildRows(memes: Meme[], containerWidth: number): RowItem[][] {
  const maxItemW = Math.floor(containerWidth * 0.25)
  const clamp = (m: Meme) => Math.min(naturalWidth(m), maxItemW)

  const rows: RowItem[][] = []
  let bucket: Meme[] = []
  let bucketW = 0

  const flush = () => {
    if (!bucket.length) return
    const gutters = (bucket.length - 1) * GUTTER
    const totalNatural = bucket.reduce((s, m) => s + clamp(m), 0)
    const scale = (containerWidth - gutters) / totalNatural
    const h = Math.round(ROW_HEIGHT * scale)
    let remaining = containerWidth - gutters
    rows.push(bucket.map((m, i) => {
      const w = i === bucket.length - 1 ? remaining : Math.round(clamp(m) * scale)
      remaining -= w
      return { meme: m, w, h }
    }))
    bucket = []
    bucketW = 0
  }

  for (const m of memes) {
    const nw = clamp(m)
    const needed = bucketW + (bucket.length > 0 ? GUTTER : 0) + nw
    if (bucket.length > 0 && needed > containerWidth) flush()
    bucketW += (bucket.length > 0 ? GUTTER : 0) + nw
    bucket.push(m)
  }
  flush()

  return rows
}

export default function MemeGrid({ query, onSelect, prependMeme }: Props) {
  const [memes, setMemes] = useState<Meme[]>([])
  const [nextPage, setNextPage] = useState<Pagination | null>(null)
  const [loading, setLoading] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const sentinelRef = useRef<HTMLDivElement>(null)
  const loadingRef = useRef(false)
  const resetRef = useRef(true)
  const containerRef = useRef<HTMLDivElement>(null)
  const [containerWidth, setContainerWidth] = useState(0)

  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const ro = new ResizeObserver(entries => setContainerWidth(entries[0].contentRect.width))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  useEffect(() => {
    setMemes([])
    setNextPage(null)
    setHasMore(true)
    resetRef.current = true
  }, [query])

  useEffect(() => {
    if (prependMeme) setMemes(prev => [prependMeme, ...prev])
  }, [prependMeme])

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

  useEffect(() => {
    if (resetRef.current) {
      resetRef.current = false
      void loadMore(null, true)
    }
  }, [query, loadMore])

  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const obs = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting && hasMore && !loadingRef.current) {
        void loadMore(nextPage, false)
      }
    }, { rootMargin: '300px' })
    obs.observe(el)
    return () => obs.disconnect()
  }, [hasMore, loadMore, nextPage])

  const rows = useMemo(
    () => containerWidth > 0 ? buildRows(memes, containerWidth) : [],
    [memes, containerWidth]
  )

  return (
    <div ref={containerRef}>
      {!loading && memes.length === 0 ? (
        <div className="flex items-center justify-center h-64 text-gray-500">
          {query ? 'No memes found.' : 'No memes yet. Upload one!'}
        </div>
      ) : (
        <div className="flex flex-col" style={{ gap: GUTTER }}>
          {rows.map((row, rowIdx) => (
            <div key={rowIdx} className="flex" style={{ gap: GUTTER, height: row[0]?.h }}>
              {row.map(({ meme, w, h }) => (
                <div key={meme.id} style={{ width: w, height: h, flexShrink: 0 }}>
                  <MemeCard meme={meme} onClick={() => onSelect(meme, memes.indexOf(meme), memes)} />
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
      {loading && (
        <div className="flex justify-center py-8">
          <div className="w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full animate-spin" />
        </div>
      )}
      <div ref={sentinelRef} className="h-1" />
    </div>
  )
}