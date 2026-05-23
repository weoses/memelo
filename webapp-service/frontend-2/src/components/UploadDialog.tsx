import { useRef, useState, DragEvent } from 'react'
import { getUploadUrl, uploadToS3Post, parseByToken, Meme } from '../api/client'
import Modal from './Modal'
import Dialog from './Dialog'

type Phase = 'idle' | 'uploading' | 'processing' | 'success' | 'duplicate' | 'error'

interface Props {
  onClose: () => void
  onUploaded: (meme: Meme) => void
}

export default function UploadDialog({ onClose, onUploaded }: Props) {
  const [phase, setPhase] = useState<Phase>('idle')
  const [progress, setProgress] = useState(0)
  const [errorMsg, setErrorMsg] = useState('')
  const [dragging, setDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const canClose = phase !== 'uploading' && phase !== 'processing'

  const startUpload = async (file: File) => {
    setPhase('uploading')
    setProgress(0)
    try {
      const { upload_url, form_fields, token } = await getUploadUrl(file.type, file.size)
      await uploadToS3Post(upload_url, form_fields, file, setProgress)
      setPhase('processing')
      const meme = await parseByToken(token)
      if (meme.status === 'duplicate') {
        setPhase('duplicate')
      } else {
        setPhase('success')
        onUploaded(meme)
        setTimeout(onClose, 1500)
      }
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : String(e))
      setPhase('error')
    }
  }

  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) void startUpload(file)
  }

  return (
    <Modal onClose={onClose} canClose={canClose}>
      <Dialog title="New Meme" onClose={onClose} canClose={canClose} className="w-full max-w-md overflow-hidden">
        <div className="p-5 space-y-4">

          {phase === 'idle' && (
            <>
              <div
                onDragOver={e => { e.preventDefault(); setDragging(true) }}
                onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setDragging(false) }}
                onDrop={onDrop}
                onClick={() => inputRef.current?.click()}
                className={`flex flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed
                            cursor-pointer py-14 transition-colors
                            ${dragging
                              ? 'border-white/40 bg-white/5'
                              : 'border-white/15 hover:border-white/30 hover:bg-white/[0.03]'}`}
              >
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" className="text-gray-400">
                  <path d="M12 16V4m0 0L8 8m4-4 4 4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"/>
                  <rect x="3" y="18" width="18" height="2" rx="1" fill="currentColor" opacity="0.4"/>
                </svg>
                <div className="text-center">
                  <p className="text-sm font-medium text-white">Select or drop file</p>
                  <p className="text-xs text-gray-500 mt-1">JPG, PNG, GIF or MP4</p>
                </div>
                <input
                  ref={inputRef}
                  type="file"
                  accept="image/*,video/*"
                  className="hidden"
                  onChange={e => {
                    const f = e.target.files?.[0]
                    if (f) void startUpload(f)
                    e.target.value = ''
                  }}
                />
              </div>

              <button
                onClick={() => inputRef.current?.click()}
                className="w-full bg-white text-black text-sm font-semibold py-3 rounded-full
                           hover:bg-gray-100 transition-colors"
              >
                Upload to Vault
              </button>

              <p className="text-xs text-gray-600 text-center">
                Uploaded media will be automatically indexed for OCR search.
              </p>
            </>
          )}

          {phase === 'uploading' && (
            <div className="flex flex-col items-center gap-4 py-8">
              <p className="text-sm text-gray-300 font-medium">Uploading… {progress}%</p>
              <div className="w-full bg-white/10 rounded-full h-2 overflow-hidden">
                <div
                  className="h-full bg-white rounded-full transition-all duration-200"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}

          {phase === 'processing' && (
            <div className="flex items-center gap-3 py-8">
              <div className="w-5 h-5 rounded-full border-2 border-white/30 border-t-white animate-spin shrink-0" />
              <p className="text-sm text-gray-300">Processing… This may take a moment.</p>
            </div>
          )}

          {phase === 'success' && (
            <div className="flex flex-col items-center gap-3 py-10">
              <div className="w-12 h-12 rounded-full bg-white/10 flex items-center justify-center">
                <svg width="22" height="22" viewBox="0 0 20 20" fill="none">
                  <path d="M4 10l5 5 7-7" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <p className="text-sm text-white font-medium">Upload complete!</p>
            </div>
          )}

          {phase === 'duplicate' && (
            <div className="flex flex-col items-center gap-3 py-10">
              <div className="w-12 h-12 rounded-full bg-white/10 flex items-center justify-center text-xl">
                ⊘
              </div>
              <p className="text-sm text-white font-medium">Already in your vault</p>
              <p className="text-xs text-gray-500">This meme is a duplicate of one you already have.</p>
            </div>
          )}

          {phase === 'error' && (
            <div className="flex flex-col items-center gap-4 py-8">
              <div className="w-12 h-12 rounded-full bg-red-500/10 flex items-center justify-center">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                  <path d="M5 5l10 10M15 5L5 15" stroke="#ef4444" strokeWidth="2" strokeLinecap="round"/>
                </svg>
              </div>
              <p className="text-sm text-white font-medium">Upload failed</p>
              <p className="text-xs text-gray-500 text-center">{errorMsg}</p>
              <button
                onClick={() => setPhase('idle')}
                className="mt-1 w-full bg-white text-black text-sm font-semibold py-2.5 rounded-full hover:bg-gray-100 transition-colors"
              >
                Try again
              </button>
            </div>
          )}
        </div>
      </Dialog>
    </Modal>
  )
}
