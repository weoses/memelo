import { useState } from 'react'
import { Meme, updateMeme } from '../api/client'
import Modal from './Modal'
import Dialog from './Dialog'

interface Props {
  meme: Meme
  onClose: () => void
  onSaved: (updated: Meme) => void
}

type EditableField = 'caption' | 'on_screen_text' | 'audio_transcript' | 'audio_track'

const FIELDS: { key: EditableField; label: string; placeholder: string }[] = [
  { key: 'caption', label: 'Caption', placeholder: 'Describe this meme…' },
  { key: 'on_screen_text', label: 'On-screen text', placeholder: 'Text visible in the image…' },
  { key: 'audio_transcript', label: 'Audio transcript', placeholder: 'Spoken words…' },
  { key: 'audio_track', label: 'Audio track', placeholder: 'Song or audio name…' },
]

export default function EditMetadataDialog({ meme, onClose, onSaved }: Props) {
  const [values, setValues] = useState<Record<EditableField, string>>({
    caption: meme.caption ?? '',
    on_screen_text: meme.on_screen_text ?? '',
    audio_transcript: meme.audio_transcript ?? '',
    audio_track: meme.audio_track ?? '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      const updated = await updateMeme(meme.id, values)
      onSaved(updated)
      onClose()
    } catch {
      setError('Failed to save. Please try again.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal onClose={onClose}>
      <Dialog title="Edit Metadata" onClose={onClose} className="w-full max-w-lg max-h-[90dvh] overflow-y-auto">
        <div className="p-5 space-y-5">
          {FIELDS.map(({ key, label, placeholder }) => (
            <div key={key} className="space-y-1.5">
              <label className="text-xs font-medium text-gray-500 uppercase tracking-wider">
                {label}
              </label>
              <textarea
                value={values[key]}
                onChange={e => setValues(v => ({ ...v, [key]: e.target.value }))}
                placeholder={placeholder}
                rows={key === 'caption' ? 2 : 3}
                className="w-full bg-[#1c1c1c] border border-white/10 focus:border-white/25 rounded-lg
                           px-3 py-2.5 text-sm text-white placeholder-gray-600 resize-none outline-none
                           transition-colors"
              />
            </div>
          ))}

          {error && <p className="text-xs text-red-400">{error}</p>}

          <div className="flex gap-3 pt-1">
            <button
              onClick={onClose}
              className="flex-1 py-2.5 rounded-lg border border-white/15 text-sm text-gray-300
                         hover:border-white/30 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex-1 py-2.5 rounded-lg bg-white text-black text-sm font-semibold
                         hover:bg-gray-100 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </Dialog>
    </Modal>
  )
}
