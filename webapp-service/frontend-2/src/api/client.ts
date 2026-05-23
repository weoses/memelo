import { useAuthStore } from '../store/authStore'

const _cfg = (window as Window & { __MEMELO_CONFIG__?: { baseUrl?: string } }).__MEMELO_CONFIG__
const BASE = _cfg?.baseUrl ? _cfg.baseUrl.replace(/\/$/, '') + '/' : ''

async function apiFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const store = useAuthStore.getState()
  const token = store.accessToken

  const headers = new Headers(init.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(input, { ...init, headers })

  if (res.status === 401) {
    const refreshToken = store.refreshToken
    if (refreshToken) {
      try {
        const refreshRes = await fetch(`${BASE}api/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        })
        if (refreshRes.ok) {
          const data = await refreshRes.json() as { access_token: string }
          useAuthStore.getState().setAccessToken(data.access_token)
          headers.set('Authorization', `Bearer ${data.access_token}`)
          return fetch(input, { ...init, headers })
        }
      } catch {
        // fall through to logout
      }
    }
    useAuthStore.getState().logout()
    window.location.href = '/login'
  }

  return res
}

export async function apiLogin(username: string, password: string): Promise<void> {
  const res = await fetch(`${BASE}api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) throw new Error(`login failed: ${res.status}`)
  const data = await res.json() as { access_token: string; refresh_token: string }
  useAuthStore.getState().setTokens(data.access_token, data.refresh_token)
}

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
  signal?: AbortSignal,
): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: query, limit: String(limit) })
  if (page) {
    params.set('after_searcher', page.searcher)
    page.sorting_after.forEach(v => params.append('after_sorting', v))
  }
  const res = await apiFetch(`${BASE}api/memes?${params}`, { signal })
  if (!res.ok) throw new Error(`search failed: ${res.status}`)
  return res.json()
}

export interface UploadUrlResponse {
  upload_url: string
  form_fields: Record<string, string>
  token: string
}

export async function getUploadUrl(
  mime: string,
  length: number,
): Promise<UploadUrlResponse> {
  const res = await apiFetch(`${BASE}api/memes/get-upload-url`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mime, length }),
  })
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
  const res = await apiFetch(`${BASE}api/memes/parse-by-token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  if (!res.ok) throw new Error(`parse by token failed: ${res.status}`)
  return res.json()
}

export async function getMeme(id: string): Promise<Meme | null> {
  const res = await apiFetch(`${BASE}api/memes/${id}`)
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`get meme failed: ${res.status}`)
  return res.json()
}

export async function updateMeme(
  id: string,
  fields: { caption?: string; on_screen_text?: string; audio_transcript?: string; audio_track?: string },
): Promise<Meme> {
  const res = await apiFetch(`${BASE}api/memes/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(fields),
  })
  if (!res.ok) throw new Error(`update meme failed: ${res.status}`)
  return res.json()
}

export async function deleteMeme(id: string): Promise<void> {
  const res = await apiFetch(`${BASE}api/memes/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`delete meme failed: ${res.status}`)
}

export async function recomputeMeme(id: string): Promise<Meme> {
  const res = await apiFetch(`${BASE}api/memes/${id}/recompute`, { method: 'POST' })
  if (!res.ok) throw new Error(`recompute meme failed: ${res.status}`)
  return res.json()
}

export async function updateMemeMedia(
  id: string,
  token: string,
  field: 'original' | 'thumbnail',
): Promise<Meme> {
  const res = await apiFetch(`${BASE}api/memes/${id}/update-media/${field}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  if (!res.ok) throw new Error(`update meme media failed: ${res.status}`)
  return res.json()
}
