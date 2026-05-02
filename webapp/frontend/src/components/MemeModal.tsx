import { useEffect, useState, useMemo, useRef } from 'react'
import { Meme, updateMeme, updateMemeMedia, getUploadUrl, uploadToS3Post } from '../api/client'
import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  meme: Meme
  onClose: () => void
  onPrev?: () => void
  onNext?: () => void
  onUpdate: (meme: Meme) => void
  isEdited?: boolean
}

function computeMediaHeight(meme: Meme): number | null {
  const srcW = meme.original_w || meme.thumbnail_w
  const srcH = meme.original_h || meme.thumbnail_h
  if (!srcW || !srcH) return null
  const dialogW = Math.min(window.innerWidth - 64, 768)
  const maxH = window.innerHeight * 0.6
  return Math.round(Math.min(dialogW / (srcW / srcH), maxH))
}

type EditableField = "id" | 'caption' | 'on_screen_text' | 'audio_transcript' | 'audio_track'

const FIELD_LABELS: Record<EditableField, string> = {
  id: "Id",
  caption: 'Caption',
  on_screen_text: 'On-screen text',
  audio_transcript: 'Audio transcript',
  audio_track: 'Audio track',
}

function PencilIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 20 20" fill="none" className="shrink-0">
      <path d="M14.85 2.15a2.12 2.12 0 0 1 3 3L6.5 16.5l-4 1 1-4L14.85 2.15z"
        stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  )
}

interface FieldRowProps {
  label: string
  value: string
  fieldKey: EditableField
  onSave: (key: EditableField, value: string) => Promise<void>
  readOnly?: boolean
  copyable?: boolean
}

