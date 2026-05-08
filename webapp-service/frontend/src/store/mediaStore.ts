import { create } from 'zustand'
import { Meme, Pagination, searchMemes, getMeme } from '../api/client'

interface MediaState {
  memes: Meme[]
  nextPage: Pagination | null
  hasMore: boolean
  loading: boolean
  currentQuery: string
  detachedMeme: Meme | null
  _loadController: AbortController | null

  load: (query: string) => Promise<void>
  loadMore: () => Promise<void>
  getByIndex: (index: number) => Meme | undefined
  getById: (id: string) => Promise<Meme | null>
  prependMeme: (meme: Meme) => void
  updateMeme: (updated: Meme) => void
  removeMeme: (id: string) => void
  setDetachedMeme: (meme: Meme | null) => void
}

export const useMediaStore = create<MediaState>((set, get) => ({
  memes: [],
  nextPage: null,
  hasMore: true,
  loading: false,
  currentQuery: '',
  detachedMeme: null,
  _loadController: null,

  load: async (query: string) => {
    get()._loadController?.abort()
    const controller = new AbortController()
    set({ memes: [], nextPage: null, hasMore: true, loading: true, currentQuery: query, _loadController: controller })
    try {
      const resp = await searchMemes(query, null, 20, controller.signal)
      set({ memes: resp.memes, nextPage: resp.next_page, hasMore: resp.memes.length > 0 && resp.next_page !== null, loading: false, _loadController: null })
    } catch (e) {
      if ((e as Error).name === 'AbortError') return
      console.error('media load error', e)
      set({ loading: false, _loadController: null })
    }
  },

  loadMore: async () => {
    const { loading, hasMore, nextPage, currentQuery, memes } = get()
    if (loading || !hasMore) return
    set({ loading: true })
    try {
      const resp = await searchMemes(currentQuery, nextPage)
      set({
        memes: [...memes, ...resp.memes],
        nextPage: resp.next_page,
        hasMore: resp.memes.length > 0 && resp.next_page !== null,
        loading: false,
      })
    } catch (e) {
      console.error('media loadMore error', e)
      set({ loading: false })
    }
  },

  getByIndex: (index: number) => get().memes[index],

  getById: async (id: string) => {
    const found = get().memes.find(m => m.id === id)
    if (found) return found
    return getMeme(id)
  },

  prependMeme: (meme: Meme) => set(s => ({ memes: [meme, ...s.memes] })),

  updateMeme: (updated: Meme) =>
    set(s => ({ memes: s.memes.map(m => m.id === updated.id ? updated : m) })),

  removeMeme: (id: string) =>
    set(s => ({ memes: s.memes.filter(m => m.id !== id) })),

  setDetachedMeme: (meme: Meme | null) => set({ detachedMeme: meme }),
}))
