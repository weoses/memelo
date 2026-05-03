export interface Meme {
  id: string
  caption: string
  type: string
  ocr_result: string
  on_screen_text: string
  audio_transcript: string
  audio_track: string
  tags: string[]
  thumbnail_url: string
  thumbnail_w: number
  thumbnail_h: number
  original_url: string
  original_w: number
  original_h: number
  sorting_id: number
  status: string
  edited: boolean
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

export interface UploadUrlResponse {
  upload_url: string
  form_fields: Record<string, string>
  token: string
}

export async function getUploadUrl(
  filename: string,
  mime: string,
  length: number,
): Promise<UploadUrlResponse> {
  const params = new URLSearchParams({ filename, mime, length: String(length) })
  const res = await fetch(`/api/memes/get-upload-url?${params}`)
  if (!res.ok) throw new Error(`get upload url failed: ${res.status}`)
  return res.json()
}

export function uploadToS3Post(
  url: string,
  formFields: Record<string, string>,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<void> {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    for (const [key, value] of Object.entries(formFields)) {
      form.append(key, value)
    }
    form.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.upload.onprogress = e => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve()
      } else {
        reject(new Error(`S3 upload failed: ${xhr.status}`))
      }
    }
    xhr.onerror = () => reject(new Error('network error'))
    xhr.open('POST', url)
    xhr.send(form)
  })
}

export async function parseByToken(token: string): Promise<Meme> {
  const res = await fetch('/api/memes/parse-by-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  if (!res.ok) throw new Error(`parse by token failed: ${res.status}`)
  return res.json()
}

export async function updateMeme(
  id: string,
  fields: { caption?: string; on_screen_text?: string; audio_transcript?: string; audio_track?: string },
): Promise<Meme> {
  const res = await fetch(`/api/memes/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(fields),
  })
  if (!res.ok) throw new Error(`update meme failed: ${res.status}`)
  return res.json()
}

export async function updateMemeMedia(
  id: string,
  token: string,
  field: 'original' | 'thumbnail',
): Promise<Meme> {
  const res = await fetch(`/api/memes/${id}/update-media/${field}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  if (!res.ok) throw new Error(`update meme media failed: ${res.status}`)
  return res.json()
}