function FieldRow({ label, value, fieldKey, onSave, readOnly, copyable }: FieldRowProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const handleCopy = () => {
    navigator.clipboard.writeText(value).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }

  useEffect(() => {
    setDraft(value)
    setEditing(false)
  }, [value])

  useEffect(() => {
    if (editing) textareaRef.current?.focus()
  }, [editing])

  const handleSave = async () => {
    if (draft === value) { setEditing(false); return }
    setSaving(true)
    setError(null)
    try {
      await onSave(fieldKey, draft)
      setEditing(false)
    } catch {
      setError('Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">{label}</span>
        <div className="flex items-center gap-1">
          {copyable && (
            <button
              onClick={handleCopy}
              className="text-gray-500 hover:text-purple-400 transition-colors p-0.5"
              aria-label="Copy"
            >
              {copied ? (
                <svg width="14" height="14" viewBox="0 0 20 20" fill="none" className="shrink-0">
                  <path d="M4 10l5 5 7-7" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              ) : (
                <svg width="14" height="14" viewBox="0 0 20 20" fill="none" className="shrink-0">
                  <rect x="7" y="7" width="10" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.8"/>
                  <path d="M13 7V5a1.5 1.5 0 0 0-1.5-1.5h-7A1.5 1.5 0 0 0 3 5v7A1.5 1.5 0 0 0 4.5 13.5H7" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/>
                </svg>
              )}
            </button>
          )}
          {!readOnly && !editing && (
            <button
              onClick={() => { setDraft(value); setEditing(true) }}
              className="text-gray-500 hover:text-purple-400 transition-colors p-0.5"
              aria-label={`Edit ${label}`}
            >
              <PencilIcon />
            </button>
          )}
        </div>
      </div>
      {!readOnly && editing ? (
        <div className="space-y-1">
          <textarea
            ref={textareaRef}
            value={draft}
            onChange={e => setDraft(e.target.value)}
            rows={3}
            className="w-full bg-gray-800 border border-gray-600 focus:border-purple-500 rounded px-2 py-1
                       text-sm text-white resize-none outline-none"
          />
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button
              onClick={handleSave}
              disabled={saving}
              className="text-xs bg-purple-600 hover:bg-purple-700 disabled:opacity-50 text-white rounded px-2 py-1 transition-colors"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button
              onClick={() => setEditing(false)}
              className="text-xs text-gray-400 hover:text-white rounded px-2 py-1 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <p className={`text-sm font-mono break-all ${value ? 'text-gray-300' : 'text-gray-600 italic'}`}>
          {value || 'empty'}
        </p>
      )}
    </div>
  )
}

export default function MemeModal({ meme, onClose, onPrev, onNext, onUpdate, isEdited }: Props) {
  const isVideo = meme.type === 'video'
  const [loaded, setLoaded] = useState(isVideo)
  const [mediaUploading, setMediaUploading] = useState(false)
  const [mediaError, setMediaError] = useState<string | null>(null)
  const mediaHeight = useMemo(() => computeMediaHeight(meme), [meme.id]) // eslint-disable-line react-hooks/exhaustive-deps
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const replaceTargetRef = useRef<'original' | 'thumbnail'>('original')
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!dropdownOpen) return
    const handler = (e: MouseEvent) => {
      if (!dropdownRef.current?.contains(e.target as Node)) setDropdownOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [dropdownOpen])

  useEffect(() => {
    if (!isVideo) setLoaded(false)
  }, [meme.id, isVideo]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft') onPrev?.()
      if (e.key === 'ArrowRight') onNext?.()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onPrev, onNext])

  const handleFieldSave = async (key: EditableField, value: string) => {
    const updated = await updateMeme(meme.id, { [key]: value })
    onUpdate(updated)
  }

  const triggerReplace = (target: 'original' | 'thumbnail') => {
    replaceTargetRef.current = target
    setDropdownOpen(false)
    fileInputRef.current?.click()
  }

  const handleReplaceMedia = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setMediaUploading(true)
    setMediaError(null)
    try {
      const { upload_url, form_fields, token } = await getUploadUrl(file.name, file.type, file.size)
      await uploadToS3Post(upload_url, form_fields, file)
      const updated = await updateMemeMedia(meme.id, token, replaceTargetRef.current)
      onUpdate(updated)
    } catch {
      setMediaError('Media replacement failed')
    } finally {
      setMediaUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const mediaStyle = mediaHeight ? { height: mediaHeight } : {}

  return (
    <Modal onClose={onClose}>
      {onPrev && (
        <button
          onClick={e => { e.stopPropagation(); onPrev() }}
          className="fixed left-3 top-1/2 -translate-y-1/2 z-[60] flex items-center justify-center
                     w-11 h-11 rounded-full bg-black/60 hover:bg-purple-600/80 backdrop-blur-sm
                     text-white transition-all duration-150 hover:scale-110 focus:outline-none"
          aria-label="Previous"
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M13 4l-6 6 6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
      )}
      {onNext && (
        <button
          onClick={e => { e.stopPropagation(); onNext() }}
          className="fixed right-3 top-1/2 -translate-y-1/2 z-[60] flex items-center justify-center
                     w-11 h-11 rounded-full bg-black/60 hover:bg-purple-600/80 backdrop-blur-sm
                     text-white transition-all duration-150 hover:scale-110 focus:outline-none"
          aria-label="Next"
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M7 4l6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </button>
      )}
      <Dialog onClose={onClose} className="max-w-3xl w-full max-h-[90vh] overflow-y-auto">
        <div className="relative w-full bg-black flex items-center justify-center overflow-hidden" style={mediaStyle}>
          {!loaded && (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-900">
              <div className="w-8 h-8 border-2 border-purple-500 border-t-transparent rounded-full animate-spin" />
            </div>
          )}
          {isVideo ? (
            <video
              key={meme.id}
              src={meme.original_url}
              controls
              autoPlay
              className="w-full h-full object-contain"
            />
          ) : (
            <img
              key={meme.id}
              src={meme.original_url || meme.thumbnail_url}
              alt={meme.caption || 'meme'}
              className="w-full h-full object-contain"
              style={{ opacity: loaded ? 1 : 0 }}
              onLoad={() => setLoaded(true)}
            />
          )}
        </div>
        <div className="flex items-center justify-between px-3 py-1.5">
          <div>
            {isEdited && (
              <span className="text-xs bg-yellow-500/20 text-yellow-400 rounded px-1.5 py-0.5">Edited</span>
            )}
          </div>
          <div className="relative" ref={dropdownRef}>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,video/*"
              className="hidden"
              onChange={handleReplaceMedia}
            />
            <button
              onClick={() => setDropdownOpen(o => !o)}
              disabled={mediaUploading}
              className="text-xs bg-gray-700 hover:bg-purple-600 disabled:opacity-50 text-white rounded px-2 py-1 transition-colors flex items-center gap-1"
            >
              {mediaUploading ? 'Uploading…' : 'Replace media'}
              {!mediaUploading && (
                <svg width="10" height="10" viewBox="0 0 10 10" fill="currentColor">
                  <path d="M2 3.5l3 3 3-3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none"/>
                </svg>
              )}
            </button>
            {dropdownOpen && (
              <div className="absolute right-0 top-full mt-1 bg-gray-800 border border-gray-700 rounded shadow-lg z-10 min-w-max">
                <button
                  onClick={() => triggerReplace('original')}
                  className="block w-full text-left text-xs text-gray-200 hover:bg-gray-700 px-3 py-2 transition-colors"
                >
                  Replace media
                </button>
                <button
                  onClick={() => triggerReplace('thumbnail')}
                  className="block w-full text-left text-xs text-gray-200 hover:bg-gray-700 px-3 py-2 transition-colors"
                >
                  Replace thumbnail
                </button>
              </div>
            )}
          </div>
        </div>
        {mediaError && (
          <p className="text-xs text-red-400 px-4 pb-1">{mediaError}</p>
        )}

        <div className="p-4 space-y-4">
          <FieldRow
            key="id"
            label={FIELD_LABELS['id']}
            value={meme.id}
            fieldKey="id"
            onSave={handleFieldSave}
            readOnly
            copyable
          />
          {((['caption', 'on_screen_text', 'audio_transcript', 'audio_track'] as EditableField[])).map(key => (
            <FieldRow
              key={key}
              label={FIELD_LABELS[key]}
              value={meme[key] ?? ''}
              fieldKey={key}
              onSave={handleFieldSave}
            />
          ))}
          {meme.tags && meme.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {meme.tags.map(t => (
                <span key={t} className="text-xs bg-gray-700 rounded px-2 py-0.5 text-gray-300">
                  {t}
                </span>
              ))}
            </div>
          )}
        </div>
      </Dialog>
    </Modal>
  )
}
