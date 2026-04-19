import { useRef, useState } from 'react'
import { uploadMeme } from '../api/client'

interface Props {
  onUploaded?: () => void
}

export default function UploadButton({ onUploaded }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [dragging, setDragging] = useState(false)

  const handleFile = async (file: File) => {
    setUploading(true)
    try {
      await uploadMeme(file)
      onUploaded?.()
    } catch (e) {
      alert(`Upload failed: ${e}`)
    } finally {
      setUploading(false)
    }
  }

  return (
    <>
      <button
        onClick={() => inputRef.current?.click()}
        disabled={uploading}
        className="shrink-0 rounded-lg bg-purple-600 hover:bg-purple-700 disabled:opacity-50
                   px-3 py-2 text-sm font-medium text-white transition-colors"
      >
        {uploading ? '…' : '+ Upload'}
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*,video/*"
        className="hidden"
        onChange={e => {
          const f = e.target.files?.[0]
          if (f) handleFile(f)
          e.target.value = ''
        }}
      />
      {/* Full-screen drag-and-drop target */}
      <div
        className={`fixed inset-0 z-40 flex items-center justify-center
                    bg-black/60 border-4 border-dashed border-purple-400
                    transition-opacity duration-150
                    ${dragging ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
        onDragOver={e => { e.preventDefault(); setDragging(true) }}
        onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setDragging(false) }}
        onDrop={e => {
          e.preventDefault()
          setDragging(false)
          const f = e.dataTransfer.files[0]
          if (f) handleFile(f)
        }}
      >
        <p className="text-2xl font-bold text-purple-300">Drop to upload</p>
      </div>
    </>
  )
}
