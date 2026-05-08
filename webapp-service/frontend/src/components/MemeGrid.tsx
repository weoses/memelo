import { useState, useEffect, useRef, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { Meme } from '../api/client'
import { useMediaStore } from '../store/mediaStore'
import MemeCard from './MemeCard'

const ROW_HEIGHT = 220
const GUTTER = 4

interface Props {
  query: string
  onSelect: (index: number) => void
  onRecompute: (meme: Meme) => void
  onDownload: (meme: Meme) => void
  onDelete: (meme: Meme) => void
}

interface RowItem { meme: Meme; w: number; h: number; index: number }

function buildRows(memes: Meme[], containerWidth: number): RowItem[][] {
  const rowHeight = containerWidth < 480 ? 140 : containerWidth < 768 ? 170 : ROW_HEIGHT
  const maxFraction = containerWidth < 480 ? 0.30 : containerWidth < 768 ? 0.34 : 0.25
  const maxItemW = Math.floor(containerWidth * maxFraction)
  const clamp = (m: Meme) => Math.min(Math.round((m.thumbnail_w && m.thumbnail_h ? m.thumbnail_w / m.thumbnail_h : 1) * rowHeight), maxItemW)

  const rows: RowItem[][] = []
  let bucket: { m: Meme; flatIdx: number }[] = []
  let bucketW = 0

  const flush = (last = false) => {
    if (!bucket.length) return
    if (last) {
      rows.push(bucket.map(({ m, flatIdx }) => ({ meme: m, w: clamp(m), h: rowHeight, index: flatIdx })))
    } else {
      const gutters = (bucket.length - 1) * GUTTER
      const totalNatural = bucket.reduce((s, { m }) => s + clamp(m), 0)
      const scale = (containerWidth - gutters) / totalNatural
      const h = Math.round(rowHeight * scale)
      let remaining = containerWidth - gutters
      rows.push(bucket.map(({ m, flatIdx }, i) => {
        const w = i === bucket.length - 1 ? remaining : Math.round(clamp(m) * scale)
        remaining -= w
        return { meme: m, w, h, index: flatIdx }
      }))
    }
    bucket = []
    bucketW = 0
  }

  let flatIdx = 0
  for (const m of memes) {
    const nw = clamp(m)
    const needed = bucketW + (bucket.length > 0 ? GUTTER : 0) + nw
    if (bucket.length > 0 && needed > containerWidth) flush()
    bucketW += (bucket.length > 0 ? GUTTER : 0) + nw
    bucket.push({ m, flatIdx })
    flatIdx++
  }
  flush(true)

  return rows
}

export default function MemeGrid({ query, onSelect, onRecompute, onDownload, onDelete }: Props) {
  const { memes, loading, load, loadMore } = useMediaStore(
    useShallow(s => ({ memes: s.memes, loading: s.loading, load: s.load, loadMore: s.loadMore }))
  )
  const sentinelRef = useRef<HTMLDivElement>(null)
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
    load(query)
  }, [query, load])

  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const obs = new IntersectionObserver(entries => {
      if (entries[0].isIntersecting) void loadMore()
    }, { rootMargin: '300px' })
    obs.observe(el)
    return () => obs.disconnect()
  }, [loadMore, memes.length])

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
              {row.map(({ meme, w, h, index }) => (
                <div key={meme.id} style={{ width: w, height: h, flexShrink: 0 }}>
                  <MemeCard
                    meme={meme}
                    onClick={() => onSelect(index)}
                    isEdited={meme.edited}
                    onRecompute={() => onRecompute(meme)}
                    onDownload={() => onDownload(meme)}
                    onDelete={() => onDelete(meme)}
                  />
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
