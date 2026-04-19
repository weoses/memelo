export interface Meme {
  id: string
  caption: string
  type: string
  ocr_result: string
  tags: string[]
  thumbnail_url: string
  thumbnail_w: number
  thumbnail_h: number
  original_url: string
  original_w: number
  original_h: number
  sorting_id: number
}

export interface Pagination {
  searcher: string
  sorting_after: string[]
}

export interface SearchResponse {
  memes: Meme[]
  next_page: Pagination | null
}

export async function searchMemes(
  query: string,
  page: Pagination | null,
  limit = 20,
): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  if (page) {
    params.set('after_searcher', page.searcher)
    page.sorting_after.forEach(v => params.append('after_sorting', v))
  }
  const res = await fetch(`/api/memes?${params}`)
  if (!res.ok) throw new Error(`search failed: ${res.status}`)
  return res.json()
}

export async function uploadMeme(file: File): Promise<Meme> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch('/api/memes', { method: 'POST', body: form })
  if (!res.ok) throw new Error(`upload failed: ${res.status}`)
  return res.json()
}
